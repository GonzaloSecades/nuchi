# Current State and Constraints

## Snapshot

The repository is between two backend architectures.

The current production-shaped backend is Hono inside Next.js. It uses Clerk
identity, Drizzle queries, direct ownership predicates, Zod validation, and
TanStack Query clients. Accounts and categories are directly scoped by
`userId`; transactions are scoped through their account. The summary endpoint
runs separate current-period, previous-period, category, and daily-series
queries. Transaction mutations add count limits, body-size checks based on
`Content-Length`, reference ownership checks, and an in-memory rate limiter.

The future backend is a separate Go API using chi, pgxpool, sqlc, goose, strict
OpenAPI-generated server types, owned JWT/password auth, and PostgreSQL RLS.
At this snapshot, the checked-in Go runtime still wires only health, pool
lifecycle, graceful shutdown, schema migrations, RLS policies, and generated
OpenAPI bindings. Resource handlers, auth middleware, RLS request binding,
sqlc query implementations, and most operational middleware are planned or
in progress rather than present in the runtime router.

## Existing strengths to preserve

- The OpenAPI 3.0.3 document covers health, auth, accounts, categories,
  transactions, bulk operations, and summary, and protects resource endpoints
  with Bearer auth.
- Generated code is isolated from hand-written business logic.
- Finance RLS is enabled and forced, so the table-owning API role is not
  exempt. Missing `app.user_id` fails closed.
- Application ownership predicates are required in addition to RLS.
- Transaction amounts remain signed integer milliunits.
- Bulk-create reference validation is set-based rather than one lookup per
  input row.
- Transaction access uses the composite `(account_id, date DESC)` index in the
  Go migration, matching the primary list pattern better than the legacy
  single-column index.
- The API parity fixtures document edge behavior, limits, errors, filtering,
  and response shapes in unusual detail.

## Gaps this project must close

| Area | Current evidence | Post-migration need |
| --- | --- | --- |
| Security | Hono has handler-local Clerk checks; Go has forced RLS but no request binding yet | Central verified principal, transaction-local RLS identity, layered authorization tests, secure cookie/origin handling, shared abuse controls |
| Performance | Useful baseline indexes and set-based bulk validation exist | Measured SLOs, pool tuning, query budgets, pagination, production-shaped EXPLAIN plans, regression benchmarks |
| Robustness | Strict date/range limits, graceful shutdown, and parity fixtures exist | Bounded server timeouts, real streamed body limits, explicit transaction policies, SQLSTATE mapping, retry/idempotency rules, fault tests |
| Documentation | OpenAPI and parity fixtures are strong | Per-operation examples, invariants, runbooks, generated drift gates, query/transaction rationale |
| Observability | Go uses `slog` and chi request IDs | Stable telemetry vocabulary, context propagation, low-cardinality metrics, trace boundaries, redaction policy, readiness semantics |

## Deferred behavior and schema decisions already recorded

The parent registry currently records:

- change transaction calendar semantics from ambiguous timestamp storage;
- converge finance IDs toward UUIDv7;
- replace in-process rate limiting with a shared mechanism;
- optionally expose an ignored count for partial bulk deletes without leaking
  resource existence; and
- normalize duplicate-category update failures to `409`.

Some OpenAPI work has already intentionally corrected the legacy duplicate
category mismatch. Before opening an implementation ticket, reconcile each
parent entry with the final migrated contract and mark it as still applicable,
superseded, or already delivered.

## Migration boundary

During parity migration:

- internal hardening is allowed if wire behavior remains identical;
- fixtures and the OpenAPI contract remain the acceptance oracle;
- behavior-visible changes stay deferred; and
- no future design in this directory overrides an active migration ticket.

After legacy teardown:

- OpenAPI-first changes may intentionally improve behavior;
- compatibility and rollout become explicit work items; and
- the gates in this directory become mandatory for all backend modules.

## Evidence that must be refreshed before implementation

The Go backend is changing. At project kickoff, regenerate this baseline by
checking the router, middleware, sqlc queries, migrations, OpenAPI contract,
fixtures, and graphify query results. Do not treat the implementation-status
statements in this snapshot as timeless.
