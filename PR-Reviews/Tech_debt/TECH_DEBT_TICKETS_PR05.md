# Technical Debt Tickets

- Source: `05.Accounts-setups.md`
- Ticket count: 10 (<=10)

## Priority Order

## P0 — Add CSRF protection for mutation endpoints

- Problem: The system explicitly calls out missing CSRF protection, which can allow cross-site request forgery against authenticated users for destructive/mutating actions.
- Evidence (quote): "⚠️ No CSRF protection (consider adding tokens for mutations)" (Security Considerations → Known Security Gaps)
- Why it matters: CSRF can cause unauthorized state-changing requests, leading to data loss or account manipulation.
- Scope / Proposed fix:
  - Decide on an approach compatible with Clerk + Next.js (e.g., same-site cookies + CSRF token double-submit, or framework middleware).
  - Add CSRF token issuance and verification for all mutating routes (create/edit/delete/bulk-delete).
  - Ensure RPC client/hook layer includes token automatically.
  - Add tests for “missing/invalid CSRF token → request rejected”.
- Acceptance criteria:
  - [ ] All mutating endpoints reject requests without a valid CSRF token.
  - [ ] Legitimate UI mutations continue to work end-to-end.
- Effort: M
- Owner suggestion: Full-stack

## P0 — Add rate limiting for bulk delete (and other sensitive endpoints)

- Problem: The API relies on auth-level protections but has no endpoint-specific rate limiting; the file notes bulk delete could be abused.
- Evidence (quote): "⚠️ No rate limiting on bulk delete" (Security Considerations → Known Security Gaps)
- Why it matters: Without rate limiting, an attacker or buggy client can cause DB load spikes or destructive repeated operations.
- Scope / Proposed fix:
  - Add rate limiting middleware for mutating endpoints, especially `/api/accounts/bulk-delete`.
  - Choose a storage/backing (e.g., Upstash Redis, in-memory for dev, edge-compatible store if deployed to edge runtime).
  - Return consistent error codes/messages when limited.
  - Add a small load/abuse test (even basic) to validate limiting behavior.
- Acceptance criteria:
  - [ ] Requests beyond configured thresholds receive a deterministic 429 response.
  - [ ] Normal user workflows are unaffected at expected usage patterns.
- Effort: M
- Owner suggestion: Backend

## P0 — Add audit trail for account changes (create/edit/delete/bulk-delete)

- Problem: The document calls out that account changes are not audited, leaving no traceability for destructive actions.
- Evidence (quote): "⚠️ No audit trail for account changes" (Security Considerations → Known Security Gaps)
- Why it matters: Without an audit log, investigating user-reported data loss or suspicious activity is difficult.
- Scope / Proposed fix:
  - Define an `audit_events` table (or equivalent) with event type, account id, actor userId, timestamp, and metadata.
  - Write audit entries inside the same logical operation for create/update/delete/bulk-delete.
  - Ensure bulk-delete records one event per account or a single event with list of ids (decide and document).
  - Add minimal admin/debug UI or tooling (optional) and/or logging integration.
- Acceptance criteria:
  - [ ] Account create/update/delete/bulk-delete operations produce corresponding audit records.
  - [ ] Audit records include enough info to attribute the action to a specific user and time.
- Effort: L
- Owner suggestion: Backend

## P0 — Add safeguards for irreversible bulk delete (soft delete + restore)

- Problem: Bulk delete is described as irreversible; while mitigated by a confirmation dialog, the file recommends considering soft deletes.
- Evidence (quote): "**Issue**: Bulk delete is irreversible" and "**Future Enhancement**: Consider soft deletes with restore functionality" (Risks & Rollout Considerations → High-Impact Risks)
- Why it matters: Irreversible deletes increase data loss risk and raise support costs.
- Scope / Proposed fix:
  - Introduce soft-delete fields (e.g., `deletedAt`, `deletedBy`) on `accounts`.
  - Update API queries to exclude soft-deleted records by default.
  - Add restore endpoint / UI affordance (optional for MVP, but at least backend support).
  - Provide a periodic hard-delete job for old soft-deleted rows (optional).
- Acceptance criteria:
  - [ ] Deleted accounts are not permanently removed from the DB immediately.
  - [ ] A restore path exists (API + minimal UI or API-only) and is validated.
- Effort: L
- Owner suggestion: Full-stack

## P1 — Add automated tests for API endpoints and hooks

- Problem: The document explicitly flags missing unit/integration tests for critical paths.
- Evidence (quote): "**Issue**: No automated tests for API endpoints or hooks" (Technical Debt Assessment → Introduced Technical Debt → Missing Unit Tests (High Priority))
- Why it matters: Lack of tests increases regression risk and slows confident delivery.
- Scope / Proposed fix:
  - Add API route tests for create, list, get-by-id, update, delete, and bulk-delete.
  - Add tests for React Query hooks (API mocking) to verify invalidation/error handling.
  - Start with happy path + 1–2 negative/security cases (401, 404, validation).
  - Wire into CI (even basic) to run on PRs.
