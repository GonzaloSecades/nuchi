# Authentication API — Technical Reference

Implementation reference for Nuchi's owned authentication and session flows.
The contract (`openapi/nuchi.openapi.json`) is authoritative for operation
shapes and status codes; this document covers the security and transactional
behavior the contract cannot express.

The user-facing companion is [Authentication — Product
Behavior](../product/auth.md).

## Security boundary

All `/api/auth/*` routes are mounted outside `RequireAuth`: they are how a
client obtains or renews credentials in the first place. They are not
unauthenticated in the sense of trusting the caller. Each protected action
proves authority with one of these credentials:

- Login proves knowledge of the password.
- Refresh and logout prove possession of the refresh cookie.
- Email verification and password reset prove possession of a random,
  single-use token.
- App resource routes remain inside `RequireAuth` and derive the user id from a
  verified Bearer access token.

Never add a body or query-string `userId` to an auth-sensitive flow. The
identity behind a refresh, verification, reset, or resource request comes from
the credential the server verified.

## The two-token session

The access token and refresh token deliberately have different storage and
lifetimes.

### Access token: short-lived and memory-only

The API issues a minimal HS256 JWT containing `sub`, `iat`, and `exp`.
`VerifyAccessToken` accepts exactly HS256, requires expiry, and validates that
`sub` is a UUID. Expiry is the only rejection reason exposed separately: the
client needs to know it may refresh. Bad signatures, algorithms, claims, and
subjects collapse into one generic invalid-token result.

In the browser the access token lives only in the closure in
`lib/api/token-store.ts`. It must not move to `localStorage`, session storage,
or a JavaScript-readable cookie. Memory-only storage means a page reload loses
the access token, so `SessionProvider` starts in `loading` and bootstraps from
the refresh cookie before deciding whether the user is authenticated.

Both sides of navigation wait for that decision. `SessionGuard` keeps a
protected route on a loading screen until bootstrap resolves, while the
sign-in page withholds its form during the same state. If navigation has
already reached `/sign-in` and bootstrap succeeds, the sign-in page restores
the validated `redirect` target (or `/`), so a successful refresh cannot leave
an authenticated user stranded on sign-in.

The `client-only` marker on the store is load-bearing. Module state in a Next
server bundle could be shared across requests and users.

### Refresh token: rotating, single-use cookie

Refresh tokens are random 256-bit values. Only a SHA-256 hash is persisted;
the raw token is carried in the `nuchi_refresh_token` cookie.

The cookie is always `HttpOnly`, uses `SameSite=Lax`, follows the configured
`Secure` setting, and is scoped to `Path=/api/auth`. That path keeps it off app
resource requests. JavaScript never reads its value; refresh and logout send it
through the browser cookie jar with credentials included.

Every successful refresh rotates the credential:

1. `ConsumeRefreshToken` performs one guarded `UPDATE`, setting `revoked_at`
   only when the presented hash is unexpired and not already revoked.
2. The winner creates a successor refresh token and issues a new access token.
3. Consuming the old token and creating its successor happen in one database
   transaction.

Exactly one concurrent request can win. If successor creation or another later
step fails, rollback makes the old token usable again; the server must never
burn the only session credential without producing its replacement. Missing,
unknown, expired, or consumed cookies all take the same invalid-refresh path,
and refresh clears the stale cookie.

The client has matching concurrency obligations. `bootstrapSession` shares one
in-flight refresh across React Strict Mode's repeated mount effect, and
`createAuthenticatedFetch` shares one refresh across concurrent resource 401s.
Without either guard, the first request would rotate the cookie and a second
request presenting the consumed value could end an otherwise valid session.
Each resource request retries at most once after refresh.

## Enumeration safety

Enumeration safety is an endpoint-specific invariant, not a blanket claim
about every auth response.

| Surface                          | Required behavior                                                                                | Enforcement                                                                                                                              |
| -------------------------------- | ------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Login                            | Unknown email and wrong password return the same error body and perform comparable password work | `LoginUser` calls `auth.DummyVerify` for an unknown email; `TestAuthLive_Login_WrongPasswordAndUnknownEmailShareShape` pins the response |
| Password-reset request           | Known, unknown, and rate-capped accounts receive the same confirmation                           | `RequestPasswordReset` always returns `resetRequestedMessage`; email sends are asynchronous and the three-per-hour cap is silent         |
| Verification/reset token consume | Unknown, expired, and already-used tokens are indistinguishable                                  | The guarded consume queries return no row for every unusable token and handlers map them to the same invalid-token response              |
| Access-token verification        | All invalid reasons are collapsed except ordinary expiry                                         | `VerifyAccessToken` exposes expiry only so the client can refresh                                                                        |

Registration is intentionally outside that guarantee: a duplicate email
returns the contract's conflict response. Do not describe registration as
enumeration-safe without first changing the contract and behavior.

The reset-request body is uniform and SMTP work is off the response path, but
the implementation does not claim constant time. A known account runs a small
serialized database transaction while an unknown account does not. That
residual timing oracle is recorded as divergence 0012.

## Transactional invariants

These flows spend a one-time credential or create coupled state. Their
transaction boundaries are correctness boundaries.

### Registration

