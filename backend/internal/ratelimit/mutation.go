// Package ratelimit implements the per-user, per-action sliding-window limit
// the transaction mutation endpoints enforce.
//
// It reproduces the legacy limiter in lib/transaction-route-utils.ts: at most
// 60 accepted requests in the preceding 60 seconds, counted separately for
// each (user, action) pair, with the 61st rejected and told how long to wait.
//
// Two differences from the Node original are deliberate. First, it is
// explicitly synchronized: Node's single event loop made its Map accesses
// atomic for free, while Go serves requests on many goroutines, so an
// unsynchronized port would allow more than 60 through under concurrency and
// would race. Second, the clock is injectable, so tests can advance time
// instead of sleeping.
//
// State is in-memory, matching legacy: it resets on restart, and N replicas
// permit N times the limit. That is recorded as post-migration improvement
// 0003 and is not addressed here.
package ratelimit

import (
	"sync"
	"time"
)

// Action distinguishes the independent budgets a single user has. Legacy keys
// the map by `${userId}:${action}`, so create/update/delete never share a
// budget; the strings differ from legacy's ("patch") only in name, which is
// unobservable — the map is process-local and empty at startup.
type Action string

const (
	ActionCreate     Action = "create"
	ActionUpdate     Action = "update"
	ActionDelete     Action = "delete"
	ActionBulkCreate Action = "bulk-create"
	ActionBulkDelete Action = "bulk-delete"
)

const (
	// Window and Limit mirror MUTATION_RATE_LIMIT_WINDOW_MS and
	// MUTATION_RATE_LIMIT_MAX_REQUESTS in lib/transaction-route-utils.ts.
	Window = time.Minute
	Limit  = 60

	// cleanupInterval mirrors MUTATION_RATE_LIMIT_CLEANUP_INTERVAL_MS. Without
	// periodic sweeping the map grows once per user forever, since a key is
	// only pruned when that same user comes back.
	cleanupInterval = 5 * time.Minute
)

// key identifies one budget: a single user's allowance for a single action.
type key struct {
	userID string
	action Action
}

// Result reports one limiter decision. RetryAfter is meaningful only when
// Allowed is false, and is the time until the oldest request in the window
// expires.
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Limiter is a concurrency-safe sliding-window limiter. The zero value is not
// usable; construct it with New.
type Limiter struct {
	now func() time.Time

	mu          sync.Mutex
	requests    map[key][]time.Time
	lastCleanup time.Time
}

// New builds a Limiter. now may be nil, in which case time.Now is used;
// tests pass a controllable clock so they never sleep.
func New(now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{now: now, requests: make(map[key][]time.Time)}
}

// Allow records an attempt for (userID, action) and reports whether it may
// proceed.
//
// A rejected attempt is deliberately NOT recorded: legacy drops the request
// without appending a timestamp, so a client hammering a limited endpoint does
// not keep pushing its own window forward and lock itself out indefinitely.
//
// Callers must invoke this once per mutation attempt, before doing the work —
// a failed ownership lookup or a database error still consumes the attempt,
// matching legacy, where the check runs before reference validation.
func (l *Limiter) Allow(userID string, action Action) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanupLocked(now)

	k := key{userID: userID, action: action}
	windowStart := now.Add(-Window)

	// Drop timestamps that have aged out, keeping only the live window.
	recent := l.requests[k][:0:0]
	for _, at := range l.requests[k] {
		if at.After(windowStart) {
			recent = append(recent, at)
		}
	}

	if len(recent) >= Limit {
		l.requests[k] = recent
		// Legacy: recentRequests[0] + WINDOW - now.
		return Result{Allowed: false, RetryAfter: recent[0].Add(Window).Sub(now)}
	}

	l.requests[k] = append(recent, now)
	return Result{Allowed: true}
}

// cleanupLocked drops keys whose every timestamp has aged out. Callers must
// hold l.mu.
func (l *Limiter) cleanupLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < cleanupInterval {
		return
	}
	l.lastCleanup = now

	windowStart := now.Add(-Window)
	for k, timestamps := range l.requests {
		live := false
		for _, at := range timestamps {
			if at.After(windowStart) {
				live = true
				break
			}
		}
		if !live {
			delete(l.requests, k)
		}
	}
}
