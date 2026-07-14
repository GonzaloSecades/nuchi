package auth

import (
	"encoding/hex"
	"testing"
)

func TestGenerateToken_ProducesMatchingHash(t *testing.T) {
	raw, hash, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: unexpected error: %v", err)
	}

	if raw == "" {
		t.Fatal("GenerateToken: expected a non-empty raw token")
	}
	if hash != HashToken(raw) {
		t.Fatalf("GenerateToken: hash %q does not match HashToken(raw) %q", hash, HashToken(raw))
	}

	decoded, err := hex.DecodeString(hash)
	if err != nil {
		t.Fatalf("GenerateToken: hash is not valid hex: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("GenerateToken: expected a 32-byte SHA-256 digest, got %d bytes", len(decoded))
	}
}

func TestGenerateToken_UniquePerCall(t *testing.T) {
	raw1, hash1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: unexpected error: %v", err)
	}
	raw2, hash2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: unexpected error: %v", err)
	}

	if raw1 == raw2 {
		t.Fatal("GenerateToken: expected two calls to produce different raw values")
	}
	if hash1 == hash2 {
		t.Fatal("GenerateToken: expected two calls to produce different hashes")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	if HashToken("same-input") != HashToken("same-input") {
		t.Fatal("HashToken: expected the same input to hash identically")
	}
	if HashToken("input-a") == HashToken("input-b") {
		t.Fatal("HashToken: expected different inputs to hash differently")
	}
}
