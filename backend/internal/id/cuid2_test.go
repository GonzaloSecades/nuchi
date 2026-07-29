package id

import (
	"regexp"
	"testing"
)

// cuid2Pattern is the format legacy @paralleldrive/cuid2 ids satisfy and
// that finance resource ids must keep for parity: 24 characters, a leading
// lowercase letter, the rest base36.
var cuid2Pattern = regexp.MustCompile(`^[a-z][0-9a-z]{23}$`)

func TestNew_MatchesCuid2Format(t *testing.T) {
	for range 100 {
		got := New()
		if !cuid2Pattern.MatchString(got) {
			t.Fatalf("id %q does not match the cuid2 format %s", got, cuid2Pattern)
		}
	}
}

func TestNew_IsUnique(t *testing.T) {
	// Enough iterations to catch a generator that collides within a single
	// millisecond tick — the failure mode a time-only id would have.
	const iterations = 10_000

	seen := make(map[string]struct{}, iterations)
	for range iterations {
		got := New()
		if _, dup := seen[got]; dup {
			t.Fatalf("duplicate id generated: %q", got)
		}
		seen[got] = struct{}{}
	}
}
