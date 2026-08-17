# Current State and Constraints

## Rebaseline record

This snapshot replaces the mid-migration description. It was taken on
2026-08-17 at `f177ade` after the Go migration and legacy teardown had landed.
The evidence came from the runtime router and middleware, every handler,
sqlc source queries, migrations 00001-00005, the OpenAPI source, tests, and
module-scoped graphify queries. Generated files were used only to confirm
contract wiring.

The shipped system is now one architecture: Next.js serves the frontend and
proxies same-origin `/api/*` requests to the Go API. The Go API owns auth and
all database access through chi, pgxpool, sqlc, goose, and PostgreSQL 17. There
is no Hono, Drizzle, Clerk, or Next API runtime left.

## Request and trust flow

`cmd/api/main.go` validates configuration, opens and pings the pool, refuses
to serve when the connection role bypasses RLS, builds the auth and resource
servers, and starts an `http.Server` with 5s header, 15s read, 30s write, and
120s idle timeouts. SIGINT/SIGTERM gets a 10s graceful-shutdown window.

`NewRouter` installs request IDs, real-IP extraction, and panic recovery.
Health and seven auth commands are public. Every finance route is in one
`RequireAuth` group. That middleware accepts only an owned HS256 access token,
derives a UUID principal from its verified subject, and stores it under an
unexported context key. No operation accepts ownership from a body, query
field, or caller-selected header.

Finance handlers call `ResourceServer.withUser`, which calls
`db.WithUserTxOptions`. The helper begins a transaction, binds
`app.user_id` with transaction-local `set_config(..., true)`, exposes only
transaction-bound sqlc queries, and commits or rolls back. Migrations force
RLS on accounts, categories, and transactions. SQL ownership predicates are
still present, and transaction writes additionally prove category ownership.
Live tests cover cross-user reads/writes, the unbound fail-closed case,
rollback, repeatable-read snapshot behavior, and connection reuse.

## Operation inventory

The OpenAPI source defines 29 operation IDs. “Owner” is the code and policy
owner for follow-up work; the phase owner map below assigns the outstanding
risk. Query counts are the current handler path's database statement class,
not an SLO. Validation or rejection can stop earlier.

