# Nuchi

Nuchi is a personal finance app for tracking accounts, categories, transactions, CSV imports, and dashboard analytics. The codebase is organized as a typed full-stack Next.js application with a Hono API layer, Clerk authentication, Drizzle ORM, and Neon Postgres.

## Stack
- Next.js 16 App Router
- React 19
- TypeScript with strict typing
- Bun package manager
- Clerk authentication
- Hono API mounted in `app/api/[[...route]]`
- TanStack Query for server-state management
- Drizzle ORM with Neon Postgres
- Tailwind CSS v4
- shadcn/ui primitives
- Recharts for dashboard analytics

## Product Surface
- Dashboard analytics with KPI cards, period comparisons, category breakdowns, and daily charts
- Account management with CRUD and bulk delete
- Category management with CRUD, bulk delete, and case-insensitive uniqueness
- Transaction management with CRUD, bulk actions, account/category assignment, and date filtering
- CSV transaction import with column mapping and account assignment
- Global account and date filters scoped through URL query params

## Canonical Docs
- Main architecture and feature guide: `docs/CODEX_CONTEXT.md`
- Agent and repo rules: `AGENTS.md`
- Active debt tracker: `PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`

## Local Setup
1. Install dependencies.

```bash
bun install
```

2. Copy environment variables from `.env.example` into `.env.local`.

3. Run database migrations.

```bash
bun run db:migrate
```

4. Start the app.

```bash
bun dev
```

The app runs at `http://localhost:3000`.

## Environment Variables
- `DATABASE_URL`
- `NEXT_PUBLIC_API_URL`
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `CLERK_SECRET_KEY`

See `.env.example` for the expected local shape.

## Common Commands
```bash
bun dev
bun run lint
bun run build
bun run db:generate
bun run db:migrate
```

## Project Structure
- `app/`: App Router pages, layouts, and the Hono API entrypoint
- `features/`: domain slices for accounts, categories, summary, and transactions
- `components/`: shared UI, filters, tables, and chart components
- `db/`: Drizzle schema and database client
- `drizzle/`: generated SQL migrations and schema snapshots
- `lib/`: typed Hono client, error helpers, and shared utilities
- `providers/`: React Query and sheet providers
- `scripts/`: migration and seed tooling

## Development Notes
- Prefer extending typed Hono routes and TanStack Query hooks instead of ad hoc `fetch`.
- Scope auth-sensitive reads and writes by `auth.userId`.
- Transaction amounts are stored in milliunits.
- Keep `db/schema.ts` as the schema source of truth and keep migrations in sync with it.
