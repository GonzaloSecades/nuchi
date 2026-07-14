// Package auth implements the cryptographic primitives for password auth
// and JWT sessions: Argon2id password hashing (this file), opaque
// refresh/verification token generation (token.go), and JWT access-token
// issuance/verification (jwt.go). It has no HTTP or database dependency;
// callers in internal/http wire it to request handling and internal/db/gen
// wire it to storage.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. Chosen for a single-tenant dev-scale API: ~64 MiB
// memory and 3 iterations is comfortably above OWASP's minimum
// recommendation while staying fast enough for interactive login on modest
// hardware. Parallelism 2 matches typical small-instance CPU counts.
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // KiB (64 MiB)
	argonThreads uint8  = 2
	argonSaltLen        = 16
	argonKeyLen  uint32 = 32
)

// ErrMalformedHash is returned by VerifyPassword when the stored hash is not
// a well-formed Argon2id PHC string. Treated as a verification failure by
// callers, never as a reason to leak information about hash internals.
var ErrMalformedHash = errors.New("auth: malformed password hash")

// HashPassword derives an Argon2id hash for password and encodes it as a
// standard PHC string: $argon2id$v=19$m=<mem>,t=<time>,p=<threads>$<salt>$<hash>.
// Encoding the parameters alongside the hash (rather than assuming fixed
// package constants) keeps future rehash-on-login parameter bumps possible
// without a data migration.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, encodedSalt, encodedKey,
	), nil
}

// VerifyPassword reports whether password matches the Argon2id PHC string
// encodedHash, re-deriving the key with the parameters embedded in the hash
// (not the package's current constants) so a future parameter change cannot
// break verification of existing hashes. Comparison is constant-time.
// A malformed encodedHash is treated as a non-match rather than a hard
// error, so callers can use the same code path for corrupt stored data as
// for a wrong password.
func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, want, err := decodeHash(encodedHash)
	if err != nil {
		return false, nil
	}

	got := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type argonParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

// decodeHash parses a PHC-formatted Argon2id string into its parameters,
// salt, and derived key. It returns ErrMalformedHash for anything that
// doesn't match the expected shape rather than panicking on attacker- or
// corruption-controlled input.
func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// Split("$argon2id$v=19$m=...,t=...,p=...$salt$hash", "$") yields
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", "salt", "hash"].
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	if version != argon2.Version {
		return argonParams{}, nil, nil, ErrMalformedHash
	}

	var params argonParams
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &threads); err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	params.threads = uint8(threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}

	return params, salt, key, nil
}

// dummyPasswordHash is a fixed, precomputed-at-init Argon2id hash used by
// DummyVerify. It is never a real user's hash.
var dummyPasswordHash = mustHashPassword("dummy-password-for-timing-safety-do-not-use")

func mustHashPassword(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		panic(fmt.Errorf("auth: failed to compute dummy password hash: %w", err))
	}
	return hash
}

// DummyVerify performs a full Argon2id verification against a fixed
// reference hash and discards the result. Callers use it on a login attempt
// for an email that does not exist, so that responding "unauthorized" costs
// the same wall-clock time whether the email is unknown or the password was
// simply wrong — otherwise the response latency itself would let an
// attacker enumerate registered emails.
func DummyVerify(password string) {
	_, _ = VerifyPassword(password, dummyPasswordHash)
}
