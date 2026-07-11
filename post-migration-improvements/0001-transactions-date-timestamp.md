# 0001 — transactions.date is timestamp without time zone

- **Migration ticket:** #39 (https://github.com/GonzaloSecades/nuchi/issues/39)
- **Area:** schema
- **Priority guess:** medium

## How it was migrated

`backend/migrations/00002_finance_base.sql` declares
`date timestamp NOT NULL` — no time zone — holding UTC-midnight datetimes
derived from `YYYY-MM-DD` user input.

## Why it was done this way

The legacy Drizzle column is `timestamp` without time zone, and the fixtures
freeze the serialized shape (`2026-06-30T00:00:00.000Z`). Switching types
mid-migration risked subtle serialization drift against the behavioral
oracle, so parity won.

## The concern

A timestamp with no zone is ambiguous by construction: correctness currently
depends on every writer agreeing it means UTC, which is a convention, not a
constraint. It also stores more precision than the domain has — the product
concept is a calendar day, and users in Argentina (UTC-3) may eventually see
day-boundary surprises if any code path interprets the value in local time.

## Proposed improvement

Change the column to `date` (the product truth: transactions belong to a
calendar day), adjusting the API layer to keep emitting the contract's
date-time shape. Alternative if time-of-day is ever wanted: `timestamptz`.
Either removes the ambiguity; `date` is smaller, indexes tighter, and makes
the day-boundary semantics self-evident. Requires a data-preserving
`ALTER TABLE ... USING date::date` migration and a fixtures/contract review,
so it must wait for the post-parity window.
