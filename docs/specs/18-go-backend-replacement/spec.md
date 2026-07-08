# Go Backend Replacement Spec

Parent issue: [#18 Spec Go backend replacement for Hono/Drizzle/Neon](https://github.com/GonzaloSecades/nuchi/issues/18)

## Problem

Nuchi currently keeps the backend inside the Next.js app:

- Next.js App Router serves the frontend.
- Hono routes live under `app/api/[[...route]]`.
- Clerk provides auth and `auth.userId` ownership scope.
- Drizzle defines the PostgreSQL schema and query layer.
- Neon/Postgres is the database target.
- TanStack Query hooks call a typed Hono client from `lib/hono.ts`.

Issue #18 replaces that backend surface with a separate Go API and local Dockerized PostgreSQL while keeping frontend behavior stable.

## Goals

- Run a separate Go API during development.
- Keep browser API calls same-origin through `/api/*`, proxied by Next.js to the Go API.
- Make OpenAPI the API contract source of truth.
- Generate Go server types and TypeScript client/types from OpenAPI.
- Replace Hono RPC client wiring with generated OpenAPI client wiring.
- Preserve existing accounts, categories, transactions, summary, and hook behavior unless listed as an intentional product change.
- Replace Clerk with owned email/password auth, email verification, password reset, JWT access tokens, and refresh-token cookies.
- Move persistence to Go with `pgxpool`, `sqlc`, and `goose`.
- Use PostgreSQL RLS as a required ownership backstop.
- Keep transaction amounts as integer milliunits.

## Non-goals

- No backend implementation in this issue.
- No frontend implementation in this issue.
- No Docker, auth, OpenAPI, migration, dependency, or config changes in this issue.
- No production data migration is required by #18.
- No currency conversion UI or currency toggle UI in this replacement sprint.
- No feature redesign beyond the intentional changes listed below.

## Target Architecture

Runtime shape:

- Next.js continues to serve the app UI.
- A separate Go API listens locally on `localhost:8080` by default.
- Next rewrites/proxies same-origin `/api/*` requests to the Go API during local development.
- The frontend keeps TanStack Query hooks as the server-state boundary.
- The generated TypeScript API client replaces `lib/hono.ts` internals.
- The generated Go server types bind handlers to the OpenAPI contract.
- PostgreSQL runs through Docker Compose for local development.
- Go runs outside Docker for normal local development.
- The current `backend/` scaffold exposes only `/api/health`; auth, database access, migrations, resource handlers, and frontend integration remain child-issue work.

Backend shape:

- `chi` owns HTTP routing/middleware.
- `pgxpool` owns database pooling.
- `sqlc` owns typed SQL access.
- `goose` owns migrations.
- OpenAPI, migrations, SQL queries, and Go handlers stay separated so generated code can be replaced without rewriting business logic.

Legacy code handling:

- Existing Hono routes, Drizzle schema, and Clerk usage are reference material during migration.
- Move Drizzle schema/routes to `deprecated/legacy` while the Go backend reaches parity.
- Remove Hono, Drizzle, Neon, Clerk, Clerk UI, and `lib/hono.ts` only after Go parity is usable.

## API Contract

OpenAPI is the contract source of truth. It must cover:

- Auth endpoints: register, login, refresh, logout, verify email, request password reset, reset password.
- Accounts endpoints: list, get, create, update, delete, bulk delete.
- Categories endpoints: list, get, create, update, delete, bulk delete.
- Transactions endpoints: list with date/account filters, get, create, bulk create, update, delete, bulk delete.
- Summary endpoint: current dashboard summary response and query filters.
- Health endpoint for local verification.

Generation plan:

- Keep the hand-edited contract at `openapi/nuchi.openapi.json`.
- Keep Go generation config at `openapi/oapi-codegen.yaml`.
- Generate Go server types from OpenAPI for handler request/response shapes.
- Generate TypeScript client/types from OpenAPI for frontend calls.
- Keep generated artifacts out of hand-edited business logic.
- Generated Go types belong under `backend/internal/openapi/`.
- Generated TypeScript client/types belong under `lib/api/generated/`.
- Child issue #29 owns filling the full resource contract.
- Child issue #54 owns pinning generator tooling and producing the generated Go server types and TypeScript client after #29 fills the contract.
- Any contract change must update OpenAPI first, then regenerate Go and TypeScript outputs.

Parity rules:

- Treat [`api-parity-fixtures.md`](./api-parity-fixtures.md) as the current Hono API behavior reference.
- Preserve current JSON envelope shape where practical: success responses return `{ data: ... }`.
- Preserve explicit JSON API auth failures separately from browser redirects: API handlers return `401` with `{ "error": "Unauthorized" }`.
- Preserve current query filters: `from`, `to`, and `accountId` for transactions/summary.
- Preserve current 401/404 behavior for missing auth or missing owned resources.
- Preserve transaction bulk-create all-or-error validation and ownership checks.
- Preserve current bulk-delete behavior unless OpenAPI intentionally changes it: missing or unowned IDs are ignored and the response contains only deleted owned IDs.
- Preserve category as optional on transactions.
- Preserve oversized transaction bulk-create and bulk-delete request handling.
- Preserve transaction mutation rate limiting semantics or document an intentional replacement in the relevant child issue.
- Decide current mismatches explicitly in OpenAPI instead of inheriting them accidentally, especially category duplicate update returning `500` while category duplicate create returns `409`.

## Auth And Sessions

Owned auth replaces Clerk.

Data model:

- Users use internal UUID IDs.
- Email is the login identifier.
- Passwords are stored only as password hashes.
- Email verification and password reset tokens are stored server-side as one-time, expiring tokens.
- Refresh tokens are stored server-side so logout and token rotation can revoke sessions.

Session flow:

- Register creates a user and sends an email verification token.
- Login accepts verified email/password and returns a short-lived JWT access token.
- Initial dev access-token lifetime is 30 minutes and configurable.
- Refresh token is set in an HttpOnly cookie.
- Refresh rotates or renews the access token from the refresh cookie.
- Logout revokes the refresh token and clears the cookie.
- JWT payload carries the required user identity, so the app does not need `/api/auth/me` on load.

Email flows:

- Email verification marks the user verified after a valid token is submitted.
- Password reset request sends a reset token if the email is eligible.
- Password reset accepts a valid token and new password, then invalidates used/reset tokens.
- Local development uses the repo's existing Mailpit direction as reference, but concrete wiring belongs to the auth child issue.

Security requirements:

- Mutating endpoints require a valid access token.
- Refresh token cookie is HttpOnly and scoped to auth refresh/logout paths where practical.
- Access token is sent as Bearer auth on API calls.
- Auth-sensitive reads and writes must derive user identity from the verified token, not request body fields.
- Browser navigation to protected app pages may redirect to custom auth pages, but JSON API endpoints must keep explicit status/error responses.

## Database And RLS

Migration strategy:

- Port current Drizzle tables to PostgreSQL migrations using `goose`.
- Preserve current table and column names where practical.
- Keep local database reset destructive and explicit.
- No production data migration is required for this plan.
- `sqlc` queries become the typed DB boundary after migrations exist.

Current tables to preserve as parity baseline:

- `accounts`: `id`, `plaid_id`, `name`, `user_id`.
- `categories`: `id`, `plaid_id`, `name`, `user_id`.
- `transactions`: `id`, `amount`, `payee`, `notes`, `date`, `account_id`, `category_id`.
- Account and category names use case-insensitive per-user uniqueness today through Postgres `citext` and unique indexes.
- Keep account/category user indexes and transaction account/category indexes unless the migration child issue proves a better equivalent.

Target ownership model:

- `users.id` is an internal UUID.
- Accounts and categories are directly owned by `user_id`.
- Transactions are owned through their required account.
- Category remains optional, but when present must belong to the same user.
- Deletes keep current behavior where practical: account deletion cascades transactions; category deletion clears `category_id`.

RLS model:

- Enable RLS on user-owned tables.
- Requests set the authenticated user UUID in the database session/transaction before queries.
- Account/category policies allow access only when `user_id` matches the current app user.
- Transaction policies allow access only through an account owned by the current app user.
- Application-level ownership checks still return friendly 404/400 responses, but RLS is the security backstop.
- SQL used by `sqlc` should still include ownership predicates so tests can assert user-facing behavior without relying only on RLS failure modes.

## Currency And Money

- Transaction amounts remain signed integer milliunits.
- Do not store floats for money.
- Positive amounts are income; negative amounts are expenses.
- Add required `currency` on transactions.
- Default transaction currency is `ARS`.
- Summary math groups only values that are safe to aggregate by currency; until multi-currency UX exists, the API should keep the current single-currency behavior with `ARS` defaults.
- CSV import must require or default currency consistently and validate all rows before inserting.

## Parity Requirements

Accounts:

- List only the current user's accounts.
- Get/update/delete only owned accounts.
- Create account with unique user/name behavior.
- Bulk delete only owned account IDs.

Categories:

- List only the current user's categories.
- Get/update/delete only owned categories.
- Create category with unique user/name behavior.
- Bulk delete only owned category IDs.

Transactions:

- List owned transactions joined through owned accounts.
- Support `from`, `to`, and optional `accountId` filters.
- Default omitted `to` to current server time and omitted `from` to 30 days before current server time.
- Parse provided date filters as strict `yyyy-MM-dd`; `from` is start of day and `to` is end of day.
- Keep inclusive date filtering and the 366-day maximum range.
- Sort list responses by date descending.
- Create/update require an owned account and, when present, an owned category.
- Bulk create validates all rows and ownership before inserting.
- Bulk create accepts 1 to 500 rows and rejects numeric `Content-Length` above 1,000,000 bytes.
- Bulk delete deletes only owned transaction IDs, accepts 1 to 500 IDs, and rejects numeric `Content-Length` above 100,000 bytes.
- Transaction create, bulk-create, update, delete, and bulk-delete keep the current per-user/action mutation limit unless intentionally replaced.
- Keep account required and category optional.

Summary:

- Preserve dashboard totals: remaining, income, expenses, category breakdown, daily income/expense series, and change percentages.
- Preserve date-range and account filters.
- Keep strict date validation and range limits from the current route behavior.
- Preserve previous-period comparison behavior and `calculatePercentageChange` semantics.
- Preserve category aggregation behavior: negative categorized transactions only, top 3 categories returned directly, remaining expense categories grouped into `Other`.
- Preserve daily series behavior: include every selected calendar day and fill missing income/expense with `0`.

CSV import:

- Keep CSV parsing as a frontend flow that posts validated rows to transaction bulk-create.
- Keep required mapped fields: `amount`, `date`, and `payee`.
- Keep amount conversion with `Math.round(amount * 1000)`.
- Keep CSV date input format `yyyy-MM-dd HH:mm:ss` and API date output `yyyy-MM-dd`.
- Keep chunking bulk-create requests at 500 transactions.
- Empty validated CSV data should not produce API calls.

Frontend:

- Preserve public behavior of TanStack Query hooks.
- Replace hook internals to call the generated OpenAPI client.
- Keep same-origin `/api/*` calls from the browser.

## Intentional Product Changes

- Auth changes from Clerk to owned email/password auth.
- Auth pages become custom app pages.
- User IDs change from Clerk IDs to internal UUIDs.
- Access tokens become app-issued JWTs.
- Refresh tokens move to HttpOnly cookies.
- OpenAPI becomes the contract source of truth.
- Transaction currency becomes required and defaults to `ARS`.
- Local development uses Dockerized PostgreSQL and a separately running Go API.

## Local Development Workflow

Target commands from #18:

- Start database: `docker compose up -d postgres`
- Run backend migrations: `cd backend && goose up`
- Run Go tests: `cd backend && go test ./...`
- Run Go API: `cd backend && go run ./cmd/api`
- Check health: `curl http://localhost:8080/api/health`
- Run frontend lint/build: `bun run lint` and `bun run build`

Workflow:

1. Start local PostgreSQL.
2. Apply backend migrations.
3. Run the Go API outside Docker.
4. Run Next.js with rewrite/proxy configuration for `/api/*`.
5. Use Mailpit or the documented local SMTP target for auth emails.
6. Use explicit reset command only when local data should be destroyed.

## Implementation Order

1. Write and review this spec.
2. Add the CI workflow so every later ticket proves the same checks.
3. Document current API parity fixtures.
4. Add OpenAPI scaffold and generation command documentation.
5. Define the shared API error/auth contract.
6. Define the full OpenAPI contract.
7. Pin generator tooling and generate Go server types and the TypeScript client.
8. Wire the Go database foundation.
9. Add auth tables, finance tables, RLS, and local dev scripts.
10. Add `sqlc` queries.
11. Implement password auth and JWT sessions.
12. Bind authenticated requests to RLS-backed DB access.
13. Implement email verification and password reset (parallelizable with resource parity).
14. Implement accounts, categories, transactions, bulk transactions, and summary parity.
15. Replace frontend rewrite/client/hook/auth-page internals.
16. Remove legacy Hono/Drizzle/Neon/Clerk code after parity is verified.

## Verification

Documentation-only issue #28:

- `bun run lint`

Replacement sprint verification targets:

- `bun run lint`
- `bun run build`
- `cd backend && go test ./...`
- `cd backend && go run ./cmd/api`
- `docker compose up -d postgres`
- `cd backend && goose up`
- `curl http://localhost:8080/api/health`

## Rollout

- Keep legacy backend code available as reference until generated client and Go routes reach parity.
- Switch frontend hooks route by route behind normal branch review, not by changing product behavior.
- Verify accounts, categories, transactions, summary, auth, and CSV import before removing legacy code.
- Remove legacy dependencies and code only in the cleanup child issue after parity is usable.

## Risks

- Auth replacement is high risk because it replaces Clerk session, email, and UI behavior.
- RLS setup is high risk because an incorrect policy can leak or hide user data.
- OpenAPI drift is high risk unless generation is mandatory after contract edits.
- Frontend client replacement is high risk because hooks must preserve current behavior.
- Currency is easy to under-specify; keep `ARS` default and milliunits invariant until real multi-currency UX exists.
- Local reset workflow can destroy data; destructive reset must stay explicit.

## Child Issue Plan

Work the tickets in this sequence. A ticket must be merged before the next ticket starts, with two exceptions: seq 00 (#53, CI) is unblocked and may run at any time, and seq 11 (#42, email flows) may run in parallel with seq 12-16 because both only depend on #41. Only the next unblocked ticket may be considered for `agent:ready`, and only if it is `risk:low`; high-risk tickets remain attended work.

Completed prerequisites:

| Issue | Title | Status |
| --- | --- | --- |
| [#19](https://github.com/GonzaloSecades/nuchi/issues/19) | Scaffold Go backend service and health route | closed |
| [#20](https://github.com/GonzaloSecades/nuchi/issues/20) | Add Docker Compose Postgres and local mail catcher | closed |
| [#34](https://github.com/GonzaloSecades/nuchi/issues/34) | Document current API parity fixtures | closed |
| [#35](https://github.com/GonzaloSecades/nuchi/issues/35) | Add OpenAPI scaffold and generation commands | closed |
| [#28](https://github.com/GonzaloSecades/nuchi/issues/28) | Finalize Go backend replacement spec | closed |
| [#36](https://github.com/GonzaloSecades/nuchi/issues/36) | Define shared API error and auth contract | closed |

Active sequence:

| Seq | Issue | Title | Risk | Dependency | Agent-ready |
| --- | --- | --- | --- | --- | --- |
| 00 | [#53](https://github.com/GonzaloSecades/nuchi/issues/53) | Add CI workflow for Go tests, frontend lint/build, and OpenAPI validation | low | none | yes |
| 03 | [#29](https://github.com/GonzaloSecades/nuchi/issues/29) | Define full OpenAPI contract | high | #36 | no |
| 03b | [#54](https://github.com/GonzaloSecades/nuchi/issues/54) | Pin OpenAPI generator tooling and generate Go server types and TypeScript client | low | #29 | no while blocked |
| 04 | [#37](https://github.com/GonzaloSecades/nuchi/issues/37) | Wire Go database foundation | high | #54 | no |
| 05 | [#38](https://github.com/GonzaloSecades/nuchi/issues/38) | Create base SQL migrations with users and auth tables | high | #37 | no |
| 06 | [#39](https://github.com/GonzaloSecades/nuchi/issues/39) | Create finance schema migrations with RLS | high | #38 | no |
| 07 | [#31](https://github.com/GonzaloSecades/nuchi/issues/31) | Add backend dev scripts and explicit DB reset workflow | low | #39 | no while blocked |
| 08 | [#40](https://github.com/GonzaloSecades/nuchi/issues/40) | Add sqlc queries for auth and owned resources | high | #31 | no |
| 09 | [#41](https://github.com/GonzaloSecades/nuchi/issues/41) | Implement password auth and JWT sessions | high | #40 | no |
| 10 | [#43](https://github.com/GonzaloSecades/nuchi/issues/43) | Add Go auth middleware and DB RLS session binding | high | #41 | no |
| 11 | [#42](https://github.com/GonzaloSecades/nuchi/issues/42) | Implement email verification and password reset | high | #41; may run parallel to 12-16 | no |
| 12 | [#44](https://github.com/GonzaloSecades/nuchi/issues/44) | Implement accounts API parity | high | #43 | no |
| 13 | [#45](https://github.com/GonzaloSecades/nuchi/issues/45) | Implement categories API parity | high | #44 | no |
| 14 | [#46](https://github.com/GonzaloSecades/nuchi/issues/46) | Implement transactions CRUD and list parity | high | #45 | no |
| 15 | [#47](https://github.com/GonzaloSecades/nuchi/issues/47) | Implement transaction bulk create and delete parity | high | #46 | no |
| 16 | [#48](https://github.com/GonzaloSecades/nuchi/issues/48) | Implement summary API parity | high | #47 | no |
| 17 | [#30](https://github.com/GonzaloSecades/nuchi/issues/30) | Add Next rewrite shape for same-origin Go API calls | low | #48 | no while blocked |
| 18 | [#49](https://github.com/GonzaloSecades/nuchi/issues/49) | Replace frontend generated client wiring | high | #30 | no |
| 19 | [#50](https://github.com/GonzaloSecades/nuchi/issues/50) | Migrate TanStack hooks to generated client | high | #49 | no |
| 20 | [#51](https://github.com/GonzaloSecades/nuchi/issues/51) | Replace Clerk auth pages with custom auth pages | high | #50 | no |
| 21 | [#27](https://github.com/GonzaloSecades/nuchi/issues/27) | Remove legacy Hono/Drizzle/Neon/Clerk backend after Go parity | high, blocked | #51 | no |

Superseded duplicates:

| Issue | Replaced by |
| --- | --- |
| [#21](https://github.com/GonzaloSecades/nuchi/issues/21) | #37, #38, #39, #40, #43 |
| [#22](https://github.com/GonzaloSecades/nuchi/issues/22) | #38, #41, #42, #43, #51 |
| [#23](https://github.com/GonzaloSecades/nuchi/issues/23) | #44, #45 |
| [#24](https://github.com/GonzaloSecades/nuchi/issues/24) | #46, #47 |
| [#25](https://github.com/GonzaloSecades/nuchi/issues/25) | #48 |
| [#26](https://github.com/GonzaloSecades/nuchi/issues/26) | #49, #50, #51 |
