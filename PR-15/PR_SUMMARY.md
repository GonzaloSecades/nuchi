# PR Overview: Dashboard Account/Date Filters with Query-Scoped Caching

## Summary

This PR adds dashboard-level account and date filters in the header, persists those filters in URL query params, and wires the same params into data fetching so transaction and summary views stay aligned with user-selected scope; it also broadens summary cache invalidation after mutation flows and updates chart empty-state behavior for all-zero periods.

**Key Statistics** bullets:

- Files changed: 20 files (+260/ -38)
- Commits: 3 (`Add account filter`, `Add date filter`, `some todos remaining`)
- Backend: None (no API route or server business-logic file changes in this PR)
- Frontend: Next.js client header filters, React Query key/invalidation updates, Radix UI popover/select integration
- Database: None (no schema, migration, or ORM model changes)

---

## Key Changes by Area

### 1. Frontend - Data/State (React Query, Next Navigation, URL Query State)

**Files:**

- `components/account-filter.tsx` - adds account select state sourced from URL and pushes updated `accountId` query params
- `components/date-filter.tsx` - adds date-range parsing/apply/reset flow tied to `from`/`to` query params
- `features/transactions/api/use-get-transactions.ts` - scopes transactions query cache key by `from`, `to`, and `accountId`
- `features/summary/use-get-summary.ts` - keeps summary query key param-scoped and removes stale TODO
- `features/accounts/api/use-delete-account.ts` - invalidates `summary` cache after account delete
- `features/accounts/api/use-edit-account.ts` - invalidates `summary` cache after account edit
- `features/categories/api/use-bulk-delete-categories.ts` - invalidates `summary` cache after category bulk delete
- `features/categories/api/use-delete-category.ts` - invalidates `summary` cache after category delete
- `features/categories/api/use-edit-category.ts` - invalidates `summary` cache after category edit
- `features/transactions/api/use-bulk-create-transactions.ts` - invalidates `summary` cache after bulk create
- `features/transactions/api/use-bulk-delete-transactions.ts` - invalidates `summary` cache after bulk delete
- `features/transactions/api/use-create-transaction.ts` - invalidates `summary` cache after create
- `features/transactions/api/use-delete-transaction.ts` - invalidates `summary` cache after delete
- `features/transactions/api/use-edit-transaction.ts` - invalidates `summary` cache after edit

**Changes:**

- Adds account filter selection that keeps existing date params while changing `accountId`.
- Adds calendar-based date-range filter that serializes dates as `yyyy-MM-dd` and applies them to URL state.
- Uses query-param-aware React Query keys for transactions to avoid cross-filter cache collisions.
- Expands summary invalidation across most create/edit/delete mutation success paths.
- Uses current query params as single source of truth for fetch scoping in transactions/summary hooks.

**Rationale:**

- Makes filtering shareable/bookmarkable through URL state.
- Improves cache correctness when switching between date/account scopes.
- Reduces stale summary widgets after transactional mutations.

### 2. Frontend - UI Components (React, Radix UI, Dashboard Layout)

**Files:**

- `components/filters.tsx` - composes account and date filters into a single reusable header block
- `components/header.tsx` - injects `Filters` beneath welcome message
- `components/account-filter.tsx` - new select-based account filter control
- `components/date-filter.tsx` - new popover calendar with Apply/Reset controls
- `components/ui/popover.tsx` - exports `PopoverClose` and normalizes style/exports
- `components/chart.tsx` - treats all-zero income/expense datasets as empty state

**Changes:**

- Adds a visible filters row to dashboard header layout.
- Introduces account selector loading-disable behavior while accounts/summary are loading.
- Introduces date popover with two-month range calendar and explicit apply/reset actions.
- Exposes `PopoverClose` primitive to close popover via action buttons.
- Updates chart empty-state logic to hide chart rendering for zeroed datasets.

**Rationale:**

- Surfaces filtering controls at the top navigation context where users expect global data scope controls.
- Keeps filter interactions explicit (Apply/Reset) to avoid accidental range changes.
- Improves chart readability when filtered periods return no meaningful values.

### 3. Developer Experience & Configuration (Dependency Management)

**Files:**

- `package.json` - adds `query-string@9.0.0`
- `bun.lock` - records transitive lock updates (`decode-uri-component`, `filter-obj`, `split-on-first`)

**Changes:**