- Acceptance criteria:
  - [ ] CI runs a test suite that covers the critical accounts flows.
  - [ ] At least one regression-preventing test exists per endpoint family (list/get/mutate).
- Effort: L
- Owner suggestion: QA

## P1 — Improve API error logging/observability for production debugging

- Problem: The file notes that DB errors return generic 500s, making user-reported issues hard to debug; it recommends structured logging and Sentry.
- Evidence (quote): "**Issue**: Generic error messages for database failures" and "**Recommendation**: Add structured logging and Sentry integration" (Technical Debt Assessment → Introduced Technical Debt → Error Handling Coverage (Medium Priority))
- Why it matters: Without sufficient logs/telemetry, diagnosing production issues is slow and unreliable.
- Scope / Proposed fix:
  - Add structured logging on API routes with request id, userId (where safe), and error code.
  - Standardize error responses (code/message) and log the underlying exception safely.
  - Integrate an error tracker (e.g., Sentry) for server-side exceptions.
  - Add a playbook snippet (README/docs) for how to trace an error.
- Acceptance criteria:
  - [ ] Server errors produce structured logs with enough context to debug.
  - [ ] Exceptions are captured in an error tracker in non-local environments.
- Effort: M
- Owner suggestion: DevOps

## P2 — Refactor API routes to consistently use centralized API error utilities

- Problem: The repo contains centralized error utilities but the document notes routes don’t consistently use them.
- Evidence (quote): "**Issue**: `lib/api-error.ts` created but not utilized in API routes" (Technical Debt Assessment → Introduced Technical Debt → API Error Utilities Not Used Everywhere (Medium Priority))
- Why it matters: Inconsistent patterns increase maintenance effort and make client-side handling less predictable.
- Scope / Proposed fix:
  - Define a single error response shape (and helper functions) in `lib/api-error.ts`.
  - Update `app/api/[[...route]]/*` routes to use the shared utilities.
  - Ensure clients/hooks map errors consistently to toast messages.
  - Add a small set of tests validating error shapes for common failure modes.
- Acceptance criteria:
  - [ ] All accounts endpoints return consistent error payloads via shared utilities.
  - [ ] No duplicated ad-hoc error response shapes remain in the accounts route.
- Effort: S
- Owner suggestion: Backend

## P2 — Add validation constraints for account name (min/max length)

- Problem: The document highlights missing validation beyond non-null; it suggests adding min/max length.
- Evidence (quote): "**Missing**: Min/max length, character restrictions" and "**Recommendation**: Add validation in next iteration" (Risks & Rollout Considerations → Low-Impact Risks → Missing Data Validation)
- Why it matters: Weak validation can lead to confusing UX (empty/huge names) and potentially data/display issues.
- Scope / Proposed fix:
  - Update `InsertAccountSchema` to enforce min length (e.g., 1) and max length (e.g., 100).
  - Ensure UI form shows validation errors clearly.
  - Add tests for boundary cases (empty string, > max length).
- Acceptance criteria:
  - [ ] Creating/updating an account rejects empty or overly long names.
  - [ ] The UI surfaces validation errors without a crash.
- Effort: S
- Owner suggestion: Full-stack

## P2 — Add pagination to accounts list endpoint

- Problem: The document notes the list endpoint has no pagination and suggests cursor-based pagination when scale increases.
- Evidence (quote): "**Issue**: No pagination on accounts list endpoint" and "**Future Enhancement**: Add cursor-based pagination" (Risks & Rollout Considerations → Medium-Impact Risks → Performance at Scale)
- Why it matters: Large account lists can degrade response times and client rendering performance.
- Scope / Proposed fix:
  - Add query params for pagination (cursor/limit).
  - Add ordering guarantees (e.g., by createdAt or id) to make cursors stable.
  - Update React Query hook(s) to support infinite query or paged fetch.
  - Update DataTable to work with server pagination (if required).
- Acceptance criteria:
  - [ ] API supports fetching accounts in pages with stable cursor semantics.
  - [ ] UI can load and navigate pages (or infinite scroll) without fetching all records.
- Effort: M
- Owner suggestion: Full-stack

## P3 — Define and document rollback procedure for database migration

- Problem: The document mentions rollback “can safely revert migration” but does not specify an actual executed, documented down-migration process.
- Evidence (quote): "**Rollback**: Can safely revert migration if issues arise" (Risks & Rollout Considerations → High-Impact Risks → Database Migration Risk)
- Why it matters: Clear rollback steps reduce deployment risk and shorten incident response time.
- Scope / Proposed fix:
  - Document the exact rollback steps for Drizzle migrations in this repo (commands + prerequisites).
  - Validate rollback in a staging/dev environment and note any caveats.
  - Add a short runbook section to the deployment checklist.
- Acceptance criteria:
  - [ ] A documented, reproducible rollback procedure exists for DB migration changes.
  - [ ] The procedure is verified at least once in a non-production environment.
- Effort: S
- Owner suggestion: DevOps
