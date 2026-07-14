package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// tokenBytes is the amount of entropy in a generated opaque token: 256 bits,
// matching the design decision recorded on #41.
const tokenBytes = 32

// GenerateToken creates a fresh 256-bit random opaque token. It returns the
// raw value (base64url, unpadded) to hand to the client — e.g. as a cookie
// value — and the SHA-256 hex digest of that raw value, which is the only
// form ever persisted (refresh_tokens.token_hash and friends). Storing only
// the hash means a database read can never recover a usable token.
func GenerateToken() (raw string, hash string, err error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("auth: generate token: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken returns the SHA-256 hex digest of a raw token value, the same
// transform GenerateToken applies and the form used to look up stored
// tokens by value (e.g. hashing an incoming refresh cookie before querying
// refresh_tokens.token_hash).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