- Introduces URL query serialization dependency used by filter components.
- Updates lockfile for deterministic installs.

**Rationale:**

- Avoids manual query-string building/parsing edge cases in filter URL updates.

---

## Rationale

### Why This Architecture?

1. URL-driven filter state
   - Query params (`from`, `to`, `accountId`) create a stable contract between UI controls and data hooks.
2. Query-key scoping in data hooks
   - Param-scoped keys separate cache entries by filter state and reduce stale reuse.
3. Shared filter composition in header
   - Centralizing controls in `Header` provides consistent navigation-level scoping for dashboard pages.
4. Broad mutation invalidation
   - Invalidating `summary` on write operations improves coherence between CRUD actions and analytics cards/charts.

### Business Value

- Users can quickly segment financial views by account and period from a single header location.
- Filter state can be shared via URL and restored on refresh/navigation.
- Dashboard totals/charts refresh more reliably after create/edit/delete flows.

---

## Risks & Rollout Considerations

### High-Impact Risks

1. No high-impact production-stability risk identified from this diff
   - Issue: Current changes are UI/query-state oriented and do not modify persistence schema or auth boundaries.
   - Mitigation: Validate end-to-end filter + mutation flows in staging before release.
   - Rollback: Revert header filter components and query-key/invalidation deltas in this PR range.

### Medium-Impact Risks

1. Date picker local state can diverge from URL-driven state after navigation
   - Issue: `DateFilter` initializes local `date` from params but does not resync when params change externally.
   - Mitigation: Sync component state from search params with an effect keyed by `from`/`to`.
   - Rollback: Keep URL as display source and remove local-state dependence for selected range.

2. Account filter currently couples control state to summary loading on all dashboard routes
   - Issue: `AccountFilter` calls `useGetSummary()` to derive `isLoadingSummary`, even where summary data is not rendered.
   - Mitigation: Decouple disabled state from summary query or gate summary hook usage to overview route.
   - Rollback: Remove summary-loading dependency from filter disable logic.

3. Assumption: Summary invalidation coverage may still be incomplete for `useBulkDeleteAccounts`
   - Issue: This PR adds multiple summary invalidations, but `useBulkDeleteAccounts` still invalidates only `accounts`.
   - Mitigation: Add summary invalidation parity check across all write mutations.
   - Rollback: Force refresh dashboard summary after bulk account delete until hook parity is added.

### Low-Impact Risks

1. Debug logs remain in account filter handler
   - Issue: `console.log(query)` and `console.log(url)` are still present in `components/account-filter.tsx`.
   - Mitigation: Remove logs or guard with development-only utility.
   - Rollback: N/A (cleanup-only).

2. Reset behavior applies default date range rather than clearing date params
   - Issue: `onReset` pushes formatted default dates, which may not match expected "clear filters" semantics.
   - Mitigation: Decide product behavior and either clear params or keep explicit defaults consistently.
   - Rollback: Keep existing behavior and document it in UI/help text.

### Deployment Considerations

**Pre-Deployment:**

- [ ] Validate account/date filter interactions on overview and transactions pages.
- [ ] Validate URL deep-linking (`from`, `to`, `accountId`) across refresh/back-forward navigation.
- [ ] Validate summary and transactions refresh after create/edit/delete and bulk mutation operations.
- [ ] Remove debug logs before production deploy.

**Post-Deployment Monitoring:**

- [ ] Monitor dashboard/transactions API request volume for unexpected increases from header-level hooks.
- [ ] Monitor client error logs around date parsing/query serialization paths.
- [ ] Monitor UX feedback about date Reset semantics.

**Rollback Plan:**

- Revert `components/account-filter.tsx`, `components/date-filter.tsx`, `components/filters.tsx`, and `components/header.tsx`.
- Revert query-key and invalidation changes under `features/**/api/*`.
- Reinstall lockfile state from previous commit and redeploy.

---

## Technical Debt Assessment

### Introduced Technical Debt

1. Debug logging left in account filter handler

- Issue: `console.log` calls remain in production component path.
- Impact: Noisy logs and accidental leakage of URL/query context in client consoles.
- Recommendation: Remove logs or wrap with development-only logging utility.
- Effort: S

2. Date filter state synchronization gap

- Issue: Local date state is initialized from params once and may drift from URL updates.
- Impact: Calendar selected state can become inconsistent with displayed active range.
- Recommendation: Add state synchronization from `useSearchParams` changes.
- Effort: S

