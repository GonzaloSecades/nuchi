# Observability Readiness

The backend should be instrumentable without rewriting business logic, while
remaining vendor neutral until an observability tool is selected. Readiness
means stable context, naming, boundaries, and redaction—not adding a specific
SDK prematurely.

## Context and correlation

- Carry `context.Context` from HTTP through service and repository calls.
- Accept a valid inbound trace context only according to a documented trust
  policy; generate a request ID when absent.
- Return the public request/correlation ID in a response header and structured
  API errors where the contract chooses to expose it.
- Use stable route templates/operation IDs, never raw paths containing IDs, as
  telemetry dimensions.
- Represent users in telemetry only when needed and then with a non-reversible
  or access-controlled pseudonymous identifier. Never attach email or tokens.

## Structured logging

Use `slog` through an injected logger enriched at boundaries. Standard fields:

- service, environment, version, instance;
- request/trace ID and OpenAPI operation ID;
- method, route template, status, duration;
- classified error code and retry count; and
- database operation name, never raw SQL values.

Log once at the boundary for a failed request. Lower layers wrap and classify
errors rather than repeatedly logging them. Validation and expected conflicts
are not error-level server failures. Define sampling for high-volume successes.

Redact authorization, cookies, passwords, password hashes, reset/verification
tokens, database URLs, raw bodies, financial notes, and other sensitive data.
Automated tests assert known secret markers do not reach logs.

## Metrics vocabulary

Prepare counters/histograms/gauges with bounded labels:

- HTTP requests and duration by operation ID, method, status class;
- classified application errors;
- database query duration/errors by stable query name;
- pool acquired/idle/total connections, acquire duration, and timeouts;
- transaction commit/rollback/retry counts;
- auth outcomes and token-reuse detection without email/user labels;
- rate-limit decisions by policy class; and
- bulk batch sizes and summary range sizes as histograms.

Never label metrics with resource ID, request ID, user ID, email, payee,
category name, raw error text, or URL. Cardinality rules must be testable in
the instrumentation adapter.

## Trace boundaries

Future spans should align to stable operations:

- inbound OpenAPI operation;
- application service/use case;
- transaction/unit of work;
- sqlc query name; and
- external mail or other dependency calls.

Do not create a span for every helper or batch item. Attach safe counts and
classified outcomes, not payloads. Context cancellation and deadlines must be
visible as distinct outcomes from internal errors.

## Health, readiness, and diagnostics

- Liveness answers whether the process/event loop is alive and must remain
  cheap.
- Readiness reports whether the instance can safely accept traffic, including
  bounded database connectivity and required initialization.
- Build/version information is available without secrets.
- Database query plans and profiles are collected through controlled operator
  workflows, never exposed as unauthenticated debug routes.

## Instrumentation seam

Define a small internal telemetry interface or use established Go context
conventions at the platform boundary. Domain rules should not import a future
vendor SDK. A no-op implementation supports tests/local execution; the future
adapter may use OpenTelemetry or another selected tool.

The seam must support request completion, service spans, named database
operations, counters/histograms, and classified errors. Benchmark the adapter
and ensure it is concurrency-safe and non-blocking under exporter failure.

## Readiness gate before selecting a tool

- Stable operation/query/error names are documented.
- Context reaches every database and external dependency call.
- Logger and telemetry dependencies are injected.
- Sensitive-field and metric-cardinality policies have tests.
- Liveness and readiness semantics are distinct.
- Endpoint SLO hypotheses and runbook signal requirements exist.
- The service functions correctly with the no-op telemetry implementation.
