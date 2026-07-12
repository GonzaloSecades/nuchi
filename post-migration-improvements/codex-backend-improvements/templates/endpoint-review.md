# Endpoint Review — `<operationId>`

## Identity

- **Method/path:**
- **Module/owner:**
- **OpenAPI operation ID:**
- **Implementation links:**
- **Observable behavior change:** yes/no; link compatibility plan if yes

## Contract

- Purpose, input, defaults, ordering, and response envelope:
- Documented success/error examples:
- Limits: body, batch, range, page, rate, timeout:
- Atomicity, partial-success, retry, and idempotency semantics:

## Security

- Auth mode and verified principal source:
- Ownership rule and SQL predicate:
- RLS policy/test evidence:
- Cross-user/non-disclosure tests:
- CSRF/origin/cookie policy if applicable:
- Sensitive fields and redaction test:
- Abuse/body/concurrency controls:

## Performance

- Operation-class objective and tested load/dataset:
- SQL statements at min/typical/max input:
- Relevant indexes and reviewed query plans:
- p50/p95/p99, throughput, response bytes:
- Cancellation/pool saturation result:

## Robustness

- Transaction/isolation policy:
- Constraints and classified SQLSTATEs:
- Retry/uncertain-commit behavior:
- Concurrency, fault, max-boundary, and shutdown tests:
- External side effects and durable handoff:

## Documentation and observability readiness

- OpenAPI/generation/drift checks:
- Stable log/metric/trace operation names:
- Context propagation and request ID:
- Low-cardinality telemetry fields:
- Runbook/alert signal impact:

## Decision

- **Status:** approved / changes required / risk accepted
- **Evidence links:**
- **Accepted risks:** owner, compensating control, expiry, ticket
- **Reviewer/date:**