| operationId              | Route                                   | Auth policy                             | Owner         | Current database work                                         |
| ------------------------ | --------------------------------------- | --------------------------------------- | ------------- | ------------------------------------------------------------- |
| `getHealth`              | `GET /api/health`                       | Public                                  | Platform HTTP | none                                                          |
| `registerUser`           | `POST /api/auth/register`               | Public, enumeration-safe conflict       | Auth/session  | bounded multi-statement transaction plus mail handoff         |
| `loginUser`              | `POST /api/auth/login`                  | Public, password verification           | Auth/session  | user lookup plus refresh-token insert                         |
| `refreshSession`         | `POST /api/auth/refresh`                | HttpOnly refresh cookie                 | Auth/session  | atomic token consume and rotation transaction                 |
| `logoutUser`             | `POST /api/auth/logout`                 | HttpOnly refresh cookie                 | Auth/session  | one revocation command                                        |
| `verifyEmail`            | `POST /api/auth/verify-email`           | Opaque one-time token                   | Auth/session  | atomic consume and user update transaction                    |
| `requestPasswordReset`   | `POST /api/auth/password-reset/request` | Public, generic response                | Auth/session  | bounded lookup/rate-check/token transaction plus mail handoff |
| `confirmPasswordReset`   | `POST /api/auth/password-reset/confirm` | Opaque one-time token                   | Auth/session  | atomic consume, password update, and session revocation       |
| `listAccounts`           | `GET /api/accounts`                     | Bearer principal + RLS                  | Accounts      | 1 query                                                       |
| `createAccount`          | `POST /api/accounts`                    | Bearer principal + RLS                  | Accounts      | 1 query                                                       |
| `bulkDeleteAccounts`     | `POST /api/accounts/bulk-delete`        | Bearer principal + RLS                  | Accounts      | bounded ownership/delete workflow                             |
| `getAccount`             | `GET /api/accounts/{id}`                | Bearer principal + RLS                  | Accounts      | 1 query                                                       |
| `updateAccount`          | `PATCH /api/accounts/{id}`              | Bearer principal + RLS                  | Accounts      | 1 query                                                       |
| `deleteAccount`          | `DELETE /api/accounts/{id}`             | Bearer principal + RLS                  | Accounts      | 1 query plus database cascade                                 |
| `listCategories`         | `GET /api/categories`                   | Bearer principal + RLS                  | Categories    | 1 query                                                       |
| `createCategory`         | `POST /api/categories`                  | Bearer principal + RLS                  | Categories    | 1 query                                                       |
| `bulkDeleteCategories`   | `POST /api/categories/bulk-delete`      | Bearer principal + RLS                  | Categories    | bounded ownership/delete workflow                             |
| `getCategory`            | `GET /api/categories/{id}`              | Bearer principal + RLS                  | Categories    | 1 query                                                       |
| `updateCategory`         | `PATCH /api/categories/{id}`            | Bearer principal + RLS                  | Categories    | 1 query                                                       |
| `deleteCategory`         | `DELETE /api/categories/{id}`           | Bearer principal + RLS                  | Categories    | 1 query plus `SET NULL` fan-out                               |
| `listTransactions`       | `GET /api/transactions`                 | Bearer principal + RLS                  | Transactions  | 1 query, unpaginated                                          |
| `createTransaction`      | `POST /api/transactions`                | Bearer principal + RLS + mutation limit | Transactions  | reference checks plus 1 insert                                |
| `bulkCreateTransactions` | `POST /api/transactions/bulk-create`    | Bearer principal + RLS + mutation limit | Transactions  | set-based reference checks plus 1 JSONB insert                |
| `bulkDeleteTransactions` | `POST /api/transactions/bulk-delete`    | Bearer principal + RLS + mutation limit | Transactions  | bounded ownership/delete workflow                             |
| `getTransaction`         | `GET /api/transactions/{id}`            | Bearer principal + RLS                  | Transactions  | 1 query                                                       |
| `updateTransaction`      | `PATCH /api/transactions/{id}`          | Bearer principal + RLS + mutation limit | Transactions  | current/new reference checks plus 1 update                    |
| `deleteTransaction`      | `DELETE /api/transactions/{id}`         | Bearer principal + RLS + mutation limit | Transactions  | 1 query                                                       |
| `getSummary`             | `GET /api/summary`                      | Bearer principal + RLS                  | Summary       | 4 queries in one repeatable-read transaction                  |

There are no operations with an unknown authentication policy. Public means
intentionally public; it does not mean unbounded or caller-owned identity.

## Module review

| Module              | Shipped evidence                                                                                                                                                            | Current constraint                                                                                                                                             |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Platform/HTTP       | Explicit chi routes, request ID, real IP, recovery, timeouts, graceful shutdown, startup pool/RLS checks                                                                    | `RealIP` has no deployment-specific trusted-proxy policy; health is liveness and readiness combined; errors are still partly handler-local                     |
| Auth/session        | Argon2id password hashing with bounded concurrency, owned JWTs, hashed one-time/refresh tokens, atomic consume paths, secure-cookie configuration, redacted external errors | One HS256 key, no midlife access-token revocation or refresh-reuse detection, process-local abuse limits, and non-durable mail handoff remain registry work    |
| Accounts            | Deterministic list, case-insensitive uniqueness, explicit ownership predicates, forced RLS                                                                                  | Bulk-delete response and large cascade cost remain product/performance decisions                                                                               |
| Categories          | Account-parallel ownership and conflict handling; transaction write policy rejects cross-owner category references                                                          | Large `ON DELETE SET NULL` fan-out needs measurement; the historical duplicate-update pattern needs a final sibling audit                                      |
| Transactions        | Signed bigint milliunits, set-based bulk insert, 500-row/byte limits for bulk paths, deterministic date/id ordering, reference validation and RLS backstop                  | List is unpaginated; date is still an ambiguous timestamp; text IDs, process-local mutation limits, and idempotency remain open                                |
| Summary             | Four RLS-scoped queries share one repeatable-read snapshot; semantics preserve current/previous totals, top-three-plus-Other, zero-filled days, and account/date filters    | Four scans are expensive at the maximum range, uncategorized spending is excluded from the chart, and accepted aggregate range is not explicit in the contract |
| Database/migrations | Five goose migrations, sqlc source queries, forced finance RLS, startup role verification, migration-upgrade and live RLS suites                                            | Local Compose uses one ordinary app role for runtime and migrations; production needs separate credentials and a migration runbook                             |