`CreateUser` and `CreateEmailVerificationToken` share one transaction. A user
must never commit without the token needed to verify the account. The email is
sent asynchronously only after commit, so SMTP work cannot hold the transaction
open or change the HTTP outcome.

### Refresh

Consuming the presented refresh token and creating its successor share one
transaction. The guarded consume is a single statement, not a read followed by
a revoke. This gives concurrent refreshes one winner and makes a later failure
roll the consume back.

### Logout

Logout also uses `ConsumeRefreshToken` as an atomic check-and-revoke. A separate
validity read followed by a revoke has a race in which concurrent refresh or
logout could turn an invalid session into a false success.

### Email verification

`ConsumeEmailVerificationToken` and `MarkUserEmailVerified` share one
transaction. If marking the user fails, rollback leaves the token usable for a
retry. Two concurrent submissions of the same token have exactly one winner.

### Password-reset issuance

Issuance runs as:

`LockUser FOR UPDATE → count recent tokens → invalidate prior tokens → create token → commit`

The row lock serializes requests for one user. It prevents concurrent requests
from both passing the three-per-hour cap and prevents more than one new token
from remaining live. Reaching the cap commits no mutation, returns the same
public confirmation, and sends no email.

### Password-reset confirmation

Confirmation validates and hashes the new password before database work, then
runs:

`consume reset token → update password → invalidate reset tokens → revoke all refresh tokens → commit`

The token remains usable if any mutation fails. A successful reset revokes
every refresh session, not only the browser that submitted it. Already-issued
access JWTs remain valid until their short expiry; see divergence 0008.

## Email delivery boundary

Verification and reset mail is best-effort and asynchronous with a bounded,
request-independent context. Callers invoke `sendAsync` only after the token
transaction commits. Failures are logged with the kind and user id, never the
raw token or email body, and do not alter the already-sent response.

There is no durable outbox or verification-resend endpoint. That is accepted
debt, not permission to send before commit or hold a transaction open during
SMTP.

## Non-negotiables when changing this code

- Keep raw passwords transient and store only Argon2id PHC hashes. Store only
  hashes of verification, reset, and refresh tokens; never log raw tokens,
  cookies, passwords, or email-link bodies.
- Keep one-time token consumption as a guarded single statement. Never replace
  it with `SELECT` then `UPDATE`.
- Keep each consume and the mutations it authorizes in the same transaction.
  Rollback-fault tests must continue to prove that a failed mutation does not
  burn the token.
- Keep refresh consume and successor creation atomic, and preserve both client
  deduplication guards. Server atomicity alone does not prevent the browser
  from clearing a newly rotated cookie after its own duplicate request.
- Keep the access token memory-only and the token-store module client-only.
- Keep the cookie name, `HttpOnly`, `Path=/api/auth`, clearing attributes, and
  contract examples synchronized. Cookie hardening is a contract and browser
  compatibility change, not a handler-local edit.
- Keep login's dummy password verification and byte-identical credential error
  shape. Keep reset-request responses uniform across unknown, capped, and
  successfully issued cases.
- Keep reset issuance's cap check inside the per-user locked transaction.
- Send mail only after commit. Preserve token redaction in logs.
- Keep verified identity as the only source of user ownership. Never accept a
  caller-selected user id.
- Change `openapi/nuchi.openapi.json` first for contract changes, regenerate
  both clients, and never hand-edit generated code.

## Where the code lives

| Concern                                                                  | File                                          |
| ------------------------------------------------------------------------ | --------------------------------------------- |
| Handlers, transaction orchestration, cookies, enumeration-safe responses | `backend/internal/http/auth.go`               |
| Password hashing and dummy verification                                  | `backend/internal/auth/password.go`           |
| JWT issue and verification policy                                        | `backend/internal/auth/jwt.go`                |
| Atomic token queries                                                     | `backend/internal/db/queries/auth_tokens.sql` |
| Auth tables                                                              | `backend/migrations/00001_auth_base.sql`      |
| Route boundary and Bearer middleware grouping                            | `backend/internal/http/router.go`             |
| Browser access-token store                                               | `lib/api/token-store.ts`                      |
| Resource refresh/retry coordination                                      | `lib/api/authenticated-fetch.ts`              |
| Page-load, verification, and logout coordination                         | `lib/auth/session-requests.ts`                |
| React session state                                                      | `lib/auth/session.tsx`                        |

## Divergences and gaps

Recorded in `post-migration-improvements/claude-backend-improvements/`:

| Entry | Note                                                                                                          |
| ----- | ------------------------------------------------------------------------------------------------------------- |
| 0007  | Access JWTs use HS256 with one static secret and no `kid` rotation window                                     |
| 0008  | Logout/reset revoke refresh sessions, but cannot revoke an issued access JWT before expiry                    |
| 0009  | Replayed rotated tokens are rejected without token-family reuse detection; there is no session-listing UI/API |
| 0010  | Auth handlers can return a generic 500 that the OpenAPI operations do not declare                             |
| 0011  | Verification mail has no resend endpoint or durable outbox; delivery is fire-and-forget                       |
| 0012  | Reset-request responses are uniform, but known accounts still do more database work than unknown accounts     |
