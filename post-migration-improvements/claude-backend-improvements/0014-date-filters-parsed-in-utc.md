# 0014 — Date filters are parsed in UTC, not the host timezone

- **Migration ticket:** #46 (https://github.com/GonzaloSecades/nuchi/issues/46)
- **Area:** api
- **Priority guess:** low

## How it was migrated

`parseDateRange` (`backend/internal/http/daterange.go`) resolves the
transaction list's `from`/`to` window entirely in UTC, from a single captured
`now`. A supplied `from` becomes 00:00:00 UTC of that calendar day and a
supplied `to` becomes the last instant of that day in UTC.

Every other rule is ported exactly from
`lib/transaction-route-utils.ts`: `to` defaults to now, `from` defaults to 30
days before now and is not re-anchored by a provided `to`, the range is
inclusive at both ends, and the inclusive span is capped at 366 days.

## Why it was done this way

Legacy parses these dates with `date-fns` in the Node process's **local**
timezone. The window therefore depends on where the process runs: the same
request, with the same stored rows, returns different results from a host in
Buenos Aires (UTC−3) than from one in UTC, for any transaction within three
hours of a day boundary.

The Go stack already treats transaction dates as UTC — `transactions.date` is
`timestamp without time zone` and every query writes UTC — so parsing filters
in a host-dependent zone would have been the odd one out, and would make the
API's answers a property of the deployment rather than of the request.

## The concern

This is a **behavior-visible divergence**, not pure hardening. If the legacy
Node process ran in a non-UTC zone, a client near a day boundary sees a
different row set from the Go API than from Hono. In practice both dev
environments and any container default to UTC, so the difference is
unobservable today — but it is a real difference and is recorded rather than
buried.

The deeper modelling issue is separate: `date` behaves like a calendar date
(users pick a day, not an instant) while the column stores a timestamp. UTC
parsing makes the current model consistent; it does not make it correct.

## Proposed improvement

After parity, decide the model deliberately rather than inheriting it:

- If a transaction's date is a **calendar date**, migrate the column to
  PostgreSQL `date` and the contract field to `format: date`. Filtering then
  needs no timezone reasoning at all, and the whole class of boundary
  ambiguity disappears.
- If it is an **instant**, migrate to `timestamptz` and let PostgreSQL hold
  the zone, then decide whose day boundary a filter means — the server's, or
  a per-user timezone preference.

The second option is the one that eventually matters for a finance app used
across timezones, but it needs a user-timezone concept that does not exist
yet. Either way this should land alongside the calendar-date decision in
entry 0001, not before it.