## Registry reconciliation

The numbered registry was checked against the shipped schema and contract.
“Gated” means this program links to the named registry ticket and does not
reimplement it.

| Entry | Shipped behavior                                                     | Status / owner                                                        |
| ----- | -------------------------------------------------------------------- | --------------------------------------------------------------------- |
| 0001  | `transactions.date` remains `timestamp without time zone`            | Open; gated on #112/#123/#127                                         |
| 0002  | Finance IDs remain text CUID2; user/token UUID defaults are v4       | Open; gated on #113                                                   |
| 0003  | Transaction mutation limit is process-local                          | Open; gated on #114                                                   |
| 0004  | Bulk delete silently ignores missing/unowned IDs                     | Open; gated on #115                                                   |
| 0005  | Duplicate category update maps to `409`                              | Delivered in #45; Phase 5 performs only the sibling-pattern audit     |
| 0006  | Amount storage is bigint and API values are exact JS-safe milliunits | Delivered in #46/00005                                                |
| 0007  | Access JWTs use one static HS256 secret                              | Open; gated on #116                                                   |
| 0008  | Access tokens cannot be revoked before expiry                        | Open; #117                                                            |
| 0009  | Refresh reuse detection/session listing are absent                   | Open; #118                                                            |
| 0010  | Auth operations do not all declare `500`                             | Open; gated on #119                                                   |
| 0011  | Mail handoff is asynchronous but not durable; no resend endpoint     | Open; gated on #120                                                   |
| 0012  | Reset-request timing still depends on the user-exists path           | Open; gated on #121                                                   |
| 0013  | Non-bulk resource bodies have no streamed byte cap                   | Open; gated on #122                                                   |
| 0014  | Date filters resolve in UTC                                          | Open only as part of #112/#123/#127                                   |
| 0015  | JS-safe amount overflow is a documented `400`                        | Correct shipped behavior; #124 owns residual contract-generation work |
| 0016  | Bulk byte caps are stream-enforced but live in prose/constants       | Correct shipped behavior; gated on #125                               |
| 0017  | Category spending excludes uncategorized expenses                    | Open product decision; gated on #126                                  |
| 0018  | Responses serialize the timestamp as a UTC instant                   | Open only as part of #112/#123/#127                                   |

## Versioned performance baseline

The checked-in
[`backend/testdata/performance-baseline-v1.sql`](../../backend/testdata/performance-baseline-v1.sql)
creates two synthetic tenants, six owner accounts, 12 categories, 100,000
owner transactions, and 5,000 neighbor transactions over two years. It has no
real personal data and is deterministic. The companion
[`performance-baseline-v1-plans.sql`](../../backend/testdata/performance-baseline-v1-plans.sql)
runs as the ordinary `nuchi` role with transaction-local RLS identity.

Reproduce against disposable local Compose data:

```bash
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d nuchi \
  < backend/testdata/performance-baseline-v1.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d nuchi \
  < backend/testdata/performance-baseline-v1-plans.sql
```

The 2026-08-17 warm-cache sample used PostgreSQL 17 Alpine with all five
migrations applied. These are evidence, not approved SLOs:

| Work                                |   Rows | Query count | Execution time | Plan observation                                                                             |
| ----------------------------------- | -----: | ----------: | -------------: | -------------------------------------------------------------------------------------------- |
| Account list                        |      6 |           1 |       0.323 ms | Small-table sequential scan and in-memory sort                                               |
| Category list                       |     12 |           1 |       0.189 ms | Small-table sequential scan and in-memory sort                                               |
| Transaction list, max 366-day range | 50,142 |           1 |     684.367 ms | Uses `(account_id, date)` bitmap index, then spills an 8.5 MB external sort for `(date, id)` |
| Summary current totals              | 50,142 |      1 of 4 |     370.712 ms | Six indexed account/date bitmap scans under RLS                                              |
| Summary previous totals             | 24,795 |      1 of 4 |     185.372 ms | Same plan shape on the previous range                                                        |
| Summary category spending           | 40,795 |      1 of 4 |     461.629 ms | Indexed scan, category join, hash aggregate                                                  |
| Summary daily totals                | 50,142 |      1 of 4 |     379.699 ms | Indexed scan and 366-group hash aggregate                                                    |

The transaction list returns every matching row and its requested ordering is
not covered by the current index. Summary spends roughly 1.40 seconds in four
database statements on this sample before network and Go shaping. Those are
the first measured priorities for Phase 3. No index change is justified by
this single host sample alone; Phase 3 must store cold/warm repetitions and
concurrency results before changing schema or pool limits.

## Risk register and owner map

| Priority | Risk                                                              | Evidence / gate                                           | Owner                                     |
| -------- | ----------------------------------------------------------------- | --------------------------------------------------------- | ----------------------------------------- |
| P0       | Proxy-derived client IP is trusted without a deployment allowlist | `middleware.RealIP`; needed before rate limits rely on it | Phase 1 #107, deploy #66                  |
| P0       | Non-bulk resource bodies are unbounded                            | Registry 0013                                             | Claude #122; Phase 1 gates on it          |
| P0       | Mail can be lost after commit and before SMTP succeeds            | Registry 0011                                             | Claude #120; Phase 2 gates on it          |
| P0       | Runtime/migrator credential separation is not production-proven   | Compose and startup RLS check                             | Phase 1 #107, deploy #66                  |
| P1       | One static JWT key has no rotation path                           | Registry 0007                                             | Claude #116; Phase 1 gates on it          |
| P1       | Process-local mutation limits diverge with multiple instances     | Registry 0003                                             | Claude #114; Phase 1/5 gate on it         |
| P1       | Transaction list is unpaginated and spills at realistic max range | baseline-v1 plan                                          | Phase 3 #109                              |
| P1       | Summary performs four repeated RLS-scoped scans                   | baseline-v1 plans                                         | Phase 3 #109; range proposal #111         |
| P1       | Readiness is not distinct from liveness                           | one `/api/health` handler                                 | Phase 4 #110 and deploy #66               |
| P1       | Stable telemetry/redaction interfaces are absent                  | direct `slog` call sites                                  | Phase 4 #110                              |
| P2       | Date/timestamp and text-ID models carry compatibility cost        | Registry 0001/0002/0014/0018                              | Claude #112/#113/#123/#127; Phase 5 gates |
| P2       | Category chart and bulk-delete semantics need product decisions   | Registry 0004/0017                                        | Claude #115/#126; Phase 5 gates           |

## Baseline conclusion

The security foundation is substantially stronger than the old snapshot:
principal derivation, RLS request binding, ownership predicates, atomic token
paths, strict validation, bounded bulk streams, server timeouts, and live
cross-user tests are shipped. The next work is not migration completion. It is
cross-cutting policy consolidation, fault/resilience evidence, measured query
work, operational contracts, and deliberate behavior/schema proposals.

No downstream phase should use the old Hono behavior as a current-state fact.
OpenAPI is the observable contract; the Go implementation and migrations are
the current internal behavior; the registry records deliberate follow-up.
