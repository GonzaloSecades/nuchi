package config

import (
	"strings"
	"testing"
	"time"
)

// validJWTSecret is a 32+ byte value good enough to satisfy Load's
// fail-fast length check in tests that aren't specifically exercising that
// check.
const validJWTSecret = "test-only-secret-at-least-32-bytes-long!!"

func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "")
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "")
	t.Setenv("AUTH_COOKIE_SECURE", "")
	t.Setenv("SMTP_ADDR", "")
	t.Setenv("MAIL_FROM", "")
	t.Setenv("APP_BASE_URL", "")
	t.Setenv("AUTH_VERIFICATION_TOKEN_TTL", "")
	t.Setenv("AUTH_RESET_TOKEN_TTL", "")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("BACKEND_HOST", "")
	t.Setenv("BACKEND_PORT", "")
	t.Setenv("DATABASE_URL", "")
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	if cfg.Host != defaultHost {
		t.Fatalf("expected default host %q, got %q", defaultHost, cfg.Host)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("expected default database URL %q, got %q", defaultDatabaseURL, cfg.DatabaseURL)
	}
	if cfg.AccessTokenTTL != defaultAccessTokenTTL {
		t.Errorf("expected default access token TTL %v, got %v", defaultAccessTokenTTL, cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != defaultRefreshTokenTTL {
		t.Errorf("expected default refresh token TTL %v, got %v", defaultRefreshTokenTTL, cfg.RefreshTokenTTL)
	}
	if cfg.CookieSecure != defaultCookieSecure {
		t.Errorf("expected default cookie secure %v, got %v", defaultCookieSecure, cfg.CookieSecure)
	}
	if string(cfg.JWTSecret) != validJWTSecret {
		t.Errorf("expected JWTSecret %q, got %q", validJWTSecret, cfg.JWTSecret)
	}
	if cfg.SMTPAddr != defaultSMTPAddr {
		t.Errorf("expected default SMTP addr %q, got %q", defaultSMTPAddr, cfg.SMTPAddr)
	}
	if cfg.MailFrom != defaultMailFrom {
		t.Errorf("expected default mail from %q, got %q", defaultMailFrom, cfg.MailFrom)
	}
	if cfg.AppBaseURL == nil || cfg.AppBaseURL.String() != defaultAppBaseURL {
		t.Errorf("expected default app base URL %q, got %v", defaultAppBaseURL, cfg.AppBaseURL)
	}
	if cfg.VerificationTokenTTL != defaultVerificationTokenTTL {
		t.Errorf("expected default verification token TTL %v, got %v", defaultVerificationTokenTTL, cfg.VerificationTokenTTL)
	}
	if cfg.ResetTokenTTL != defaultResetTokenTTL {
		t.Errorf("expected default reset token TTL %v, got %v", defaultResetTokenTTL, cfg.ResetTokenTTL)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("BACKEND_HOST", "127.0.0.1")
	t.Setenv("BACKEND_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://someone:secret@example.invalid:5432/nuchi?sslmode=require")
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "15m")
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "168h")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	t.Setenv("SMTP_ADDR", "mail.example.invalid:2525")
	t.Setenv("MAIL_FROM", "no-reply@example.invalid")
	t.Setenv("APP_BASE_URL", "https://app.example.invalid")
	t.Setenv("AUTH_VERIFICATION_TOKEN_TTL", "72h")
	t.Setenv("AUTH_RESET_TOKEN_TTL", "15m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected overridden host, got %q", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected overridden port, got %q", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://someone:secret@example.invalid:5432/nuchi?sslmode=require" {
		t.Fatalf("expected overridden database URL, got %q", cfg.DatabaseURL)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("expected overridden access token TTL 15m, got %v", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 168*time.Hour {
		t.Errorf("expected overridden refresh token TTL 168h, got %v", cfg.RefreshTokenTTL)
	}
	if !cfg.CookieSecure {
		t.Error("expected overridden cookie secure to be true")
	}
	if cfg.SMTPAddr != "mail.example.invalid:2525" {
		t.Errorf("expected overridden SMTP addr, got %q", cfg.SMTPAddr)
	}
	if cfg.MailFrom != "no-reply@example.invalid" {
		t.Errorf("expected overridden mail from, got %q", cfg.MailFrom)
	}
	if cfg.AppBaseURL == nil || cfg.AppBaseURL.String() != "https://app.example.invalid" {
		t.Errorf("expected overridden app base URL, got %v", cfg.AppBaseURL)
	}
	if cfg.VerificationTokenTTL != 72*time.Hour {
		t.Errorf("expected overridden verification token TTL 72h, got %v", cfg.VerificationTokenTTL)
	}
	if cfg.ResetTokenTTL != 15*time.Minute {
		t.Errorf("expected overridden reset token TTL 15m, got %v", cfg.ResetTokenTTL)
	}
}

func TestLoad_MissingJWTSecretFailsFast(t *testing.T) {
	clearAuthEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load: expected an error when AUTH_JWT_SECRET is unset")
	}
	if !strings.Contains(err.Error(), "AUTH_JWT_SECRET") {
		t.Errorf("Load: expected error to mention AUTH_JWT_SECRET, got %q", err.Error())
	}
}

func TestLoad_ShortJWTSecretFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "too-short")

	_, err := Load()
	if err == nil {
		t.Fatal("Load: expected an error when AUTH_JWT_SECRET is shorter than 32 bytes")
	}
}

func TestLoad_InvalidAccessTokenTTLFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load: expected an error for a malformed AUTH_ACCESS_TOKEN_TTL")
	}
}

func TestLoad_InvalidRefreshTokenTTLFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load: expected an error for a malformed AUTH_REFRESH_TOKEN_TTL")
	}
}

func TestLoad_NonPositiveTTLsFailFast(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"zero access TTL", "AUTH_ACCESS_TOKEN_TTL", "0s"},
		{"negative access TTL", "AUTH_ACCESS_TOKEN_TTL", "-5m"},
		{"sub-second access TTL", "AUTH_ACCESS_TOKEN_TTL", "500ms"},
		{"zero refresh TTL", "AUTH_REFRESH_TOKEN_TTL", "0s"},
		{"negative refresh TTL", "AUTH_REFRESH_TOKEN_TTL", "-720h"},
		{"sub-second refresh TTL", "AUTH_REFRESH_TOKEN_TTL", "10ms"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAuthEnv(t)
			t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
			t.Setenv(tc.key, tc.value)

			if _, err := Load(); err == nil {
				t.Fatalf("Load: expected an error for %s=%s - parseable but non-positive TTLs issue already-expired tokens", tc.key, tc.value)
			}
		})
	}
}

func TestLoad_InvalidCookieSecureFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_COOKIE_SECURE", "not-a-bool")

	if _, err := Load(); err == nil {
		t.Fatal("Load: expected an error for a malformed AUTH_COOKIE_SECURE")
	}
}

func TestLoad_MalformedAppBaseURLFailsFast(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"missing scheme", "localhost:3000"},
		{"missing host", "http://"},
		{"unparsable", "://bad"},
		{"scheme only, no host or slashes", "not-a-url"},
		// The value becomes a link in an email, so it must be a web origin:
		// a non-web scheme carries a host and would otherwise pass, then
		// produce a link the frontend cannot handle.
		{"ftp scheme", "ftp://example.invalid"},
		{"javascript scheme", "javascript://example.invalid"},
		{"file scheme", "file://example.invalid"},
		{"userinfo", "https://user:pass@example.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAuthEnv(t)
			t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
			t.Setenv("APP_BASE_URL", tc.url)

			if _, err := Load(); err == nil {
				t.Fatalf("Load: expected an error for APP_BASE_URL=%q", tc.url)
			}
		})
	}
}

// TestLoad_NonOriginAppBaseURLFailsFast pins the origin-only rule.
// internal/mail builds links by replacing the base URL's path and query, so
// anything carried in APP_BASE_URL beyond the origin would be silently
// dropped from the link inside an email — the failure this validation
// exists to prevent. It must be a startup error, not a surprise later.
func TestLoad_NonOriginAppBaseURLFailsFast(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"subpath", "https://example.invalid/app"},
		{"deep subpath", "https://example.invalid/a/b"},
		{"query", "https://example.invalid?tenant=nuchi"},
		{"fragment", "https://example.invalid#top"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAuthEnv(t)
			t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
			t.Setenv("APP_BASE_URL", tc.url)

			if _, err := Load(); err == nil {
				t.Fatalf("Load: expected an error for non-origin APP_BASE_URL=%q", tc.url)
			}
		})
	}
}

// A bare origin, with or without a trailing slash, is the supported shape.
func TestLoad_OriginAppBaseURLAccepted(t *testing.T) {
	for _, raw := range []string{"https://example.invalid", "https://example.invalid/", "http://localhost:3000"} {
		t.Run(raw, func(t *testing.T) {
			clearAuthEnv(t)
			t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
			t.Setenv("APP_BASE_URL", raw)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: unexpected error for APP_BASE_URL=%q: %v", raw, err)
			}
			if cfg.AppBaseURL.Host == "" {
				t.Errorf("expected a parsed host for APP_BASE_URL=%q", raw)
			}
		})
	}
}

func TestLoad_InvalidVerificationTokenTTLFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_VERIFICATION_TOKEN_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load: expected an error for a malformed AUTH_VERIFICATION_TOKEN_TTL")
	}
}

func TestLoad_InvalidResetTokenTTLFailsFast(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
	t.Setenv("AUTH_RESET_TOKEN_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load: expected an error for a malformed AUTH_RESET_TOKEN_TTL")
	}
}

func TestLoad_NonPositiveMailTTLsFailFast(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"zero verification TTL", "AUTH_VERIFICATION_TOKEN_TTL", "0s"},
		{"negative verification TTL", "AUTH_VERIFICATION_TOKEN_TTL", "-48h"},
		{"sub-second verification TTL", "AUTH_VERIFICATION_TOKEN_TTL", "500ms"},
		{"zero reset TTL", "AUTH_RESET_TOKEN_TTL", "0s"},
		{"negative reset TTL", "AUTH_RESET_TOKEN_TTL", "-30m"},
		{"sub-second reset TTL", "AUTH_RESET_TOKEN_TTL", "10ms"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAuthEnv(t)
			t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
			t.Setenv(tc.key, tc.value)

			if _, err := Load(); err == nil {
				t.Fatalf("Load: expected an error for %s=%s - parseable but non-positive TTLs issue already-expired tokens", tc.key, tc.value)
			}
		})
	}
}
