# Target Backend Architecture

## Core rule

Handlers coordinate HTTP concerns; services own use-case rules; repositories
own SQL; PostgreSQL enforces relational and ownership invariants. Generated
OpenAPI code defines types and bindings but contains no business logic.

## Request flow

```text
client
  -> proxy / TLS boundary
  -> request ID + panic recovery + security headers
  -> route-specific body/time limits
  -> authentication or public-endpoint policy
  -> rate limit / abuse policy
  -> OpenAPI request validation
  -> handler adapter
  -> application service
  -> user-scoped database transaction
       -> set_config('app.user_id', userID, true)
       -> sqlc repository queries with explicit ownership predicates
       -> commit or rollback
  -> typed response / centralized error mapping
  -> structured log, metric, and trace hooks
```

Middleware order is a security and correctness decision and must have an
integration test. Public endpoints use the same pipeline with an explicit
public policy; they must not become public merely because auth middleware was
forgotten.

## Package boundaries

The exact package names may evolve, but dependencies should point inward:

```text
cmd/api
  -> platform/http, platform/config, platform/db, platform/telemetry
  -> transport/openapi adapters
  -> application services
  -> domain rules
  -> repositories (interfaces owned by the application/domain consumer)
  -> sqlc-generated database implementation
```

- `cmd/api` is the composition root only.
- HTTP adapters translate OpenAPI types to domain commands and back.
- Services accept `context.Context`, typed principals, and transaction-capable
  repository interfaces; they do not depend on `http.Request`.
- Repositories return domain values and classified errors, not HTTP statuses.
- Generated OpenAPI and sqlc packages are replaceable implementation details.
- Cross-cutting dependencies are injected; package globals are limited to
  immutable constants.

## User-scoped database unit of work

Every request that touches an RLS-protected table executes in a database
transaction, including reads. The first database statement binds the verified
user ID using transaction-local `set_config`. Business queries execute on the
same acquired connection and transaction. Commit/rollback is centralized and
tested to prevent identity leakage through pooled connections.

The unit of work API must make an unbound user query difficult to express. A
service should receive a scoped query interface only after binding succeeds.
Direct pool access from handlers is prohibited.

## Endpoint policy registry

Every OpenAPI operation ID must map to an explicit policy containing:

- authentication mode: public, Bearer, or refresh cookie;
- authorization/resource ownership rule;
- maximum decoded and wire body size;
- request timeout and database statement timeout;
- rate-limit class;
- idempotency behavior;
- transaction isolation/retry policy; and
- telemetry name and sensitive-field classification.

Central defaults may exist, but absence of a policy must fail tests or startup.
This prevents new generated operations from silently inheriting unsafe
behavior.

## Error boundary

Use one central mapper from classified domain/infrastructure errors to the
OpenAPI `ApiErrorResponse`. External responses expose stable codes and safe
messages. Internal logs retain the wrapped cause, operation, and request ID.
Expected validation, conflict, not-found, and rate-limit outcomes must never
be counted as unclassified server faults.

## Configuration

Configuration is typed, validated once at startup, and divided into safe
runtime values and secrets. Production must fail fast for missing keys,
insecure cookie settings, unusable pool bounds, or invalid timeouts. Startup
logs report non-secret effective configuration; database credentials, tokens,
passwords, reset links, and raw authorization headers are never logged.

## Architecture acceptance tests

- Every OpenAPI operation is registered exactly once and has an endpoint
  policy.
- Protected operation tests fail without a principal.
- RLS integration tests prove cross-user read and write isolation with the
  production application role.
- A pooled connection reused after commit cannot observe the previous user ID.
- HTTP packages cannot import generated sqlc code directly.
- Contract generation and repository generation are reproducible and clean.
