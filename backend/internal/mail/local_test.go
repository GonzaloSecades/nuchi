package mail

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLocalLinkMailerLogsLinkWhenDeliveryFails(t *testing.T) {
	var output bytes.Buffer
	smtpErr := errors.New("SMTP unavailable")
	inner := NewCapturingMailer()
	inner.Err = smtpErr
	m := NewLocalLinkMailer(inner, mustParseURL(t, "http://localhost:3000"), slog.New(slog.NewTextHandler(&output, nil)))
	err := m.SendVerificationEmail(context.Background(), "test@example.com", "a+b&c")
	if !errors.Is(err, smtpErr) {
		t.Fatalf("expected SMTP error, got %v", err)
	}
	if !strings.Contains(output.String(), "http://localhost:3000/verify-email?token=a%2Bb%26c") {
		t.Fatalf("missing escaped verification link: %s", output.String())
	}
}

func TestLocalLinkMailerPreservesDelivery(t *testing.T) {
	var output bytes.Buffer
	inner := NewCapturingMailer()
	m := NewLocalLinkMailer(inner, mustParseURL(t, "http://localhost:3000"), slog.New(slog.NewTextHandler(&output, nil)))
	if err := m.SendVerificationEmail(context.Background(), "test@example.com", "verify-token"); err != nil {
		t.Fatal(err)
	}
	if sends := inner.VerificationSends(); len(sends) != 1 || sends[0].Token != "verify-token" {
		t.Fatalf("verification delivery changed: %v", sends)
	}
	output.Reset()
	if err := m.SendPasswordResetEmail(context.Background(), "test@example.com", "reset-token"); err != nil {
		t.Fatal(err)
	}
	if len(inner.ResetSends()) != 1 || !strings.Contains(output.String(), "http://localhost:3000/reset-password?token=reset-token") {
		t.Fatal("password reset must be delivered and its local link logged")
	}
}

func TestLocalLinkMailerLogsResetWhenDeliveryFails(t *testing.T) {
	var output bytes.Buffer
	smtpErr := errors.New("SMTP unavailable")
	inner := NewCapturingMailer()
	inner.Err = smtpErr
	m := NewLocalLinkMailer(inner, mustParseURL(t, "http://localhost:3000"), slog.New(slog.NewTextHandler(&output, nil)))
	if err := m.SendPasswordResetEmail(context.Background(), "test@example.com", "a+b&c"); !errors.Is(err, smtpErr) {
		t.Fatalf("expected SMTP error, got %v", err)
	}
	if !strings.Contains(output.String(), "http://localhost:3000/reset-password?token=a%2Bb%26c") {
		t.Fatalf("missing escaped reset link: %s", output.String())
	}
}
