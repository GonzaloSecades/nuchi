# Technical Debt Tickets

- Source: 11.07.Transactions-Feature.md
- Ticket count: 4

## Priority Order

## P0 — Enforce ownership validation on transaction creation

- Problem: Transaction create endpoints do not verify that the provided account/category belongs to the authenticated user, creating a cross-user data access risk.
- Evidence (quote): "Missing ownership validation on create" (Risks & Rollout Considerations → High-Impact Risks)
- Why it matters: Without ownership checks, users could write transactions into other users’ accounts or categories.
- Scope / Proposed fix:
  - Add ownership checks for `accountId` and optional `categoryId` prior to insert.
  - Reuse account/category ownership queries similar to read/update paths.
  - Apply validation to both single-create and bulk-create endpoints.
- Acceptance criteria:
  - [ ] Create endpoints reject account/category IDs not owned by the authenticated user.
  - [ ] Automated test covers unauthorized ownership attempts.
- Effort: S
- Owner suggestion: Backend

## P1 — Include filters in transaction list query key

- Problem: Transaction list caching ignores filter parameters, which can return stale results when filters change.
- Evidence (quote): "Inconsistent query key usage" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Users may see incorrect data when switching date ranges or account filters.
- Scope / Proposed fix:
  - Include `from`, `to`, and `accountId` in the React Query `queryKey`.
  - Ensure cache invalidation aligns with the new key.
- Acceptance criteria:
  - [ ] Changing filters triggers a new fetch and reflects correct results.
  - [ ] Query key includes filter parameters.
- Effort: S
- Owner suggestion: Frontend

## P2 — Implement summary cache invalidation hooks

- Problem: Summary invalidation is marked TODO in mutations, leaving related summary views potentially stale.
- Evidence (quote): "TODOs in mutation hooks" (Technical Debt Assessment → Introduced Technical Debt)
- Why it matters: Summary data can drift from transaction updates, reducing trust in analytics.
- Scope / Proposed fix:
  - Identify summary query keys impacted by transactions.
  - Add invalidation calls in create/edit/delete/bulk operations.
- Acceptance criteria:
  - [ ] Summary queries update after transaction mutations.
  - [ ] TODO comments removed from mutation hooks.
- Effort: S
- Owner suggestion: Full-stack

## P2 — Add automated tests for transactions API and utilities

- Problem: Automated coverage is missing for core CRUD flows and conversion helpers.
- Evidence (quote): "API tests for CRUD and bulk endpoints" (Testing Recommendations → Automated Testing Needs)
- Why it matters: Lack of tests increases regression risk in financial data handling.
- Scope / Proposed fix:
  - Add API tests for create/edit/delete/bulk with auth/ownership cases.
  - Add unit tests for amount conversion helpers and currency formatting.
- Acceptance criteria:
  - [ ] Automated tests cover CRUD and bulk endpoints including auth/ownership.
  - [ ] Conversion helpers have unit tests for positive/negative values.
- Effort: M
- Owner suggestion: Full-stack
