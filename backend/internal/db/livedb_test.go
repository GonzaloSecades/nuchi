package db

import (
	"os"
	"testing"
)

// liveDatabaseURL returns TEST_DATABASE_URL for tests that need a real
// PostgreSQL instance.
//
// Locally an unset value skips, so `go test ./...` stays useful without a
// database running. In CI it is a hard failure instead: CI provisions a
// postgres service and runs the migrations, so a missing URL there means
// the workflow broke, and the whole point of these tests is that they
// cannot quietly stop running. Silent skips are how this suite went a
// stretch of tickets covering nothing in CI.
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

// adminDatabaseURL returns ADMIN_DATABASE_URL, a connection that authenticates
// as a role which bypasses RLS (the bootstrap superuser). Tests use it to
// prove VerifyRLSActive rejects such a role — the negative half of the startup
// guard, which cannot be shown with the ordinary TEST_DATABASE_URL role.
//
// Same skip-local / fail-in-CI contract as liveDatabaseURL: CI exports
// ADMIN_DATABASE_URL for the whole backend job, so a missing value there means
// the workflow broke rather than that the guard is untested.
func adminDatabaseURL(t *testing.T, what string) string {
	t.Helper()

	databaseURL := os.Getenv("ADMIN_DATABASE_URL")
	if databaseURL != "" {
		return databaseURL
	}
	if os.Getenv("CI") != "" {
		t.Fatalf("ADMIN_DATABASE_URL is unset in CI; %s must run against the workflow's admin role, never be skipped", what)
	}
	t.Skipf("ADMIN_DATABASE_URL not set; skipping %s", what)
	return ""
}
