# 0018 — A transaction's date is serialized as a UTC instant, not a calendar date

- **Migration ticket:** #87 (https://github.com/GonzaloSecades/nuchi/issues/87)
- **Area:** api/schema
- **Priority guess:** medium

## How it was migrated

`transactions.date` is `timestamp without time zone`
(`backend/migrations/00002_finance_base.sql`), holding midnight on the calendar
day the user picked. The response projection hands the column straight to the
generated type — `Date: t.Date.Time` in `toTransaction` and the list projection,
`backend/internal/http/transactions.go` — where it is a `time.Time` and marshals
as RFC 3339.

pgx decodes a `timestamp without time zone` into `pgtype.Timestamp` with the
wall-clock reading and `time.UTC`, so a row stored as `2026-08-07 00:00:00` is
emitted as:

```json
{ "date": "2026-08-07T00:00:00Z" }
```

Writes are unaffected: the contract's `DateString` (`format: date`, pattern
`^\d{4}-\d{2}-\d{2}$`) is parsed to a naive `time.Time` and stored as midnight.
Only the response side is at issue here.

That midnight is a property of the current write path, not of the table. The
column carries no `CHECK`, and the legacy path stored arbitrary instants — which
is why the proposed fix below begins with an audit rather than an `ALTER TABLE`.

## Why it was done this way

Parity froze the column type. Entry 0001 records why the `timestamp` survived
the migration — the fixtures pin the serialized shape, and changing the type
mid-migration risked drift against the behavioral oracle — and the response
serialization is downstream of that choice. With the column left alone, reading
its bytes as UTC was the only self-consistent option available: the Go stack
already treats stored transaction dates as UTC everywhere else, and entry 0014
made the filter side parse in UTC for the same reason. Serializing the response
in some other zone would have made a response body a property of the deployment
host.

The legacy stack reads the same bytes differently. node-postgres interprets a
naive timestamp as **host-local** and produces a `Date`, so the same row is
serialized by Hono as:

|                                            | value                      |
| ------------------------------------------ | -------------------------- |
| stored                                     | `2026-08-07 00:00:00`      |
| node-postgres (legacy Hono), host at UTC−3 | `2026-08-07T03:00:00.000Z` |
| Go                                         | `2026-08-07T00:00:00Z`     |

