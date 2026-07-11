# 0003 — Transaction rate limiting is in-memory

- **Migration ticket:** #40 context; implementation lands with handlers (#44+) (https://github.com/GonzaloSecades/nuchi/issues/40)
- **Area:** api/infra
- **Priority guess:** medium

## How it was migrated

The legacy behavior (fixtures "Transaction Mutation Rate Limit"): 60
requests per 60-second window, keyed `userId:action`, returning `429` with a
`Retry-After` header. The Go port will reproduce this with an in-process
store, matching legacy semantics. No database table backs it (confirmed
during #40 briefing: no DB surface).

## Why it was done this way

Parity: the fixtures freeze the window, limits, key shape, and error body.
An in-process implementation is the faithful port and is fine for a
single-instance deployment.

## The concern

In-memory limits silently reset on every deploy/restart and do not aggregate
across instances — if the API ever runs with more than one replica, the real
limit becomes `limit × replicas` and an attacker can rotate. Also, auth
endpoints (login attempts, reset-token issuance) will want rate limiting
with **security** stakes, which deserves a store that survives restarts.

## Proposed improvement

Move rate limiting to a shared store once deployment topology demands it:
either a small Postgres table (we already track `created_at` on token tables
for exactly this — see #41 design decisions) or Redis if latency ever
matters. Unify transaction-mutation limits and auth limits behind one
mechanism with per-key configuration.
