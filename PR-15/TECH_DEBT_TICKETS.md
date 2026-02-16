# Technical Debt Tickets

- Source: `PR-15/PR_SUMMARY.md`
- Ticket count: 5 (<=10)

## Priority Order

## P1 — Complete summary invalidation parity for bulk account delete

- Problem: Summary cache invalidation appears incomplete for at least one mutation path, which can leave analytics views stale after destructive account operations.
- Evidence (quote): "Assumption: Summary invalidation coverage may still be incomplete for `useBulkDeleteAccounts`" (`Risks & Rollout Considerations` → `Medium-Impact Risks`)
- Why it matters: Users can see outdated KPI/chart data after bulk account deletion, reducing trust in dashboard accuracy.
- Scope / Proposed fix:
  - Add `queryClient.invalidateQueries({ queryKey: ['summary'] })` to `useBulkDeleteAccounts` success path.
  - Audit all mutation hooks for consistent invalidation policy across `summary`, `transactions`, and related entities.
  - Add a lightweight shared helper/pattern to reduce future invalidation drift.
- Acceptance criteria:
  - [ ] Bulk account delete triggers summary refetch for current filter scope.
  - [ ] Mutation-hook invalidation behavior is documented and consistent across CRUD/bulk paths.
- Effort: S
- Owner suggestion: Full-stack

## P2 — Synchronize DateFilter local state with URL query changes

- Problem: The date filter component initializes from URL params but may become out of sync when params change through navigation history or external URL updates.
- Evidence (quote): "Date picker local state can diverge from URL-driven state after navigation" (`Risks & Rollout Considerations` → `Medium-Impact Risks`)
- Why it matters: Users may see one active date range in label/URL and a different selected range in the calendar popover.
- Scope / Proposed fix:
  - Add an effect to update local `date` state when `from`/`to` search params change.
  - Normalize parse + fallback behavior in a dedicated utility for deterministic behavior.
  - Add component tests covering back/forward navigation and external param updates.
- Acceptance criteria:
  - [ ] Calendar selected range always matches active URL date params.
  - [ ] Navigation back/forward does not produce stale calendar selections.
- Effort: S
- Owner suggestion: Frontend

## P2 — Decouple account filter disabled state from summary loading on non-summary pages

- Problem: Account filter control currently depends on summary hook loading state globally in dashboard layout, potentially triggering unnecessary summary fetches and blocking interaction where summary is not rendered.
- Evidence (quote): "Account filter currently couples control state to summary loading on all dashboard routes" (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Adds avoidable request/load overhead and can degrade responsiveness on accounts/categories/transactions routes.
- Scope / Proposed fix:
  - Remove direct `useGetSummary` dependency from account filter disabled logic.
  - Gate summary loading usage by route, or pass loading state only from pages that render summary.
  - Add integration coverage for filter behavior on non-overview pages.
- Acceptance criteria:
  - [ ] Account filter remains usable on non-summary routes without waiting on summary fetch.
  - [ ] No extra summary API call is triggered solely by rendering header filters outside overview.
- Effort: M
- Owner suggestion: Frontend

## P2 — Define and implement explicit DateFilter reset contract

- Problem: Reset currently applies default dates instead of clearing date params, which can be interpreted differently by users and maintainers.
- Evidence (quote): "Reset behavior applies default date range rather than clearing date params" (`Risks & Rollout Considerations` → `Low-Impact Risks`)
- Why it matters: Ambiguous behavior creates UX inconsistency and complicates testing/support.
- Scope / Proposed fix:
  - Decide product contract: clear params vs enforce explicit default range.
  - Implement reset path to match contract and update button/help text as needed.
  - Add tests for reset behavior and resulting URL state.
- Acceptance criteria:
  - [ ] Reset behavior is documented and deterministic in UI + URL.
  - [ ] Automated tests validate chosen reset semantics.
- Effort: S
- Owner suggestion: Frontend

## P3 — Remove debug logging from account filter handler

- Problem: Debug `console.log` statements remain in a production-path client component.
- Evidence (quote): "Debug logs remain in account filter handler" (`Risks & Rollout Considerations` → `Low-Impact Risks`)
- Why it matters: Console noise reduces signal during troubleshooting and can expose internal query state in shared debugging sessions.
- Scope / Proposed fix:
  - Remove `console.log` statements from `components/account-filter.tsx`.
  - If logging is needed, introduce a development-only logger guard.
  - Add lint/static rule enforcement for stray console statements in production components.
- Acceptance criteria:
  - [ ] No raw console logs remain in account filter component.
  - [ ] CI/lint prevents future unintended console logging in app components.
- Effort: S
- Owner suggestion: Frontend
