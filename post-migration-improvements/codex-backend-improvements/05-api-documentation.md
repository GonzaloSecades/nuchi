# API Contract and Documentation Standard

OpenAPI remains the source of truth for observable HTTP behavior. Generated Go
and TypeScript artifacts are outputs, never hand-edited sources.

## Required operation documentation

Every operation documents:

- purpose and authorization mode;
- ownership behavior and non-disclosure rule;
- parameters, defaults, inclusive/exclusive boundaries, normalization, and
  deterministic ordering;
- request/response examples, including nullable fields and empty results;
- all expected error codes/statuses and relevant response headers;
- body, batch, date-range, page-size, and rate limits;
- atomicity, partial-success, retry, and idempotency semantics;
- money units, currency rules, date/time semantics, and ID opacity; and
- deprecation/replacement information when behavior changes.

Descriptions must explain behavior a client needs, not Go or SQL internals.
Internal rationale belongs in query files, ADRs, or this project set.

## Contract workflow

1. Propose behavior with compatibility and rollout notes.
2. Edit `openapi/nuchi.openapi.json` first.
3. Validate and lint the contract.
4. Regenerate strict Go server bindings and TypeScript types.
5. Implement handlers/services/repositories.
6. Add example-backed contract tests and negative cases.
7. Verify generation produces no unexplained diff.
8. Update module/runbook documentation and changelog.

CI fails on stale generated files, duplicate/missing operation IDs, missing
security declarations, undocumented non-default errors, or incompatible
changes without explicit approval/versioning.

## Internal documentation

Each sqlc query has a stable name and comments for non-obvious ownership,
locking, ordering, or performance decisions. Complex queries link to their
reviewed plan artifact and dataset assumptions. Each service method records
its transaction and idempotency policy.

Architecture decisions with lasting tradeoffs use a short ADR: context,
decision, alternatives, consequences, migration, rollback, and verification.
Do not duplicate an OpenAPI field catalogue in Markdown.

## Operational documentation

Before production readiness, maintain runbooks for:

- database migration, rollback/forward-fix, and compatibility windows;
- key rotation and compromised-session response;
- database saturation, slow queries, elevated 5xx, and rate-limit spikes;
- backup/restore verification;
- deployment, draining, and rollback; and
- telemetry configuration once a vendor/tool is selected.

Runbooks name owners, safe diagnostic queries, expected signals, stop
conditions, and escalation paths. Commands that destroy or expose data are
clearly marked and never part of routine automation.

## Documentation acceptance gate

- The OpenAPI validator and both generators pass.
- Generated artifacts are clean after regeneration.
- Every changed operation has success and principal failure examples.
- Endpoint limits, ordering, atomicity, and auth are unambiguous.
- Internal transaction/query rationale is linked from the implementation.
- Any behavior-visible change has compatibility, rollout, and rollback notes.
