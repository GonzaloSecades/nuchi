# Security Standard

Security is a release gate, not a best-effort quality. The application must
assume identifiers, filters, cookies, headers, CSV-derived input, and all JSON
fields are attacker controlled.

## Identity and session requirements

- Allowlist JWT algorithms; validate signature, issuer, audience, subject,
  expiry, not-before, and required token type. Never select an algorithm from
  an untrusted token without policy validation.
- Keep access tokens short lived and support key rotation with explicit key
  IDs. Reject unknown or retired keys safely.
- Store only hashed refresh, verification, and reset tokens. Consume one-time
  tokens atomically and rotate refresh tokens transactionally.
- Detect refresh-token reuse and revoke the affected session family.
- Cookies are `HttpOnly`, `Secure` in production, narrowly pathed where
  practical, and use an intentional `SameSite` policy.
- Cookie-authenticated refresh/logout endpoints validate allowed origin and
  CSRF assumptions. Same-origin proxying is not a substitute for a documented
  CSRF policy.
- Login, registration, verification, and reset responses resist account
  enumeration. Abuse limits apply before expensive password hashing or mail
  work when possible.

## Authorization and ownership

Defense in depth is mandatory:

1. Middleware creates a typed principal only from a verified credential.
2. Services derive ownership from that principal, never request JSON or query
   parameters.
3. sqlc queries include ownership predicates.
4. Forced PostgreSQL RLS provides the final backstop.
5. Tests attempt horizontal access with a second real user.

Transaction ownership includes both the required account and optional
category. Account/category validation and transaction mutation occur inside
one database transaction so references cannot change between check and write.
Database constraints or an equivalent atomic statement must enforce any
cross-table invariant that application validation alone cannot guarantee.

Not-found responses must not reveal whether another user owns an identifier.
Bulk-delete behavior may report aggregate ignored counts only if the contract
preserves that non-disclosure property.

## Input and resource controls

- OpenAPI validation is necessary but not sufficient; domain validation owns
  semantic rules such as date bounds, currencies, milliunit ranges, and
  reference compatibility.
- Enforce maximum request bytes while reading the stream. `Content-Length` is
  an early rejection hint, not the enforcement mechanism; chunked and missing
  lengths must still be bounded.
- Bound array lengths, string lengths, date ranges, page size, decompressed
  size, and query complexity.
- Reject unknown JSON fields for mutation commands unless compatibility
  explicitly requires them.
- Normalize identifiers and names once. Use parameterized sqlc queries only.
- Apply server-level header limits, read/write/idle timeouts, and concurrency
  protection against slow clients.

## Database and secret controls

- Runtime roles receive only required schema privileges and cannot bypass RLS.
- Migration credentials are separate from runtime credentials.
- Production database connections require verified TLS.
- Use transaction-local `app.user_id`; never use session-scoped state on a
  pooled connection.
- Secrets come from the deployment secret mechanism, never source, generated
  docs, telemetry, or error bodies.
- Dependency, Go vulnerability, secret, and static analysis checks run in CI
  with a documented triage policy.

## Security tests required per resource module

- missing, malformed, expired, wrong-audience, and wrong-token-type credentials;
- owned, unowned, nonexistent, and malformed IDs;
- cross-user reads, updates, deletes, bulk operations, and foreign references;
- unset/empty/malformed RLS identity;
- oversized known-length, chunked, and decompressed bodies;
- rate-limit boundary and bypass attempts;
- safe SQLSTATE/error mapping without constraint or credential leakage; and
- logs checked for token, cookie, password, email-link, and database-secret
  leakage.

## Security review evidence

Each endpoint review links its threat cases, authorization tests, RLS test,
body-limit test, rate-limit policy, sensitive-data classification, and any
accepted risk. An accepted security risk requires a named owner, compensating
control, expiry date, and ticket; “follow up later” is not sufficient.
