package httpapi

import (
	"testing"
	"time"
)

// fixedNow is an arbitrary mid-afternoon instant. The time of day matters:
// several rules below are only distinguishable when "now" is not midnight.
var fixedNow = time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)

// present marks a query parameter as supplied with the given value. A nil
// pointer means the parameter was omitted; &"" means it was supplied empty,
// which is malformed rather than a request for the default.
func present(value string) *string { return &value }

func TestParseDateRange_Defaults(t *testing.T) {
	start, end, err := parseDateRange(nil, nil, fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !end.Equal(fixedNow) {
		t.Errorf("default end should be now (%v), got %v", fixedNow, end)
	}
	if want := fixedNow.AddDate(0, 0, -30); !start.Equal(want) {
		t.Errorf("default start should be 30 days before now (%v), got %v", want, start)
	}
}

// TestParseDateRange_ProvidedToDoesNotMoveDefaultFrom pins the rule that is
// easiest to implement wrongly: supplying only `to` must not re-anchor the
// default `from` to 30 days before `to`.
func TestParseDateRange_ProvidedToDoesNotMoveDefaultFrom(t *testing.T) {
	start, end, err := parseDateRange(nil, present("2026-07-20"), fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := fixedNow.AddDate(0, 0, -30); !start.Equal(want) {
		t.Errorf("start must stay 30 days before now (%v), not 30 days before `to`; got %v", want, start)
	}
	if want := time.Date(2026, 7, 20, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC); !end.Equal(want) {
		t.Errorf("end should be the end of the provided day (%v), got %v", want, end)
	}
}

func TestParseDateRange_BoundariesAreWholeDays(t *testing.T) {
	start, end, err := parseDateRange(present("2026-06-01"), present("2026-06-30"), fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Errorf("start should be the start of the day (%v), got %v", want, start)
	}
	if want := time.Date(2026, 6, 30, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC); !end.Equal(want) {
		t.Errorf("end should be the end of the day (%v), got %v", want, end)
	}
}

func TestParseDateRange_InvalidInputs(t *testing.T) {
	cases := []struct {
		name    string
		from    *string
		to      *string
		wantMsg string
	}{
		{"malformed from", present("2026/06/01"), nil, errDateFormat.message},
		{"malformed to", nil, present("june"), errDateFormat.message},
		{"non-calendar date", present("2026-02-30"), nil, errDateFormat.message},
		{"short year", present("26-06-01"), nil, errDateFormat.message},
		{"from after to", present("2026-06-30"), present("2026-06-01"), errDateOrder.message},
		// 367 inclusive days: one past the cap.
		{"range too long", present("2025-07-01"), present("2026-07-02"), errDateSpan.message},
		// Present but empty. url.Values.Get cannot distinguish these from
		// omission, which is why presence is passed in as a pointer; the
		// contract's parameters do not set allowEmptyValue (false by default in
		// OpenAPI 3.0.3), so an empty value is malformed, not a default request.
		{"empty from", present(""), nil, errDateFormat.message},
		{"empty to", nil, present(""), errDateFormat.message},
		{"both empty", present(""), present(""), errDateFormat.message},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseDateRange(tc.from, tc.to, fixedNow)
			if err == nil {
				t.Fatalf("expected an error for from=%v to=%v", tc.from, tc.to)
			}
			if err.Error() != tc.wantMsg {
				t.Errorf("expected message %q, got %q", tc.wantMsg, err.Error())
			}
		})
	}
}

// TestParseDateRange_SpanBoundary pins both sides of the 366-day cap: legacy
// rejects when differenceInCalendarDays + 1 > 366, so endpoints 365 days apart
// (366 inclusive days) are the largest accepted.
func TestParseDateRange_SpanBoundary(t *testing.T) {
	if _, _, err := parseDateRange(present("2025-07-01"), present("2026-07-01"), fixedNow); err != nil {
		t.Errorf("exactly 366 inclusive days must be accepted, got %v", err)
	}
	if _, _, err := parseDateRange(present("2025-07-01"), present("2026-07-02"), fixedNow); err == nil {
		t.Error("367 inclusive days must be rejected")
	}
}

// TestParseDateRange_SameDayIsValid guards against an off-by-one that would
// reject the narrowest legitimate range.
func TestParseDateRange_SameDayIsValid(t *testing.T) {
	start, end, err := parseDateRange(present("2026-06-15"), present("2026-06-15"), fixedNow)
	if err != nil {
		t.Fatalf("a single-day range must be valid, got %v", err)
	}
	if !start.Before(end) {
		t.Errorf("expected start (%v) before end (%v) for a single day", start, end)
	}
}

// TestParseDateRange_IsUTCRegardlessOfInputZone documents the deliberate
// divergence from legacy, which parses in the host process's local timezone
// and therefore returns different rows depending on where it runs.
func TestParseDateRange_IsUTCRegardlessOfInputZone(t *testing.T) {
	buenosAires := time.FixedZone("ART", -3*60*60)
	nowThere := fixedNow.In(buenosAires)

	start, end, err := parseDateRange(present("2026-06-01"), present("2026-06-30"), nowThere)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Location() != time.UTC || end.Location() != time.UTC {
		t.Errorf("range must be pinned to UTC, got start=%v end=%v", start.Location(), end.Location())
	}
	if want := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Errorf("expected %v, got %v", want, start)
	}
}
