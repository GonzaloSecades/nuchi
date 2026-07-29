package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a controllable clock so no test ever sleeps.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestAllow_PermitsExactlyLimitThenRejects(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	for i := range Limit {
		if got := limiter.Allow("user-1", ActionCreate); !got.Allowed {
			t.Fatalf("request %d of %d was rejected; the limit must allow exactly %d", i+1, Limit, Limit)
		}
	}

	got := limiter.Allow("user-1", ActionCreate)
	if got.Allowed {
		t.Fatalf("request %d was allowed; only %d may pass in one window", Limit+1, Limit)
	}
	if got.RetryAfter != Window {
		t.Errorf("expected RetryAfter %v (whole window, since all requests landed at the same instant), got %v", Window, got.RetryAfter)
	}
}

func TestAllow_BudgetsAreIndependentPerActionAndUser(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	for range Limit {
		limiter.Allow("user-1", ActionCreate)
	}

	// Same user, different action: untouched budget.
	if got := limiter.Allow("user-1", ActionUpdate); !got.Allowed {
		t.Error("exhausting the create budget must not affect update")
	}
	if got := limiter.Allow("user-1", ActionDelete); !got.Allowed {
		t.Error("exhausting the create budget must not affect delete")
	}
	// Different user, same action: untouched budget.
	if got := limiter.Allow("user-2", ActionCreate); !got.Allowed {
		t.Error("one user exhausting its budget must not affect another user")
	}
}

func TestAllow_WindowSlides(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	for range Limit {
		limiter.Allow("user-1", ActionCreate)
	}
	if limiter.Allow("user-1", ActionCreate).Allowed {
		t.Fatal("expected the budget to be exhausted")
	}

	// Just before the window closes, still blocked.
	clock.Advance(Window - time.Second)
	if limiter.Allow("user-1", ActionCreate).Allowed {
		t.Error("requests must stay blocked until the oldest one ages out")
	}

	// Once the original burst ages out, the budget is free again.
	clock.Advance(2 * time.Second)
	if !limiter.Allow("user-1", ActionCreate).Allowed {
		t.Error("expected the budget to recover after the window elapsed")
	}
}

func TestAllow_RejectedAttemptsDoNotExtendTheWindow(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	for range Limit {
		limiter.Allow("user-1", ActionCreate)
	}

	// Hammer the limiter while blocked. If rejections were recorded, they would
	// keep pushing the window forward and the caller could never recover.
	clock.Advance(30 * time.Second)
	for range 100 {
		limiter.Allow("user-1", ActionCreate)
	}

	// The original burst is now older than the window, so the budget is free
	// regardless of the rejected flood in between.
	clock.Advance(31 * time.Second)
	if !limiter.Allow("user-1", ActionCreate).Allowed {
		t.Error("rejected attempts must not extend the window and lock the caller out")
	}
}

// TestAllow_ConcurrentCallersSeeExactlyLimitApprovals is the test that would
// fail on an unsynchronized port. Node's event loop serialized the original
// Map accesses for free; Go serves requests on many goroutines, so without the
// mutex both the count and the map itself would race.
func TestAllow_ConcurrentCallersSeeExactlyLimitApprovals(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	const attempts = 500
	var allowed atomic.Int64
	var wg sync.WaitGroup
	wg.Add(attempts)
	for range attempts {
		go func() {
			defer wg.Done()
			if limiter.Allow("user-1", ActionCreate).Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := allowed.Load(); got != Limit {
		t.Errorf("expected exactly %d concurrent approvals, got %d", Limit, got)
	}
}

func TestAllow_RetryAfterShrinksAsTheWindowAdvances(t *testing.T) {
	clock := newFakeClock()
	limiter := New(clock.Now)

	for range Limit {
		limiter.Allow("user-1", ActionCreate)
	}

	clock.Advance(20 * time.Second)
	got := limiter.Allow("user-1", ActionCreate)
	if got.Allowed {
		t.Fatal("expected rejection")
	}
	if want := 40 * time.Second; got.RetryAfter != want {
		t.Errorf("expected RetryAfter %v, got %v", want, got.RetryAfter)
	}
}
