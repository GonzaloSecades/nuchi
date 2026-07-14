package auth

import (
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