3. Reset semantics are ambiguous

- Issue: Reset pushes default date values instead of clearing date params.
- Impact: Potential mismatch between user expectation ("clear") and actual behavior ("set defaults").
- Recommendation: Define/reset contract and implement consistently (clear or explicit default).
- Effort: S

4. Assumption: Summary invalidation coverage may still be incomplete for `useBulkDeleteAccounts`

- Issue: Mutation invalidation was expanded broadly, but one account bulk-delete path appears unaligned.
- Impact: Potential stale summary widgets after bulk account deletion.
- Recommendation: Standardize invalidation policy and audit all mutation hooks.
- Effort: S

5. Account filter currently couples control state to summary loading on all dashboard routes

- Issue: Filter disabled-state logic depends on summary loading even when summary is not displayed.
- Impact: Extra background requests and unnecessary control blocking in non-summary routes.
- Recommendation: Route-scope summary fetching or decouple filter disabled criteria.
- Effort: M

### Mitigated Technical Debt

- Replaces unscoped transactions cache key with filter-aware cache key to reduce stale data reuse.
- Adds `summary` invalidation to multiple mutation hooks that previously left TODOs.
- Consolidates filter controls into reusable components instead of page-specific ad-hoc implementations.

### Recommended Next Steps

**Immediate (This Sprint):**

- Remove debug logs from `components/account-filter.tsx`.
- Add summary invalidation in `useBulkDeleteAccounts` for parity.
- Confirm and document date reset behavior (clear vs default range).

**Short-term (Next Sprint):**

- Add URL-param-to-local-state synchronization tests for `DateFilter`.
- Add integration tests for filter-driven fetch scoping on overview and transactions pages.
- Decouple account filter disabled state from summary loading where not required.

**Medium-term (Within Quarter):**

- Define a shared React Query invalidation policy helper for mutation hooks.
- Add route-level performance baselines for header-mounted data dependencies.

**Long-term (Future Quarters):**

- Consider central filter-state management abstraction (URL + UI sync) to reduce duplicated query-state logic.

---

## Testing Recommendations

### Manual Testing Checklist

- [ ] Open dashboard overview, change account filter, and confirm URL/query + cards/charts update.
- [ ] Open transactions page with filters and verify table rows respect `from`/`to`/`accountId`.
- [ ] Use browser back/forward after changing dates and verify displayed range and selected calendar range are consistent.
- [ ] Edit/delete/create transactions and categories, then verify summary cards/charts refresh.
- [ ] Bulk-delete accounts and verify whether summary refreshes correctly (current assumption risk).
- [ ] Confirm all-zero filtered periods show expected chart empty state messaging.

### Automated Testing Needs

- [ ] Component tests for `AccountFilter` URL serialization and "All Accounts" behavior.
- [ ] Component tests for `DateFilter` Apply/Reset behavior and state synchronization with URL changes.
- [ ] Hook/integration tests verifying transactions query key partitioning by filter params.
- [ ] Mutation-hook tests validating `summary` invalidation parity across all CRUD/bulk operations.

---

## Dependencies & Prerequisites

### Required Environment Variables

- None introduced by this PR.

### External Service Dependencies

- Existing dashboard stack only (Next.js app router, Hono API client, React Query, Radix UI primitives).

### Breaking Changes

- None.

---

## Performance Considerations

### Frontend

- Query-key scoping improves cache correctness but can increase cache entry count for many filter combinations.
- Header-mounted account filter now includes summary loading coupling, which may add background summary requests on non-overview pages.

### API

- No direct API contract or endpoint implementation changes in this PR.

### Database

- No schema/query-layer changes introduced.

---

## Security Considerations

- No authn/authz boundary changes in this PR.
- No new sensitive data fields introduced in URL params; filters use account IDs and date strings already used by existing API contracts.
- Server-side validation remains important for `from`/`to`/`accountId` query inputs regardless of client formatting behavior.

---

## Conclusion

This PR delivers useful dashboard filtering and better cache coherence for analytics-driven UI, with moderate follow-up needed around filter-state sync semantics and invalidation parity.

- Deployment Status: ⚠️ Needs Follow-up (recommended to address debug logs and invalidation/sync gaps before production hardening)

- Generated: February 16, 2026
- Branch: 10.SearchFilters
- Base: origin/master
- Files Changed: 20 files (+260/ -38)
