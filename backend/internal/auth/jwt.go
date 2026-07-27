package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidAccessToken is returned by VerifyAccessToken for any token that
// fails signature verification or does not carry a well-formed user id in
// its subject claim. Callers should treat it as a generic 401, never
// distinguishing the reason (that would leak information about token
// internals to a caller who should not have a valid token in the first
// place). The one deliberate carve-out is expiry: see ErrAccessTokenExpired.
var ErrInvalidAccessToken = errors.New("auth: invalid access token")

// ErrAccessTokenExpired is returned by VerifyAccessToken specifically when
// the token's signature and structure are otherwise valid and only its exp
// claim has passed. This is the one exception to ErrInvalidAccessToken's
// "never distinguish the reason" rule: an expired token was validly signed
// (its holder already proved possession of the secret's issuer-side
// counterpart, i.e. they authenticated successfully at some point), so
// telling the client "your token expired" leaks nothing an attacker could
// use, and the client needs the distinction to know it should call
// /api/auth/refresh rather than re-prompt for credentials. Every other
// rejection reason (bad signature, disallowed alg, missing/invalid subject)
// stays collapsed into ErrInvalidAccessToken.
var ErrAccessTokenExpired = errors.New("auth: access token expired")

// IssueAccessToken signs a short-lived HS256 JWT for userID. The claim set
// is intentionally minimal — sub (user id), iat, and exp only, per the #41
// design decision. Anything else the client needs (email, verification
// status) travels in the AuthSessionResponse body instead, keeping the
// token itself small and free of data that would go stale before it
// expires.
func IssueAccessToken(secret []byte, userID uuid.UUID, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	now := time.Now().UTC()
	expiresAt = now.Add(ttl)

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

// VerifyAccessToken verifies an HS256 access token against secret and
// returns the user id carried in its subject claim. It rejects tokens
// signed with any algorithm other than exactly HS256 ("alg": "none", RSA,
// and sibling HMAC strengths like HS512 are all refused — the policy is one
// algorithm, not one algorithm family), tokens without an exp claim (jwt/v5
// treats exp as optional by default; access tokens here are required to be
// short-lived, so a missing exp is a forgery signal, not a long-lived
// token), expired tokens, and tokens whose subject is not a valid UUID.
// This is the helper #43 wires into resource-route middleware to derive the
// authenticated user.
//
// Carve-out (#43): an expired-but-otherwise-valid token returns
// ErrAccessTokenExpired instead of the generic ErrInvalidAccessToken, so
// callers (the auth middleware) can surface the contract's distinct
// ACCESS_TOKEN_EXPIRED code. Every other failure — bad signature, wrong
// secret, disallowed alg, missing exp, non-UUID subject, garbage input —
// still collapses into ErrInvalidAccessToken.
func VerifyAccessToken(secret []byte, tokenString string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.UUID{}, ErrAccessTokenExpired
		}
		return uuid.UUID{}, ErrInvalidAccessToken
	}
	if !token.Valid {
		return uuid.UUID{}, ErrInvalidAccessToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, ErrInvalidAccessToken
	}

	return userID, nil
}
