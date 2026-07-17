# 0010 — Auth operations do not declare 500 responses in the contract

- **Migration ticket:** #41 (https://github.com/GonzaloSecades/nuchi/issues/41), flagged in PR #62 review
- **Area:** api
- **Priority guess:** low

## How it was migrated

The auth handlers return a 500 with the shared `ApiErrorResponse` shape
(`writeInternalError`) on unexpected faults (DB outage, crypto failure),
but the OpenAPI contract's auth operations declare only their 2xx/4xx
responses — 500 is undocumented.

## Why it was done this way

OpenAPI-first: adding responses to the contract is a contract change that
regenerates both sides, which is heavier than a PR review round should
carry, and unexpected-fault responses are conventionally omitted from
operation definitions. The emitted body already reuses the documented
`ApiErrorResponse` schema, so clients that handle documented errors
generically handle 500s too.

## The concern

A contract that claims to be the oracle is silent about a response the
server demonstrably produces. Generated clients get no typed 500 handling,
and contract-based tests can't assert the fault shape.

## Proposed improvement

Add a shared `default` (or explicit `500`) response referencing
`ApiErrorResponse` to every operation in one contract pass, then
regenerate. Do it once for the whole API — auth and resource operations
together — rather than per-ticket, so it lands as a single OpenAPI-first
change after parity.
