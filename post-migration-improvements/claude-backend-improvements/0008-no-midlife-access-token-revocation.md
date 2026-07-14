# 0008 — Access tokens cannot be revoked mid-life

- **Migration ticket:** #41 (https://github.com/GonzaloSecades/nuchi/issues/41)
- **Area:** auth
- **Priority guess:** low

## How it was migrated

Access tokens are stateless JWTs valid until `exp` (default 30 minutes).
Logout and password reset revoke the *refresh* session immediately, but an
already-issued access token keeps working until it expires — nothing
server-side can kill it sooner.

## Why it was done this way

Standard stateless-JWT architecture, accepted in the #41 design review
(extensibility table row 4): the short TTL bounds exposure, the refresh
cookie — the long-lived credential — is instantly revocable, and a per-
request denylist lookup would reintroduce the DB hit that stateless tokens
exist to avoid.

## The concern

A stolen access token has a worst-case ≤30-minute window that no server
action can close. For a personal finance app the window bounds read/write
access to one user's financial data.

## Proposed improvement

Only if a concrete need appears (compromise response, compliance): add a
`jti` claim and a small denylist checked by the auth middleware (in-process
cache backed by a table; entries expire with the token, so it stays tiny).
Additive change — no token format break, no migration. Revisit alongside
0003 (shared rate-limit store), which would provide the same storage.
