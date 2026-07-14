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
