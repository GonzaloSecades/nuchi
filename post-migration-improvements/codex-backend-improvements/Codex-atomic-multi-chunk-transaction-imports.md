# Codex — Atomic Multi-Chunk Transaction Imports

- **Modules/operations:** transaction CSV import, `bulkCreateTransactions`
- **Owner:** transactions/API
- **Priority:** P2
- **Parent registry entry:** none; discovered during migration ticket #47
- **Target milestone:** post-migration Phase 2 — transaction and resilience hardening

## Problem and evidence

`POST /api/transactions/bulk-create` accepts at most 500 transactions and is
atomic for one request. The frontend imports a larger CSV by splitting it into
500-row chunks and awaiting each request sequentially in
`app/(dashboard)/transactions/page.tsx`.

That creates an import-level partial-success state: if chunk 1 commits and chunk
2 fails validation, loses its connection, or receives a database error, the
first 500 rows remain visible while the UI reports that the import failed. A
retry can then duplicate the already-committed prefix because the current bulk
operation has no import identity or idempotency key.

This is documented behavior, not a #47 parity defect. The migration guarantees
atomicity per request; it cannot provide atomicity across independent HTTP
requests without a new API and persistence model.

## Invariants

- Ownership comes only from the authenticated principal and is enforced for
  every referenced account and category through RLS-backed transactions.
- No staged row becomes visible in transaction lists or summaries before the
  entire import commits.
- Amounts remain signed integer milliunits and currency remains explicit.
- Final transaction order is stable and can be correlated to source row order.
- Retries with the same import identity and payload do not create duplicates.
- Existing `bulkCreateTransactions` behavior remains available and unchanged.
- Abandoned imports are bounded by row/byte limits and expire safely.

## Proposed design

Add a persisted import-session workflow instead of holding a database
transaction open across requests:

1. Create an authenticated import session with declared row count, byte count,
   and a client-provided idempotency key.
2. Upload numbered chunks into an RLS-protected staging table. Store a payload
   fingerprint per chunk so an identical retry is replay-safe and a conflicting
   retry is rejected.
3. Finalize once all chunks are present. In one `WithUserTx`, validate the full
   staged set and its owned references, insert all transactions, verify the
   returned ID set/order, and mark the session committed.
4. Allow explicit abort and expire abandoned sessions with a bounded cleanup
   job. Committed sessions retain enough metadata for safe response replay for a
   documented period.

Keep the existing bulk-create endpoint for ordinary bounded batches. The import
session is a separate workflow because staging, finalization, expiry, and replay
semantics should not complicate the simple bulk mutation.

## Alternatives and tradeoffs

- **Status quo:** simplest implementation, but permits partial imports and
  duplicate prefixes after ambiguous failures.
- **One larger JSON request:** preserves a single transaction but requires a
  much larger body cap, buffers more data, and still has weak retry semantics.
- **Streaming CSV/NDJSON endpoint:** can parse incrementally and commit once,
  but couples transport parsing to domain validation and makes resumable uploads
  harder.
- **Compensating deletes after a failed chunk:** is not atomic, can itself fail,
  and can race with user edits; do not use it as the correctness mechanism.

The staged-session design adds schema and cleanup complexity, but it avoids
long-lived database transactions and gives retries an explicit identity.

## Contract and compatibility

- **OpenAPI change and generated artifacts:** add import-session create, chunk
  upload, finalize, status, and abort operations with total row/byte limits,
  idempotency conflict responses, and expiry semantics.
- **Client migration/versioning:** update the CSV import UI to use sessions;
  leave `bulkCreateTransactions` unchanged for existing clients.
- **Rollout and rollback:** deploy staging schema and dormant endpoints first,
  then switch the UI behind a feature flag. Roll back the UI independently;
  retain cleanup support until all staged sessions expire.
- **Deprecation window:** none required unless product later decides to remove
  client-side chunked imports through the existing endpoint.

## Verification

- Cross-user access to sessions and staged chunks is denied by SQL predicates
  and RLS under the runtime role.
- A validation/reference failure in any chunk leaves zero new transactions.
- Finalization inserts every staged row exactly once and preserves source order.
- Re-uploading an identical chunk and retrying finalization replay safely;
  conflicting payloads with the same key are rejected.
- Cancellation, timeout, lost-response, concurrent-finalize, abort/finalize,
  and expiry races have deterministic outcomes.
- Maximum-size imports meet agreed latency, transaction-duration, pool, and
  staging-storage objectives with stored query plans.
- Transaction list and summary queries never expose uncommitted staged rows.

## Risks

- **Staging growth:** transactions/API owns row/byte quotas, expiry metrics, and
  cleanup alerts; disable new sessions if cleanup falls behind.
- **Long finalization transactions:** transactions/API owns batch-size and
  duration limits; detect via transaction-duration telemetry and retain the
  client-chunked fallback during rollout.
- **Ambiguous finalization response:** persisted terminal state and response
  replay are mandatory; do not infer success from the client connection.
- **Contract complexity:** require an endpoint review and compatibility plan
  before acceptance.

## Decision record

- **Status:** accepted direction; implementation not started
- **Decision/date:** 2026-08-17, Phase 2 (#108)
- **Approvers:** backend optimization program owner
- **Follow-up tickets:** open one OpenAPI/schema delivery ticket before
  implementation; keep `bulkCreateTransactions` unchanged and do not combine
  the staging workflow with #115 or #120
