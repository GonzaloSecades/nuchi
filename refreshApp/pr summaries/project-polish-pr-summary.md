# ProjectPolish PR Summary

## Purpose

Create a clean local development restart point and close high-priority security/performance review findings before further feature work.

## Main Changes

- Added Docker PostgreSQL 17 local development through `docker-compose.dev.yml`.
- Changed `bun dev` to start/check local Postgres, run migrations, and boot Next against the local DB.
- Kept Neon development available through `bun run dev:neon`.
- Switched Drizzle runtime from Neon HTTP to `node-postgres` via `db/connection.ts`.
- Moved API route runtime to Node.js.
- Hardened transaction writes with account/category tenant ownership checks for create, bulk-create, and patch.
- Added bulk transaction caps, request body guards, request rate limiting, and CSV import chunking.
- Guarded seed deletes behind `APP_ENV=local`, `ALLOW_DB_SEED=true`, and local DB URL validation.
- Tightened summary/transaction date filters with strict `yyyy-MM-dd` parsing and inclusive end-of-day `to` ranges.
- Remote Postgres TLS now verifies certificates by default, with `ALLOW_INSECURE_DATABASE_TLS=true` as an explicit opt-out for known self-signed scenarios.
- App Drizzle pool creation uses a `globalThis` cache outside production to avoid Next.js dev HMR pool multiplication.
- Removed the debug `/api/summary/asd` route.
- Fixed browser API base URL resolution to avoid local dev cross-port CORS errors.
- Auto-selects inline-created accounts/categories inside transaction flows.
- Removed `localEnv.zip` and ignored zip files.

## Key Files

- `docker-compose.dev.yml`
- `scripts/dev-local.ts`
- `db/connection.ts`
- `db/drizzle.ts`
- `app/api/[[...route]]/transactions.ts`
- `app/api/[[...route]]/summary.ts`
- `scripts/seed.ts`
- `components/select.tsx`
- `features/transactions/components/new-transaction-sheet.tsx`
- `features/transactions/components/edit-transaction-sheet.tsx`
- `features/accounts/hooks/use-select-account.tsx`
- `lib/transaction-route-utils.ts`
- `lib/transaction-limits.ts`
- `lib/api-base-url.ts`
- `lib/chunk-items.ts`
- `lib/select-options.ts`
- `db/connection.test.ts`

## Tests Added

- `db/connection.test.ts`
- `lib/transaction-route-utils.test.ts`
- `lib/api-base-url.test.ts`
- `lib/chunk-items.test.ts`
- `lib/select-options.test.ts`

## Verification To Re-run

```bash
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun test
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH ./node_modules/.bin/tsc --noEmit
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun run lint
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH npm run build
```

## Operational Notes

- `bun dev` uses `postgres://nuchi:nuchi_dev_password@127.0.0.1:54329/nuchi_dev`.
- `bun dev` leaves the Docker Postgres container running after the Next process exits.
- Seed will fail unless explicitly run against local DB with `APP_ENV=local ALLOW_DB_SEED=true`.
- Current mutation rate limiting is in-process only; use a shared store before multi-instance production.

## Follow-Up

- Dependency advisory cleanup for Clerk, Hono, Drizzle, and Next remains separate from this branch.
- Consider route-level integration tests with authenticated fixtures for tenant ownership guarantees.
