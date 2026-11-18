package mail

import (
	"context"
	"log/slog"
	"net/url"
)

// localLinkMailer is enabled only by the local-development wiring in cmd/api.
// Both methods still send through the underlying SMTP mailer.
type localLinkMailer struct {
	Mailer
	baseURL *url.URL
	logger  *slog.Logger
}

func NewLocalLinkMailer(inner Mailer, baseURL *url.URL, logger *slog.Logger) Mailer {
	return &localLinkMailer{Mailer: inner, baseURL: baseURL, logger: logger}
}

func (m *localLinkMailer) SendVerificationEmail(ctx context.Context, to, token string) error {
	m.logger.InfoContext(ctx, "local signup verification link", "url", buildLink(m.baseURL, verifyEmailPath, token))
	return m.Mailer.SendVerificationEmail(ctx, to, token)
}

func (m *localLinkMailer) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	m.logger.InfoContext(ctx, "local password reset link", "url", buildLink(m.baseURL, resetPasswordPath, token))
	return m.Mailer.SendPasswordResetEmail(ctx, to, token)
}