The differential parity harness (#82) found this the first time it was pointed
at the two stacks, and asserted the difference was present rather than asserting
the two agreed — an assertion that fails inside a parity freeze that forbids
fixing it would only have been ignored and then deleted. That harness is retired
in #90, which is why this entry exists: it becomes the durable record.

## The concern

**The contract itself shows the modelling error.** A write takes `DateString` —
`format: date`, `2026-06-30` — and the matching read returns `format: date-time`
with example `2026-06-30T00:00:00.000Z`. A calendar date goes in and an instant
comes out. The time-of-day component is not data; it is an artifact of the
column type, and it carries a timezone claim the domain never made.

That artifact is load-bearing for clients. `new Date("2026-08-07T00:00:00Z")`
re-read in Argentina (UTC−3) is the 6th at 21:00, so a naive client renders
**every transaction date one day early** — in the app's own market, with `ARS`
as its default currency. It is worse than a display bug: an edit that changed
nothing else would submit `2026-08-06`, walking the transaction backwards one
day per save.

The frontend is already resilient to this and does not need the backend to
change first. `features/transactions/api/transaction-date.ts` converts
explicitly at each edge — UTC parts coming from the API, local parts coming from
a picker — so a date travels as `yyyy-MM-dd` between them and the round trip is
stable in every timezone. That contains the damage for this client; it does not
remove the trap for the next one, because the trap is that the API advertises an
instant for a value that is a day.

Two narrower observations, recorded so they are not rediscovered:

- Even at UTC, where the two stacks agree on the instant, they disagree on the
  string: Go emits `2026-08-07T00:00:00Z` and node-postgres
  `2026-08-07T00:00:00.000Z`. The fixtures froze the millisecond form. Both
  parse to the same instant in JavaScript, so nothing observable in the app
  turns on it, but a byte-comparison against the fixtures does.
- Because the divergence is invisible at UTC, any CI runner or container
  defaulting to UTC sees the two stacks agree. That is why it survived to be
  found by a differential harness rather than by the existing suites.

## Proposed improvement

Fix the column, not the serializer — this is the same underlying decision as
entries **0001** (the column type) and **0014** (the filter side), and all three
should land together rather than being patched at separate layers.

If a transaction's date is a **calendar date**, which the product and the write
contract both say it is:

1. **Audit the existing rows first.** Midnight is a property of the _current Go
   write path_, not an invariant of the data:

   ```sql
   SELECT id, date
   FROM transactions
   WHERE date <> date_trunc('day', date);
   ```

   Nothing has ever enforced midnight. The column is a plain
   `date timestamp NOT NULL` with no `CHECK`, and the legacy write path
   accepted arbitrary instants: `InsertTransactionSchema` overrides the field
   with `z.coerce.date()` (`db/schema.ts`), and the Hono create handler spreads
   the validated values straight into the insert, so a body carrying
   `2026-08-07T15:45:00Z` was stored with its time intact.

2. **Decide a normalization policy for whatever the audit finds**, and apply it
   explicitly before converting. This is the step that cannot be skipped,
   because a non-midnight legacy row does not merely carry noise in its time
   component — the time component may be the only thing identifying the right
   calendar day.

   node-postgres serializes a JS `Date` into a `timestamp without time zone`
   using the _Node process's local wall clock_ (`dateToString` in `pg/lib/utils.js`
   reads `getFullYear`/`getTimezoneOffset`, and Postgres then discards the
   offset for a zone-less column). So the stored wall time is the chosen
   instant rendered in the **legacy host's** zone, and the day it lands on
   depends on the gap between that zone and the user's:

   | user picks | user zone | legacy host | stored                | `date::date`  |
   | ---------- | --------- | ----------- | --------------------- | ------------- |
   | 2026-08-07 | UTC−3     | UTC−3       | `2026-08-07 00:00:00` | 2026-08-07 ✅ |
   | 2026-08-07 | UTC−3     | UTC         | `2026-08-07 03:00:00` | 2026-08-07 ✅ |
   | 2026-08-07 | UTC+9     | UTC         | `2026-08-06 15:00:00` | 2026-08-06 ❌ |

   Truncating the third row silently files the transaction a day early and
   permanently — the evidence is destroyed by the same statement. Decide which
   zone the legacy values are to be read in, confirm it against the deployment
   history rather than assuming, and normalize those rows as their own
   reviewed migration step.

3. **Then** convert the column, once the audit returns nothing outstanding:

   ```sql
   ALTER TABLE transactions ALTER COLUMN date TYPE date USING date::date;
   ```

   With every row already normalized to midnight this is lossless. Run it only
   in that state; as a first step it is a silent, irreversible data change.

4. Change the response schema from `format: date-time` to `DateString`, so read
   and write agree on the type. Filtering then needs no timezone reasoning at
   all and the whole class of day-boundary ambiguity disappears, including
   0014's.
5. Regenerate both sides from the contract and re-cut the affected fixtures —
   the serialized shape changes, which is exactly why this could not be done
   during the migration.

Step 4 is a **breaking response change** for any client parsing `date` as an
instant. The current frontend is not one: `calendarDateFromApi` already passes a
plain `yyyy-MM-dd` through unchanged, so it keeps working across the switch
without modification. That is worth confirming rather than assuming when the
work is picked up.

The alternative — `timestamptz` — is only right if transactions ever need a
time of day, and it needs a per-user timezone concept that does not exist. See
0014 for why that is the deeper but more distant option. It should not be chosen
by default just because it is the more general type.

Related: entry **0001** (column type), entry **0014** (filter parsing), and the
"Known gaps" note in `docs/api/transactions.md`.
