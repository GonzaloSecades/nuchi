# Technical Debt Tickets

- Source: `06.Categories-setup.md`
- Ticket count: 5 (<=10)

## Priority Order

## P0 — Add rate limiting to categories API endpoints

- Problem: Categories endpoints have no rate limiting, leaving them vulnerable to abuse and potential service disruption.
- Evidence (quote): "No API Rate Limiting" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Unthrottled endpoints can cause outages or unexpected costs.
- Scope / Proposed fix:
  - Add rate limiting middleware for categories routes.
  - Choose a backing store (e.g., Vercel KV or Upstash Redis) for counters.
  - Apply limits consistently across all categories endpoints.
  - Add basic logging/metrics for throttled requests.
- Acceptance criteria:
  - [ ] Excessive requests to categories endpoints receive 429 responses with a clear error message.
  - [ ] Normal usage remains unaffected and documented rate limits are enforced.
- Effort: M
- Owner suggestion: DevOps

## P1 — Trim category name inputs in API and UI

- Problem: Category names are not trimmed, allowing confusing duplicates with leading/trailing spaces.
- Evidence (quote): "No Input Trimming/Sanitization" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Users can create visually identical categories that behave inconsistently.
- Scope / Proposed fix:
  - Trim whitespace in frontend form validation.
  - Trim whitespace in API validation before database insert/update.
  - Add tests for whitespace handling on create and update.
- Acceptance criteria:
  - [ ] Creating " Food" results in the stored value "Food".
  - [ ] Duplicate detection treats "Food" and " Food " as the same name.
- Effort: S
- Owner suggestion: Full-stack

## P2 — Add automated tests for categories feature

- Problem: The categories feature has no automated tests, increasing regression risk.
- Evidence (quote): "No Automated Testing" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Changes to categories will be risky and harder to validate quickly.
- Scope / Proposed fix:
  - Add unit tests for categories API handlers (happy path + duplicate handling).
  - Add hook tests for React Query mutations/queries.
  - Add a minimal E2E smoke test for create/edit/delete flow.
- Acceptance criteria:
  - [ ] Test suite covers at least one happy-path CRUD flow and one duplicate error case.
  - [ ] CI passes with the new tests enabled.
- Effort: L
- Owner suggestion: QA

## P2 — Implement soft deletes for categories

- Problem: Deletes are permanent with no recovery option.
- Evidence (quote): "No Soft Deletes" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Accidental deletions cause irrecoverable data loss for users.
- Scope / Proposed fix:
  - Add `deleted_at` to categories schema and migration.
  - Filter out soft-deleted records in queries.
  - Add restore/undo capability in UI.
- Acceptance criteria:
  - [ ] Deleted categories are no longer shown by default.
  - [ ] Soft-deleted categories can be restored.
- Effort: M
- Owner suggestion: Full-stack

## P3 — Centralize hard-coded error messages

- Problem: Error messages are hard-coded in the categories API, limiting reuse and localization.
- Evidence (quote): "Hard-Coded Error Messages" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Makes future i18n or consistent messaging harder.
- Scope / Proposed fix:
  - Extract error messages to a constants or i18n layer.
  - Update categories API to reference the centralized messages.
- Acceptance criteria:
  - [ ] Categories API error messages are sourced from a shared constants/i18n module.
  - [ ] No hard-coded strings remain in categories API error branches.
- Effort: S
- Owner suggestion: Backend
