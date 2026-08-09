# Nuchi Project Context

## Overview

Nuchi is a personal finance application built around a small set of user-scoped domains:

- accounts
- categories
- transactions
- summary analytics

The app uses Next.js App Router for the UI shell and TanStack Query for
client-side server state. All server behavior lives in a separate Go API; Next
serves no API routes of its own.

## Core Stack

- Next.js 16 App Router (frontend only)
- React 19
- TypeScript strict mode
- Bun
- Go API (chi, pgxpool, sqlc, goose) over PostgreSQL, in `backend/`
- Owned JWT auth in `lib/auth/`, with the API contract in `openapi/nuchi.openapi.json`
- Generated OpenAPI client in `lib/api/`
- TanStack Query
- Tailwind CSS v4
- shadcn/ui
- Recharts

## Canonical Architecture

### Runtime Shape

- App pages and layouts live under `app/`. There are no Next API routes.
- `/api/*` is rewritten same-origin to the Go API by `next.config.ts`
  (`beforeFiles`), so cookies stay first-party and no CORS is involved.
- Domain handlers own server behavior, in `backend/internal/http/`:
  - `accounts.go`
  - `categories.go`
  - `transactions.go`, `transactions_bulk.go`
  - `summary.go`

### Client-to-Database Flow

1. A page renders a dashboard or feature-specific component.
2. The component calls a TanStack Query hook from `features/<domain>/api/` or `features/summary/`.
3. The hook calls the generated OpenAPI client in `lib/api/`, which attaches the access token and transparently refreshes it once on expiry.
4. Next rewrites the same-origin `/api/*` request to the Go API.
5. The Go handler validates input, derives the user from the verified JWT, binds the RLS session, and runs sqlc queries.
6. The API returns a normalized JSON payload, usually under `{ data }` on success.
7. The hook adapts payload details for presentation, including milliunit-to-display conversions.

### Layer Responsibilities

- `app/`
  - routing, layouts, page composition
- `features/`
  - domain hooks, forms, sheets, and feature-specific UI state
- `components/`
  - shared UI controls, tables, filters, and chart components
- `backend/`
  - the Go API: handlers, sqlc queries, migrations, auth
- `lib/`
  - generated API client and auth, API error helpers, date/amount utilities
- `providers/`
  - app-wide React Query and sheet wiring

## Current Domain Model

### Accounts

Fields:
- `id`
- `plaidId`
- `name`
- `userId`

Rules:
- user-scoped
- case-insensitive uniqueness on `(userId, name)`

### Categories

Fields:
- `id`
- `plaidId`
- `name`
- `userId`

Rules:
- user-scoped
- case-insensitive uniqueness on `(userId, name)`

### Transactions

Fields:
- `id`
- `amount`
- `payee`
- `notes`
- `date`
- `accountId`
- `categoryId`

Rules:
- linked to an account
- optionally linked to a category
- stored in milliunits
- account deletion cascades to transactions
- category deletion sets `categoryId` to `null`

## Feature Guide

### Dashboard Summary

Primary files:
- `app/(dashboard)/page.tsx`
- `features/summary/use-get-summary.ts`
- `backend/internal/http/summary.go`
- `components/data-grid.tsx`
- `components/data-charts.tsx`

Behavior:
- loads KPI metrics for income, expenses, and remaining balance
- supports account and date filtering through URL params
- returns chart-ready day-series and category aggregates
- renders multiple visualization variants with shared chart controls

### Accounts

Primary files:
- `app/(dashboard)/accounts/`
- `features/accounts/api/`
- `features/accounts/components/`
- `backend/internal/http/accounts.go`

Behavior:
- list, create, edit, delete, and bulk delete accounts
- open create/edit flows in sheets
- invalidate account-dependent queries after mutations

### Categories

Primary files:
- `app/(dashboard)/categories/`
- `features/categories/api/`
- `features/categories/components/`
- `backend/internal/http/categories.go`

Behavior:
- list, create, edit, delete, and bulk delete categories
- enforce duplicate-name protection per user
- propagate category changes to dependent views through query invalidation

### Transactions

Primary files:
- `app/(dashboard)/transactions/`
- `features/transactions/api/`
- `features/transactions/components/`
- `backend/internal/http/transactions.go`

Behavior:
- list transactions with account/date filtering
- create, edit, delete, and bulk delete transactions
- associate transactions with accounts and categories
- render amount badges, date pickers, and account/category selects

### CSV Import

Primary files:
- `app/(dashboard)/transactions/import-card.tsx`
- `app/(dashboard)/transactions/import-table.tsx`
- `app/(dashboard)/transactions/upload-button.tsx`
- `features/accounts/hooks/use-select-account.tsx`

Behavior:
- upload CSV data client-side
- map incoming columns to transaction fields
- preview transformed rows before submission
- choose a destination account before bulk creation

### Shared Filters

Primary files:
- `components/account-filter.tsx`
- `components/date-filter.tsx`
- `components/filters.tsx`
- `components/header.tsx`

Behavior:
- keep `accountId`, `from`, and `to` in the URL
- drive both transaction and summary queries from the same filter state
- expose preset ranges and manual date selection

## Cross-Cutting Conventions

- Call the API through the generated client in `lib/api/`, never direct `fetch`.
- Keep server-state logic in TanStack Query hooks.
- Keep reusable primitives in `components/ui/`.
- Keep domain-specific UI under `features/<domain>/`.
- Use `openapi/nuchi.openapi.json` as the source of truth for data shape; both sides regenerate from it.
- Keep amounts in milliunits at the API and database layers.
- Scope auth-sensitive reads and writes by authenticated user ownership.
- Avoid `any` in app code.

## API Shape

Current registered route groups:
- `/api/accounts`
- `/api/categories`
- `/api/transactions`
- `/api/summary`

General pattern:
- authenticated routes sit behind the Go `RequireAuth` middleware and derive the user from the verified JWT
- handlers validate params/query/body in Go against the contract
- successful responses typically return `{ data }`
- client hooks normalize non-OK responses through `createApiError()`

## Data and Query Patterns

- `lib/api/client.ts` exposes the generated client; types come from `lib/api/generated/schema.d.ts`
- query keys are domain-scoped and include active filters where needed
- mutations invalidate related queries to keep dashboard and lists in sync
- display components convert milliunits close to the rendering boundary

## Operational Notes

When changing schema:
- add a goose migration under `backend/migrations/`
- update the sqlc queries and regenerate
- verify related hooks and API contracts

When changing API contracts:
- update `openapi/nuchi.openapi.json` first, then regenerate both sides
- update the Go handler in `backend/internal/http/`
- update the matching TanStack Query hooks
- update forms, tables, and charts that consume that data

When changing transaction amount logic:
- keep conversions centralized
- do not mix display-unit amounts with stored milliunits

## Known Active Debt

The canonical debt tracker remains:

- `PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`

Known themes already called out there include:
- transaction ownership validation gaps in create and bulk-create flows
- summary route validation and cleanup
- CSV import validation hardening
- avoiding tighter coupling between header filters and summary loading
