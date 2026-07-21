package httpapi

import (
	"os"
	"testing"
)

// liveDatabaseURL returns TEST_DATABASE_URL for tests that need a real
// PostgreSQL instance. Mirrors internal/db's helper of the same name (the
// two packages cannot share a test helper without exporting it from a
// non-test package).
//
// Locally an unset value skips, so `go test ./...` stays useful without a
// database running. In CI it is a hard failure instead: CI provisions a
// postgres service and runs the migrations, so a missing URL there means
// the workflow broke. Every behavioral test for the auth and email flows
// lives behind this gate, and a silent skip would take all of it out of CI
// without turning anything red.
func liveDatabaseURL(t *testing.T, what string) string {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL != "" {
		return databaseURL
	}
	if os.Getenv("CI") != "" {
		t.Fatalf("TEST_DATABASE_URL is unset in CI; %s must run against the workflow's postgres service, never be skipped", what)
	}
	t.Skipf("TEST_DATABASE_URL not set; skipping %s", what)
	return ""
}
