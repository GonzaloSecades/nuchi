# 0002 — Finance tables use text cuid IDs; UUID default is v4

- **Migration ticket:** #38, #39 (https://github.com/GonzaloSecades/nuchi/issues/38, https://github.com/GonzaloSecades/nuchi/issues/39)
- **Area:** schema
- **Priority guess:** low

## How it was migrated

- `accounts`/`categories`/`transactions` keep `text` primary keys holding
  app-generated cuid-style ids, exactly as the legacy schema.
- `users` and token tables use `uuid` with app-side UUIDv7 generation
  (from #41), but the DB-level default remains `gen_random_uuid()` (UUIDv4).

## Why it was done this way

- The OpenAPI contract documents `ResourceId` as opaque cuid-style text and
  the fixtures freeze existing id shapes — changing finance id types would
  break parity.
- Native `uuidv7()` as a column default requires PostgreSQL 18; both dev
  environments (compose `postgres:17-alpine`, WSL 17.7) run 17, so v7 lives
  in app code and the v4 default is only an ad-hoc-insert safety net.

## The concern

Two id regimes in one schema (text cuid vs uuid) is permanent cognitive
overhead, and text PKs are wider than uuid columns — larger indexes, slower
joins at scale. Time-ordered ids (v7/cuid both qualify) matter for the
append-heavy transactions table; the current cuid ids do sort roughly by
creation, so this is a consistency/size concern, not a hot defect.

## Proposed improvement

After parity: converge on UUIDv7 for all tables (contract's `ResourceId` is
deliberately opaque, so the format can change without an API break, but
existing client-side id references need a data migration path). When the
stack moves to PostgreSQL 18, replace `gen_random_uuid()` defaults with
native `uuidv7()` — a one-line migration each — so DB-side inserts match
app-side generation.
