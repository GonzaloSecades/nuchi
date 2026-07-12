# Delivery Roadmap and Quality Gates

## Phase 0 — Rebaseline after migration

Goal: replace this snapshot with evidence from the final Go backend.

- Query graphify for each module and inspect the final router, middleware,
  services, sqlc queries, migrations, OpenAPI, and tests.
- Reconcile all parent `NNNN` entries with shipped contract behavior.
- Build an operation inventory from OpenAPI operation IDs.
- Record current latency/query counts/plans against a versioned realistic
  dataset.
- Create a risk register and owner map.

Exit: every operation has an owner, current-state review, baseline measurement,
and no unknown auth policy.

## Phase 1 — Security and correctness foundation

- Introduce the endpoint policy registry and centralized typed principal.
- Implement the RLS-scoped transaction/unit-of-work abstraction.
- Prove runtime-role RLS isolation and pooled-connection cleanup.
- Centralize safe error classification/mapping.
- Enforce streamed body limits, server timeouts, context deadlines, and secure
  proxy/origin/cookie settings.
- Complete JWT/session/token and rate-limit threat-driven tests.

Exit: all P0 security tests pass; no handler directly uses the pool or accepts
ownership from client input; protected routes fail closed.

## Phase 2 — Transaction and resilience hardening

- Document and implement transaction/isolation/retry policy per service.
- Make bulk and reference workflows atomic.
- Add cancellation, database failure, concurrency, and graceful-drain tests.
- Add durable external-side-effect strategy where required.
- Decide idempotency scope for retried import/auth commands.

Exit: max-size and concurrency fault suites pass without partial state or
unbounded retry.

## Phase 3 — Query and runtime performance

- Set approved operation-class SLOs and budgets.
- Add deterministic pagination and ordering through OpenAPI-first changes.
- Review all query plans under RLS/runtime role with realistic distribution.
- Tune indexes and pool/runtime limits using measured evidence.
- Optimize summary only after semantic characterization tests lock behavior.

Exit: every operation meets its objective at agreed load, with stored plans and
no functional/security regression.

## Phase 4 — Documentation and observability readiness

- Complete operation examples, limits, retry/atomicity semantics, and changelog.
- Enforce generated artifact and compatibility checks in CI.
- Add stable logging fields, telemetry interfaces/no-op adapter, context
  propagation, redaction/cardinality tests, and distinct readiness.
- Write security, database, performance, deploy, and incident runbooks.

Exit: all endpoint review templates are complete and the service can adopt an
observability adapter without modifying domain/application logic.

## Phase 5 — Behavior and schema evolution

Evaluate the parent registry using measured priority:

- transaction `date` versus timestamp;
- UUIDv7 convergence;
- distributed rate limiting;
- explicit aggregate ignored counts for bulk delete; and
- any remaining duplicate-conflict mismatch.

Each becomes its own OpenAPI/schema proposal with data migration, compatibility,
rollout, and rollback. Do not bundle unrelated schema changes.

## Pull request gates

All backend changes:

- focused tests, race/static/vulnerability checks as applicable;
- OpenAPI/sqlc generated artifacts clean;
- no secrets, debug endpoints, raw SQL logging, or dead code;
- documentation updated for changed behavior or internal policy; and
- graphify updated after code/document changes.

Route/query/schema changes additionally require:

- endpoint review or a linked existing review;
- ownership/RLS and negative tests;
- transaction policy and SQLSTATE mapping;
- query count/plan evidence when data access changes;
- compatibility assessment; and
- rollback or forward-fix plan for migrations.

## Release gates

- Contract tests pass against the built service.
- Migrations pass from supported versions and a restore drill is current.
- Load test meets SLOs without pool exhaustion or correctness drift.
- Security regression and cross-user isolation suites pass.
- Readiness, shutdown, and dependency-failure behavior are verified.
- Dashboards/alerts are not required before a tool is selected, but their
  required signals and runbooks must be defined.

## Evidence storage convention

For each improvement ticket, keep links or small checked-in artifacts for:

- proposal/ADR;
- endpoint review;
- benchmark command, dataset version, and results;
- query plans;
- contract compatibility report;
- migration validation; and
- rollout outcome.

Avoid committing real user financial data, secrets, or massive transient
profiles. Record reproducible commands and sanitized summaries instead.
