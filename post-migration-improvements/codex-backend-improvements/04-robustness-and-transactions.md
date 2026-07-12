# Robustness and Transaction Semantics

Robustness means predictable behavior under invalid input, concurrency,
partial infrastructure failure, cancellation, retries, and shutdown—not only
the happy path.

## Transaction policy

Every service method documents:

- whether it is read-only or mutating;
- why a transaction is required;
- isolation level;
- statements that must be atomic;
- retryable failure classes and maximum attempts;
- idempotency behavior; and
- externally visible outcome if commit status becomes uncertain.

All RLS-protected reads still require a transaction because user identity is
transaction-local. The scoped unit of work begins, binds `app.user_id`, runs
all queries on the same transaction, and commits/rolls back in one helper.
Rollback errors are logged with the original cause preserved.

## Domain-specific atomicity

- Account/category create and rename map unique violations consistently to
  `409`; they do not rely on a race-prone pre-check.
- Transaction create/update validates the owned account and optional owned
  category in the same transaction as the write. RLS remains the backstop.
- Bulk create is all-or-error: validate the full bounded batch, then insert it
  atomically. No partial inserts escape on a bad reference or row.
- Bulk delete atomically deletes only owned rows and returns only affected IDs;
  any future ignored count is computed without revealing ownership.
- Auth token rotation/consumption and relevant session revocation are atomic.
- Account cascade deletes and category `SET NULL` behavior have explicit
  high-volume and concurrency tests.

## Concurrency and retry

Use PostgreSQL constraints as the arbiter for uniqueness. Classify SQLSTATEs,
including unique, foreign-key, check, serialization, deadlock, cancellation,
and connection failures. Retry only operations proven safe and only for a
small jittered budget. Never blindly retry validation, authorization, unique
conflict, or a mutation whose commit outcome is unknown.

Introduce idempotency keys for externally retried high-value commands—most
notably bulk import—only through an OpenAPI and persistence design. Define
scope, request fingerprint, retention, concurrent duplicate behavior, and
response replay. Do not implement an in-memory idempotency map.

## Time, dates, money, and IDs

- Make transaction calendar semantics explicit. The existing timestamp
  ambiguity is tracked in parent entry `0001`; prefer a PostgreSQL `date` if
  the product remains day-granular.
- Use UTC for instants, expiry, logs, and tokens. Inject clocks into tests for
  boundary behavior.
- Keep money as signed integer milliunits and currency as an explicit dimension.
  Validate ranges and prevent aggregation overflow.
- Treat resource IDs as opaque at the API boundary. Any UUIDv7 convergence
  follows a dual-read/write or data migration plan with rollback evidence.

## Failure behavior

- Classify errors once and return the structured OpenAPI error envelope.
- Preserve request cancellation and deadlines through services, repositories,
  password hashing orchestration, and mail dispatch boundaries.
- Use an outbox or equivalent durable handoff if a committed database change
  must trigger email or another external side effect reliably.
- Health/liveness does not claim database readiness. Readiness checks are
  bounded and fail when the instance cannot safely serve dependent routes.
- Shutdown stops admission, drains bounded in-flight requests, closes workers,
  and then closes the pool.
- Panics are recovered at the boundary, correlated, and returned as a safe 500.

## Testing pyramid

- Unit/property tests: parsers, validation, amount/date boundaries, error
  classification, percentage semantics, and cursor encoding.
- Repository integration tests: real PostgreSQL, migrations, runtime role,
  RLS, constraints, query results, cancellation, and transaction rollback.
- HTTP contract tests: generated bindings, auth policy, content types, limits,
  errors, headers, and examples.
- Concurrency/fault tests: duplicate creates, conflicting updates, token reuse,
  deadlocks/serialization, database loss, slow clients, and shutdown.
- Parity tests remain until intentional contract changes replace them with a
  versioned expectation.

Flaky retry-based tests are not robustness evidence. Tests must control clocks,
seed data, and concurrency barriers wherever practical.
