package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestIssueAndVerifyAccessToken_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	token, expiresAt, err := IssueAccessToken(secret, userID, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("IssueAccessToken: expected a non-empty token")
	}
	if time.Until(expiresAt) > 30*time.Minute || time.Until(expiresAt) < 29*time.Minute {
		t.Fatalf("IssueAccessToken: expected expiresAt ~30m from now, got %v", expiresAt)
	}

	gotID, err := VerifyAccessToken(secret, token)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %v", err)
	}
	if gotID != userID {
		t.Fatalf("VerifyAccessToken: expected user id %v, got %v", userID, gotID)
	}
}

func TestIssueAccessToken_ClaimsAreMinimal(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	token, _, err := IssueAccessToken(secret, userID, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser()
	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		t.Fatalf("ParseUnverified: unexpected error: %v", err)
	}

	want := map[string]bool{"sub": true, "iat": true, "exp": true}
	for key := range claims {
		if !want[key] {
			t.Errorf("IssueAccessToken: unexpected extra claim %q in %v", key, claims)
		}
	}
	for key := range want {
		if _, ok := claims[key]; !ok {
			t.Errorf("IssueAccessToken: expected claim %q, got %v", key, claims)
		}
	}
	if sub, _ := claims["sub"].(string); sub != userID.String() {
		t.Errorf("IssueAccessToken: expected sub %q, got %q", userID.String(), sub)
	}
}

func TestVerifyAccessToken_ExpiredRejected(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	token, _, err := IssueAccessToken(secret, userID, -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	if _, err := VerifyAccessToken(secret, token); err == nil {
		t.Fatal("VerifyAccessToken: expected an expired token to be rejected")
	}
}

func TestVerifyAccessToken_WrongSecretRejected(t *testing.T) {
	userID := uuid.New()
	token, _, err := IssueAccessToken([]byte("secret-one-at-least-32-bytes-long!!"), userID, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	if _, err := VerifyAccessToken([]byte("secret-two-at-least-32-bytes-long!!"), token); err == nil {
		t.Fatal("VerifyAccessToken: expected a token signed with a different secret to be rejected")
	}
}

func TestVerifyAccessToken_GarbageTokenRejected(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")

	cases := []string{
		"",
		"not-a-jwt",
		"a.b.c",
		"eyJhbGciOiJub25lIn0.eyJzdWIiOiJhZG1pbiJ9.",
	}
	for _, tok := range cases {
		if _, err := VerifyAccessToken(secret, tok); err == nil {
			t.Errorf("VerifyAccessToken(%q): expected an error for garbage input", tok)
		}
	}
}

func TestVerifyAccessToken_RejectsAlgNone(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build alg=none token fixture: %v", err)
	}

	if _, err := VerifyAccessToken(secret, signed); err == nil {
		t.Fatal("VerifyAccessToken: expected an alg=none token to be rejected")
	}
}

func TestVerifyAccessToken_RejectsSiblingHMACAlg(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	// Correctly signed with the same secret but HS512: the policy is exactly
	// HS256, not "any HMAC", so this must be rejected even though the
	// signature itself would verify.
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign HS512 fixture token: %v", err)
	}

	if _, err := VerifyAccessToken(secret, signed); err == nil {
		t.Fatal("VerifyAccessToken: expected an HS512-signed token to be rejected")
	}
}

func TestVerifyAccessToken_MissingExpRejected(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")
	userID := uuid.New()

	// Correctly signed HS256 token with no exp claim: jwt/v5 treats exp as
	// optional by default, which would make this token valid forever.
	// Access tokens are required to be short-lived, so it must be rejected.
	claims := jwt.RegisteredClaims{
		Subject:  userID.String(),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign no-exp fixture token: %v", err)
	}

	if _, err := VerifyAccessToken(secret, signed); err == nil {
		t.Fatal("VerifyAccessToken: expected a token without exp to be rejected")
	}
}

func TestVerifyAccessToken_NonUUIDSubjectRejected(t *testing.T) {
	secret := []byte("test-secret-at-least-32-bytes-long!")

	claims := jwt.RegisteredClaims{
		Subject:   "not-a-uuid",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign fixture token: %v", err)
	}

	if _, err := VerifyAccessToken(secret, signed); err == nil {
		t.Fatal("VerifyAccessToken: expected a non-UUID subject to be rejected")
	}
}
