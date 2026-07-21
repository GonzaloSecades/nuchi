package db

import (
	"context"
	"testing"
	"time"
)

func TestNewPool_InvalidURL(t *testing.T) {
	ctx := context.Background()

	pool, err := NewPool(ctx, "not-a-valid-connection-string://")
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for invalid database URL, got nil")
	}
	if pool != nil {
		t.Fatal("expected nil pool on error")
	}
}

func TestNewPool_UnreachableServer(t *testing.T) {
	// Port 1 has nothing listening on it locally, so the ping fails fast
	// (connection refused) and, at worst, within the bounded timeout rather
	// than hanging indefinitely.
	ctx := context.Background()
	start := time.Now()

	pool, err := NewPool(ctx, "postgres://nuchi:nuchi@127.0.0.1:1/nuchi?sslmode=disable")

	elapsed := time.Since(start)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for unreachable database, got nil")
	}
	if pool != nil {
		t.Fatal("expected nil pool on error")
	}
	if elapsed > 6*time.Second {
		t.Fatalf("expected NewPool to fail within ~5s timeout, took %s", elapsed)
	}
}

// TestNewPool_LiveDatabase optionally exercises NewPool against a real
// PostgreSQL instance -- the postgres service CI provisions, or a local one.
// Skipped only when TEST_DATABASE_URL is unset outside CI.
func TestNewPool_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "live database test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("expected pool to be pingable, got error: %v", err)
	}
}
