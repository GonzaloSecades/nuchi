// Package mail sends the two auth-flow emails (verification, password
// reset) behind a small interface so HTTP handlers never depend on SMTP
// directly. SMTPMailer (smtp.go) is the production implementation, talking
// to a local Mailpit instance in dev; tests inject a capturing fake
// (fake.go) instead of standing up an SMTP server.
package mail

import (
	"context"
	"fmt"
	"net/url"
)

// Mailer sends the two auth-flow emails. Both methods take the recipient
// address and the raw (unhashed) one-time token; implementations build the
// verification/reset link themselves.
type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, token string) error
	SendPasswordResetEmail(ctx context.Context, to, token string) error
}

// verifyEmailPath and resetPasswordPath are the frontend routes (#51,
// not yet built) that a verification/reset link points at, with the token
// carried in a query parameter.
const (
	verifyEmailPath   = "/verify-email"
	resetPasswordPath = "/reset-password"
)

// buildLink joins baseURL with path and an escaped ?token= query parameter.
// baseURL is expected to already be validated (scheme + host present) by
// internal/config at load time; buildLink does not re-validate it.
func buildLink(baseURL *url.URL, path, token string) string {
	u := *baseURL
	u.Path = path
	q := url.Values{}
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

// verificationEmailBody and passwordResetEmailBody render the plain-text
// body for each email: a clickable link first, then the raw token on its
// own line for curl-based manual testing (per #42's scope — HTML templates
// are out of scope).
func verificationEmailBody(link, token string) string {
	return fmt.Sprintf(
		"Verify your email address by visiting the link below. This link expires soon.\n\n%s\n\nIf the link does not work, use this token directly:\n%s\n",
		link, token,
	)
}

func passwordResetEmailBody(link, token string) string {
	return fmt.Sprintf(
		"Reset your password by visiting the link below. This link expires soon and can only be used once.\n\nIf you did not request a password reset, you can ignore this email.\n\n%s\n\nIf the link does not work, use this token directly:\n%s\n",
		link, token,
	)
}
