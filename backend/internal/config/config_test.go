package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("BACKEND_HOST", "")
	t.Setenv("BACKEND_PORT", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.Host != defaultHost {
		t.Fatalf("expected default host %q, got %q", defaultHost, cfg.Host)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("expected default database URL %q, got %q", defaultDatabaseURL, cfg.DatabaseURL)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("BACKEND_HOST", "127.0.0.1")
	t.Setenv("BACKEND_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://someone:secret@example.invalid:5432/nuchi?sslmode=require")

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected overridden host, got %q", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected overridden port, got %q", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://someone:secret@example.invalid:5432/nuchi?sslmode=require" {
		t.Fatalf("expected overridden database URL, got %q", cfg.DatabaseURL)
	}
}
