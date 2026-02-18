# Technical Debt Tickets

- Source: `PR-15/PR_SUMMARY.md`
- Ticket count: 6 (<=10)

## Priority Order

## P1 — Ensure summary invalidation parity for account bulk delete

- Problem: Summary cache invalidation appears inconsistent across mutation paths, with account bulk-delete called out as an uncovered case.
- Evidence (quote): "Assumption: Summary invalidation is still incomplete for account bulk delete path" (`Risks & Rollout Considerations` -> `Medium-Impact Risks`)
- Why it matters: Dashboard totals/charts can remain stale after bulk account deletion, creating correctness drift.
- Scope / Proposed fix:
  - Add `summary` invalidation in `useBulkDeleteAccounts` success handler.
  - Audit all mutation hooks and document required invalidation keys.
  - Add one shared helper/checklist to enforce invalidation consistency.
- Acceptance criteria:
  - [ ] Bulk account delete triggers summary refetch for current filter context.
  - [ ] All write mutation hooks follow the documented invalidation policy.
- Effort: S
- Owner suggestion: Full-stack

## P2 — Replace hardcoded All time lower bound with real data boundary

- Problem: The All time preset uses a fixed year-2000 start date in client logic.
- Evidence (quote): "Hardcoded baseline for \"All time\" preset" (`Technical Debt Assessment` -> `Introduced Technical Debt`)
- Why it matters: Transactions earlier than the fixed date are excluded, producing incomplete analytics.
- Scope / Proposed fix:
  - Source earliest transaction date from backend metadata/query.
  - Use that value (or configured product min date) for All time preset.
  - Add fallback behavior when no transactions exist.
- Acceptance criteria:
  - [ ] All time preset includes full historical range for the user.
  - [ ] Behavior is deterministic when no historical data exists.
- Effort: M
- Owner suggestion: Full-stack

## P2 — Decouple account filter disabled logic from summary loading on non-summary routes

- Problem: Account filter interactivity is tied to summary hook loading globally in dashboard header.
- Evidence (quote): "Header filter disables account selector based on summary loading globally" (`Risks & Rollout Considerations` -> `Medium-Impact Risks`)
- Why it matters: Adds avoidable background requests and can block filter interaction unnecessarily.
- Scope / Proposed fix:
  - Remove direct summary-loading dependency from `AccountFilter` disabled criteria.
  - Gate summary dependency by route where summary is actually rendered.
  - Add route-level integration tests for filter responsiveness.
- Acceptance criteria:
  - [ ] Account filter remains responsive on non-summary pages.
  - [ ] Non-summary routes do not trigger unnecessary summary fetches from header filters.
- Effort: M
- Owner suggestion: Frontend

## P2 — Clarify and enforce reset semantics for date filtering

- Problem: Reset currently applies a default range instead of clearing date params, but behavior intent is not explicitly settled.
- Evidence (quote): "Reset behavior enforces default range instead of clearing date params" (`Risks & Rollout Considerations` -> `Low-Impact Risks`)
- Why it matters: Ambiguous behavior causes UX confusion and inconsistent expectations in QA/support.
- Scope / Proposed fix:
  - Decide canonical reset behavior (clear vs default range).
  - Implement chosen behavior consistently in UI and URL state.
  - Add tests for reset outcomes and documentation text.
- Acceptance criteria:
  - [ ] Reset behavior is explicitly documented and consistent in UI + URL.
  - [ ] Automated tests validate the chosen reset contract.
- Effort: S
- Owner suggestion: Frontend

## P3 — Remove dead commented DateFilter implementation block

- Problem: Obsolete commented JSX from prior implementation remains in production component.
- Evidence (quote): "Large commented legacy JSX block left in date filter" (`Risks & Rollout Considerations` -> `Low-Impact Risks`)
- Why it matters: Dead commented code increases cognitive load and maintenance friction.
- Scope / Proposed fix:
  - Delete the commented legacy popover block in `components/date-filter.tsx`.
  - Keep current implementation only.
  - Add lint rule/policy for avoiding large commented dead code blocks.
- Acceptance criteria:
  - [ ] `components/date-filter.tsx` contains no large dead commented implementation block.
  - [ ] Team coding standards/lint checks discourage reintroducing similar dead blocks.
- Effort: S
- Owner suggestion: Frontend

## P3 — Remove debug logs from account filter handler

- Problem: Debug `console.log` calls remain in user-facing filter change path.
- Evidence (quote): "Debug logs still present in account filter" (`Risks & Rollout Considerations` -> `Low-Impact Risks`)
- Why it matters: Console noise reduces debugging signal quality and weakens production logging hygiene.
- Scope / Proposed fix:
  - Remove `console.log(query)` and `console.log(url)` from `components/account-filter.tsx`.
  - If required, replace with dev-only logger wrapper.
  - Add lint guard for console usage in production components.
- Acceptance criteria:
  - [ ] No raw debug `console.log` calls remain in account filter component.
  - [ ] Lint/static checks prevent accidental reintroduction.
- Effort: S
- Owner suggestion: Frontend
