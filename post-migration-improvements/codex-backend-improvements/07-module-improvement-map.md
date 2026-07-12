# Module Improvement Map

This map identifies the first review questions for each module. It is not a
substitute for inspecting the final migrated implementation at kickoff.

## Platform and HTTP

Current foundation: chi request ID, real IP, recovery, a health handler,
pgxpool startup ping, graceful shutdown, and strict generated server bindings.

First improvements:

- compose generated strict handlers into the runtime router;
- add explicit trusted-proxy handling before relying on forwarded client IP;
- add security headers, streamed body limits, full server timeouts, admission
  control, centralized errors, and an endpoint policy registry;
- distinguish liveness/readiness and make graceful drain observable; and
- configure pool bounds and statement/request deadlines from validated config.

## Authentication and sessions

Quality focus:

- strong password hashing with measured parameters and bounded concurrency;
- enumeration-resistant commands and shared distributed abuse controls;
- JWT algorithm/claim/key-rotation policy;
- hashed, atomic, rotating refresh tokens with reuse detection;
- atomic verification/reset consumption and durable email delivery handoff;
- origin/CSRF policy for refresh-cookie endpoints; and
- session/key operational runbooks.

Critical tests: timing-safe invalid login classes, token replay/races, revoked
sessions, wrong token type/audience, clock boundaries, mail failure after
commit, and redaction.

## Accounts

Quality focus:

- deterministic bounded listing;
- consistent case-insensitive unique conflict mapping on create and update;
- conditional single-statement update/delete with ownership predicates;
- cascade-delete cost and safety for accounts with many transactions;
- bulk-delete maximum size/body limit and partial-success contract; and
- RLS isolation using the runtime role.

Review whether account deletion should remain synchronous at the maximum
supported transaction volume. Any change requires an explicit product/API
decision.

## Categories

Quality focus:

- match account conflict/error consistency;
- reconcile the legacy duplicate-update `500`, current OpenAPI `409`, and
  parent entry `0005` against the final migrated behavior;
- verify category-to-transaction ownership on transaction mutations;
- measure `ON DELETE SET NULL` for heavily used categories; and
- preserve non-disclosing bulk behavior.

## Transactions

This is the highest data-volume and mutation-risk module.

Quality focus:

- introduce deterministic keyset pagination after a contract design;
- enforce real streaming body limits in addition to `Content-Length` hints;
- keep batches bounded and set-based with constant query count;
- validate owned account/category and write within one transaction;
- benchmark `account_id, date DESC` and possible `(date, id)` cursor support;
- replace process-local rate limiting when multi-instance/security needs demand;
- design durable idempotency for retried CSV bulk imports;
- decide calendar-date storage and UUIDv7 migration proposals; and
- prove milliunit/currency bounds and overflow safety.

Critical tests: 1/500/501 rows, bad final-row rollback, duplicate requests,
chunked oversize bodies, cross-owner mixed references, concurrent reference
deletion, cancellation during insert, and deterministic equal-date ordering.

## Summary

Quality focus:

- preserve current and previous-period, top-three-plus-Other, daily-fill, date
  range, account filter, and percentage-change semantics;
- benchmark conditional aggregation versus the legacy four-query shape;
- test all queries under RLS with the runtime role and realistic skew;
- establish an explicit latency objective for the maximum 366-day range;
- keep results currency-safe; and
- defer caching/pre-aggregation until indexed direct queries demonstrably miss
  the objective.

Critical tests: empty data, all income/all expense, uncategorized expenses,
equal category totals, daylight-saving/local-date boundaries, max range,
previous-period boundary, very large milliunit totals, and unowned account
filter non-disclosure.

## Database and migrations

Quality focus:

- separate migrator/runtime roles and verify runtime cannot bypass RLS;
- use expand/migrate/contract for incompatible changes;
- test migrations from empty and representative prior versions;
- record locks, expected duration, rollback/forward-fix, and data validation;
- run sqlc generation and contract generation as drift gates; and
- maintain realistic benchmark fixtures without production personal data.

## Cross-module priority table

| Priority | Work | Reason |
| --- | --- | --- |
| P0 | RLS-scoped unit of work and cross-user tests | Data isolation foundation |
| P0 | Auth/session cryptographic and cookie/origin policy | Account compromise boundary |
| P0 | Streamed limits, deadlines, centralized safe errors | Resource exhaustion and failure containment |
| P0 | Atomic bulk/reference/token workflows | Prevent partial or cross-owner state |
| P1 | Query benchmarks, pagination, pool tuning, summary plans | Scale and latency |
| P1 | Contract completeness and generation/drift gates | Client/server correctness |
| P1 | Telemetry vocabulary, context, redaction seam | Operational readiness |
| P2 | Date/ID schema convergence and richer bulk responses | Valuable but migration-heavy behavior/schema changes |

P0/P1/P2 indicate sequencing, not permission to bypass active migration parity.
