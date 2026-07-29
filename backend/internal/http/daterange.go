package httpapi

import (
	"time"
)

// maxDateRangeDays is the inclusive cap on a transaction/summary date range,
// matching the legacy helper's maxDays (lib/transaction-route-utils.ts). A
// range of exactly 366 days is accepted; 367 is not.
const maxDateRangeDays = 366

// defaultRangeDays is how far back the range starts when `from` is omitted.
const defaultRangeDays = 30

// dateRangeError carries the exact message one of the three INVALID_QUERY
// failures must produce. The strings are fixtures, not paraphrases.
type dateRangeError struct{ message string }

func (e dateRangeError) Error() string { return e.message }

var (
	errDateFormat = dateRangeError{"from and to must use yyyy-MM-dd dates."}
	errDateOrder  = dateRangeError{"from must be less than or equal to to."}
	errDateSpan   = dateRangeError{"Date range cannot exceed 366 days."}
)

// parseDateRange resolves the optional `from`/`to` query strings into the
// inclusive [start, end] window the transaction and summary queries filter on.
// from and to are the raw query values; an empty string means the parameter
// was absent.
//
// Everything is computed in UTC from a single `now`. Legacy parses these in
// the Node process's local timezone (lib/transaction-route-utils.ts), which
// makes the result depend on the host's TZ: the same request answered by a
// process in Buenos Aires and one in UTC returns different rows near midnight.
// The Go API pins UTC instead, so the boundary is a property of the API rather
// than of whichever machine served the request. That is a deliberate,
// behavior-visible divergence, recorded as post-migration improvement 0014.
//
// Rules, ported exactly:
//   - `to` defaults to now.
//   - `from` defaults to 30 days before now — NOT 30 days before a provided
//     `to`. A request supplying only `to` still starts 30 days before now.
//   - a provided `from` is the start of that calendar day (00:00:00.000).
//   - a provided `to` is the end of that calendar day (23:59:59.999999999).
//   - the range is inclusive at both ends and capped at 366 days.
func parseDateRange(from, to string, now time.Time) (start, end time.Time, err error) {
	now = now.UTC()

	end = now
	if to != "" {
		parsed, ok := parseCalendarDate(to)
		if !ok {
			return time.Time{}, time.Time{}, errDateFormat
		}
		end = endOfDay(parsed)
	}

	start = now.AddDate(0, 0, -defaultRangeDays)
	if from != "" {
		parsed, ok := parseCalendarDate(from)
		if !ok {
			return time.Time{}, time.Time{}, errDateFormat
		}
		start = parsed
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, errDateOrder
	}

	// Counted in calendar days, inclusive, so a range whose endpoints are 365
	// days apart spans 366 days and is the largest accepted.
	if calendarDaysBetween(start, end) >= maxDateRangeDays {
		return time.Time{}, time.Time{}, errDateSpan
	}

	return start, end, nil
}

// parseCalendarDate accepts exactly yyyy-MM-dd and returns the start of that
// day in UTC. It rejects anything else, including the shapes time.Parse would
// otherwise tolerate, and calendar-invalid dates such as 2026-02-30 (Go's
// parser rejects those rather than normalizing them).
func parseCalendarDate(value string) (time.Time, bool) {
	parsed, err := time.ParseInLocation(time.DateOnly, value, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// endOfDay returns the last representable instant of day's calendar date.
// Legacy uses date-fns endOfDay, which is millisecond-resolution; the column
// is a microsecond-resolution timestamp, so the extra precision here only
// widens the inclusive upper bound to cover sub-millisecond values that
// date-fns would have excluded.
func endOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
}

// calendarDaysBetween counts whole calendar days from start's date to end's
// date, ignoring the time of day — mirroring date-fns
// differenceInCalendarDays, which the legacy cap is expressed in terms of.
func calendarDaysBetween(start, end time.Time) int {
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return int(endDay.Sub(startDay).Hours() / 24)
}
