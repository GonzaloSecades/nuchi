// Package db wires the PostgreSQL connection pool used by the API. It owns
// only pool lifecycle (open, verify, close); sqlc-generated queries and
// business logic live elsewhere.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pingTimeout bounds how long NewPool waits for the initial connectivity
// check before giving up.
const pingTimeout = 5 * time.Second

// NewPool parses databaseURL, creates a pgxpool.Pool, and verifies
// connectivity with a bounded ping. The returned pool is ready for use; the
// caller owns closing it. NewPool never panics and never logs; callers decide
// how to surface failures.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return pool, nil
}
