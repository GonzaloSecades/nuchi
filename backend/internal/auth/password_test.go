package auth

import (
	"fmt"
	"strings"
	"testing"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword: expected the original password to verify")
	}
}

func TestHashPassword_EncodesPHCString(t *testing.T) {
	hash, err := HashPassword("some-password")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("HashPassword: expected PHC-encoded argon2id string with the configured params, got %q", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("HashPassword: expected 6 '$'-delimited segments, got %d: %q", len(parts), hash)
	}
}

func TestHashPassword_UniqueSaltPerCall(t *testing.T) {
	hash1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}
	hash2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("HashPassword: expected two hashes of the same password to differ (random salt)")
	}
}

func TestVerifyPassword_WrongPasswordRejected(t *testing.T) {
	hash, err := HashPassword("the-right-password")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}

	ok, err := VerifyPassword("a-wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: unexpected error: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword: expected a wrong password to be rejected")
	}
}

func TestVerifyPassword_TamperedHashRejected(t *testing.T) {
	hash, err := HashPassword("tamper-me")
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}

	// Flip a character deep in the encoded key segment so the derived key
	// no longer matches, without producing a malformed PHC string.
	parts := strings.Split(hash, "$")
	keySegment := []byte(parts[5])
	if keySegment[0] == 'A' {
		keySegment[0] = 'B'
	} else {
		keySegment[0] = 'A'
	}
	parts[5] = string(keySegment)
	tampered := strings.Join(parts, "$")

	ok, err := VerifyPassword("tamper-me", tampered)
	if err != nil {
		t.Fatalf("VerifyPassword: unexpected error: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword: expected a tampered hash to fail verification")
	}
}

func TestVerifyPassword_MalformedHashRejectedWithoutError(t *testing.T) {
	cases := []string{
		"",
		"not-a-phc-string",
		"$argon2id$v=19$m=65536,t=3,p=2$onlyonefield",
		"$argon2i$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2g", // wrong variant
		"$argon2id$v=1$m=65536,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2g", // wrong version
		"$argon2id$v=19$m=not-a-number$c2FsdHNhbHQ$aGFzaGhhc2g",
		"$argon2id$v=19$m=65536,t=3,p=2$not-base64!!!$aGFzaGhhc2g",
		// Syntactically valid PHC strings with hostile parameters: t=0 and
		// p=0 panic inside x/crypto if they reach argon2.IDKey, p=256 wraps
		// to 0 through the uint8 cast, and a huge m turns verification into
		// a memory-exhaustion primitive. All must degrade to a quiet
		// non-match.
		"$argon2id$v=19$m=65536,t=0,p=2$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA",          // t=0
		"$argon2id$v=19$m=65536,t=3,p=0$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA",          // p=0
		"$argon2id$v=19$m=65536,t=3,p=256$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA",        // p wraps to 0
		"$argon2id$v=19$m=4294967295,t=3,p=2$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA",     // m ~4 TiB
		"$argon2id$v=19$m=65536,t=4294967295,p=2$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA", // absurd t
		"$argon2id$v=19$m=4,t=3,p=2$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaA",              // m below 8*p floor
		"$argon2id$v=19$m=65536,t=3,p=2$$aGFzaGhhc2hoYXNoaGFzaA",                             // empty salt
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2E$",                                // empty key
	}

	for _, encoded := range cases {
		ok, err := VerifyPassword("anything", encoded)
		if err != nil {
			t.Errorf("VerifyPassword(%q): expected no error for malformed hash, got %v", encoded, err)
		}
		if ok {
			t.Errorf("VerifyPassword(%q): expected malformed hash to fail verification", encoded)
		}
	}
}

func TestDecodeHash_OperationalBounds(t *testing.T) {
	// The acceptance ceiling is an operational verification budget
	// (4x the production profile), not just parse sanity: exactly the
	// ceiling parses, the first value past each bound is rejected. Tested
	// on decodeHash directly because VerifyPassword's public behavior is an
	// indistinguishable non-match either way.
	const saltSeg = "c2FsdHNhbHRzYWx0c2E"   // >= 8 bytes decoded
	const keySeg = "aGFzaGhhc2hoYXNoaGFzaA" // 16 bytes decoded

	phc := func(m, tm, p uint32) string {
		return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", m, tm, p, saltSeg, keySeg)
	}

	if _, _, _, err := decodeHash(phc(maxArgonMemory, maxArgonTime, maxArgonThreads)); err != nil {
		t.Errorf("decodeHash: expected the ceiling profile (m=%d,t=%d,p=%d) to be accepted, got %v",
			maxArgonMemory, maxArgonTime, maxArgonThreads, err)
	}

	rejected := []struct {
		name string
		phc  string
	}{
		{"memory one past ceiling", phc(maxArgonMemory+1, 3, 2)},
		{"time one past ceiling", phc(64*1024, maxArgonTime+1, 2)},
		{"threads one past ceiling", phc(64*1024, 3, maxArgonThreads+1)},
	}
	for _, tc := range rejected {
		if _, _, _, err := decodeHash(tc.phc); err == nil {
			t.Errorf("decodeHash(%s): expected rejection, got acceptance", tc.name)
		}
	}
}

func TestDummyVerify_DoesNotPanicOrLeak(t *testing.T) {
	// DummyVerify must be safe to call with arbitrary input and must not
	// somehow "succeed" for a real login attempt.
	DummyVerify("")
	DummyVerify("whatever a real user might type")

	ok, err := VerifyPassword("whatever a real user might type", dummyPasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword against dummy hash: unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected an arbitrary password to not match the fixed dummy hash")
	}
}
