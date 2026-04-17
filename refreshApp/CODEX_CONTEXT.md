# ProjectPolish Context

## Restart Point

- Branch: `ProjectPolish`
- Goal: establish a clean local-development baseline before further feature work.
- Local development should use Docker Postgres 17 by default.
- Neon remains available through `bun run dev:neon` and the existing `.env.local` flow.

## Memory Index

- [ProjectPolish PR Summary](./pr%20summaries/project-polish-pr-summary.md) - concise branch summary, verification commands, key files, and follow-up notes.

## Local Database

- Compose file: `docker-compose.dev.yml`
- Service: `nuchi-postgres`
- Image: `postgres:17-alpine`
- Local URL: `postgres://nuchi:nuchi_dev_password@127.0.0.1:54329/nuchi_dev`
- `bun dev` starts the container, waits for health, runs migrations, then starts Next.

## Safety Rules

- Do not commit `.env.local` or zipped env exports.
- Seed refuses to run unless `APP_ENV=local` and `ALLOW_DB_SEED=true`.
- Transaction writes must validate account/category ownership against `auth.userId`.
- Summary date filters must reject invalid dates and excessive ranges.
- Bulk transaction endpoints cap array sizes, guard `Content-Length`, and use a per-process mutation rate limit.

## Implemented Scope

- API runtime moved to Node.js so Drizzle can use `node-postgres`.
- `bun dev` targets the local Docker database through `scripts/dev-local.ts`.
- `bun run dev:neon` preserves the existing Neon-backed development path.
- `localEnv.zip` was removed and zip files are ignored.
- The debug `/asd` route was removed.
- Browser API calls now use same-origin URLs so local dev port fallback does not cause CORS errors when creating accounts.
- CSV transaction imports are chunked into the same batch size accepted by the bulk-create endpoint.
- Inline account/category creation inside transaction forms auto-selects the newly created option.
- `to=yyyy-MM-dd` filters are inclusive through the end of that calendar day, so newly created same-day transactions are not hidden by date filters.
- Remote Postgres TLS verifies certificates by default; `ALLOW_INSECURE_DATABASE_TLS=true` is required to opt out for known self-signed scenarios.
- The app Drizzle pool uses a `globalThis` cache outside production to avoid Next.js dev HMR creating multiple `pg.Pool` instances.

## Verification Commands

```bash
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun run lint
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH ./node_modules/.bin/tsc --noEmit
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun test
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun run build
PATH=/Users/gonzalo.secades/.nvm/versions/node/v22.20.0/bin:$PATH bun dev
```

## Last Verified

- Date: 2026-04-17
- `bun run lint`: passed.
- `./node_modules/.bin/tsc --noEmit`: passed.
- `bun test`: passed focused tests for transaction route utility guards, browser API base URL resolution, import chunking, select option merging, inclusive end-of-day filtering, Postgres TLS configuration, and dev pool caching.
- `npm run build`: passed.
- `bun dev`: pulled/started `postgres:17-alpine`, ran migrations, and booted Next dev. Port 3000 was occupied, so Next used `http://localhost:3001`.
- Docker state after smoke test: `nuchi-postgres-dev` healthy on host port `54329`.

## Follow-Up Memory

- The previous review found dependency advisories in Clerk, Hono, Drizzle, and Next. Those are outside this ProjectPolish implementation unless explicitly scheduled next.
- Current focus is local DB startup, tenant safety, bulk/date guardrails, seed safety, and a stable handoff state.
