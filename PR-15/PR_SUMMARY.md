# PR Overview: Header Filters Expansion with Preset Date Ranges and Query-Scoped Refresh

## Summary

This PR evolves dashboard filtering by introducing account/date URL-driven filters in the header, extending the date filter UX with preset ranges and controlled popover state, scoping transactions/summary fetching to query params, and broadening summary invalidation after write mutations; it also includes branch-level review artifact files committed into `PR-15/`.

**Key Statistics** bullets:

- Files changed: 22 files (+854/ -38)
- Commits: 5 (`Enhance DateFilter component with preset date ranges and improved state management`, `codex review files`, `some todos remaining`, `Add date filter`, `Add account filter`)
- Backend: None (no backend route/controller/database-query logic changed in this PR)
- Frontend: Next.js client filters, React Query cache key/invalidation updates, Radix popover/select usage, preset date-range UX
- Database: None (no migration/schema/model changes)

---

## Key Changes by Area

### 1. Frontend - Data/State (React Query, URL query state, Next navigation)

**Files:**

- `components/account-filter.tsx` - reads/writes `accountId`, preserves `from`/`to`, and updates URL query state
- `components/date-filter.tsx` - parses URL dates, manages range/month/open state, applies filter query params, and adds preset range logic
- `features/transactions/api/use-get-transactions.ts` - updates query key to include `from`, `to`, and `accountId`
- `features/summary/use-get-summary.ts` - keeps param-scoped summary key and uses query params as request input
- `features/accounts/api/use-delete-account.ts` - invalidates `summary` after delete
- `features/accounts/api/use-edit-account.ts` - invalidates `summary` after edit
- `features/categories/api/use-bulk-delete-categories.ts` - invalidates `summary` after bulk delete
- `features/categories/api/use-delete-category.ts` - invalidates `summary` after delete
- `features/categories/api/use-edit-category.ts` - invalidates `summary` after edit
- `features/transactions/api/use-bulk-create-transactions.ts` - invalidates `summary` after bulk create
- `features/transactions/api/use-bulk-delete-transactions.ts` - invalidates `summary` after bulk delete
- `features/transactions/api/use-create-transaction.ts` - invalidates `summary` after create
- `features/transactions/api/use-delete-transaction.ts` - invalidates `summary` after delete
- `features/transactions/api/use-edit-transaction.ts` - invalidates `summary` after edit

**Changes:**

- Adds account-level filtering that persists in URL query params.
- Adds date-range URL serialization (`yyyy-MM-dd`) with explicit Apply/Reset actions.
- Adds preset selectors: Today, Yesterday, This week, Last week, This month, Last month, This year, Last year, All time.
- Re-seeds date/month local state from URL each time the filter popover opens.
- Scopes transactions query caching by active filter params to avoid stale cross-filter cache reuse.
- Expands summary invalidation across most create/edit/delete mutation success handlers.

**Rationale:**

- Keeps filters shareable and refresh-safe through URL state.
- Improves cache correctness under rapid filter switching.
- Reduces stale summary UI after transactional writes.

### 2. Frontend - UI Components (Dashboard layout, filters, chart empty-state)

**Files:**

- `components/filters.tsx` - new wrapper combining account and date controls
- `components/header.tsx` - injects filters into dashboard header under welcome message
- `components/account-filter.tsx` - adds account selector control with loading-based disabling
- `components/date-filter.tsx` - implements two-pane popover (preset rail + two-month calendar) and action buttons
- `components/ui/popover.tsx` - exports `PopoverClose` primitive used in date actions
- `components/chart.tsx` - expands empty-state logic to treat all-zero periods as no-data

**Changes:**

- Adds visible global filters area in header.
- Introduces popover UX with preset quick-selects and controlled month navigation.
- Keeps apply/reset explicit instead of auto-submitting selection changes.
- Uses icon-only reset button and apply action inside popover footer.
- Prevents chart rendering for datasets where every day has `income=0` and `expenses=0`.

**Rationale:**

- Improves discoverability and speed of common time-range filtering.
- Preserves predictable user intent with explicit apply step.
- Avoids misleading "flat-zero" visualizations in charts.

### 3. Developer Experience & Configuration (Dependencies, lockfile)

**Files:**

- `package.json` - adds `query-string@9.0.0`
- `bun.lock` - lockfile updates for `query-string` and transitive packages

**Changes:**

- Introduces library-based query serialization for stable URL construction.
- Updates dependency graph entries (`decode-uri-component`, `filter-obj`, `split-on-first`).

**Rationale:**

- Reduces manual query-string handling complexity and encoding edge cases.

### 4. Documentation & PR Artifacts (Review docs)

**Files:**

- `PR-15/PR_SUMMARY.md` - PR technical summary artifact
- `PR-15/TECH_DEBT_TICKETS.md` - derived debt-ticket artifact

**Changes:**

- Adds branch-local review documentation files committed in this PR.

**Rationale:**

- Captures review output and debt tracking directly in repository context for this branch.

---

## Rationale

### Why This Architecture?

1. URL-as-source-of-truth filtering
   - `from`/`to`/`accountId` query params align navigation state and fetch state.
2. Preset-first date UX layered over range picker
   - Quick ranges reduce clicks for common analytics windows while retaining custom range support.
3. Query-key partitioning
   - Param-scoped keys isolate cache entries per filter context.
4. Broad summary invalidation strategy
   - Summary data is refreshed after most mutation paths to preserve dashboard consistency.

### Business Value

- Faster insight slicing by account and period from a single header control area.
- Better daily usability via one-click date presets.
- Fewer stale analytics scenarios after transaction/account/category updates.

---

## Risks & Rollout Considerations

### High-Impact Risks

1. No high-impact production outage/security risk directly introduced by this diff
   - Issue: Changes are primarily UI/query-state/caching behavior.
   - Mitigation: Validate end-to-end filter + mutation flows in staging with realistic data.
   - Rollback: Revert header filter/date preset components and query-key/invalidation deltas.

### Medium-Impact Risks

1. Assumption: Summary invalidation is still incomplete for account bulk delete path
   - Issue: `useBulkDeleteAccounts` remains unmodified in this PR while other mutations now invalidate `summary`.
   - Mitigation: Add `summary` invalidation parity check across all mutation hooks.
   - Rollback: Trigger manual refresh or route reload post bulk-delete until parity patch lands.

2. "All time" preset uses a hardcoded start date
   - Issue: Preset currently sets `from` to `new Date(2000, 0, 1)`.
   - Mitigation: Replace with product/system minimum transaction date from backend or config.
   - Rollback: Temporarily hide/rename preset if it causes user confusion.

3. Header filter disables account selector based on summary loading globally
   - Issue: `AccountFilter` uses `useGetSummary()` loading state even on routes where summary is not the primary content.
   - Mitigation: Decouple disable criteria from summary loading or scope summary loading by route.
   - Rollback: Remove summary dependency from disabled prop.

### Low-Impact Risks

1. Debug logs still present in account filter
   - Issue: `console.log(query)` and `console.log(url)` remain in client component.
   - Mitigation: Remove logs or gate through dev-only logger utility.
   - Rollback: N/A.

2. Large commented legacy JSX block left in date filter
   - Issue: Previous popover implementation remains commented in `components/date-filter.tsx`.
   - Mitigation: Remove dead commented block to reduce maintenance noise.
   - Rollback: N/A.

3. Reset behavior enforces default range instead of clearing date params
   - Issue: Reset pushes default `from`/`to` values.
   - Mitigation: Confirm product intent and align reset semantics/documentation.
   - Rollback: Keep current behavior and document as "Reset to last 30 days".

### Deployment Considerations

**Pre-Deployment:**

- [ ] Validate all presets produce expected `from`/`to` URL values.
- [ ] Validate account filter + date filter interactions on overview and transactions pages.
- [ ] Validate back/forward navigation and deep-linking with query params.
- [ ] Validate summary/charts refresh after create/edit/delete and bulk operations.
- [ ] Remove debug logs and dead commented code before production release.

**Post-Deployment Monitoring:**

- [ ] Monitor dashboard/transactions request rates for filter-related overfetch.
- [ ] Monitor client error logs around date parsing and preset application.
- [ ] Monitor support feedback about All time and Reset semantics.

**Rollback Plan:**

- Revert `components/date-filter.tsx`, `components/account-filter.tsx`, `components/filters.tsx`, and `components/header.tsx`.
- Revert query-key/invalidation updates in `features/**/api` hooks.
- Revert dependency additions in `package.json` and `bun.lock` if filter feature is disabled.

---

## Technical Debt Assessment

### Introduced Technical Debt

1. Hardcoded baseline for "All time" preset

- Issue: All-time range starts at a fixed year-2000 date in UI logic.
- Impact: Data before 2000 (or product-specific true start) is silently excluded.
- Recommendation: Resolve minimum selectable date from backend metadata or first-transaction query.
- Effort: M

2. Mutation invalidation policy is still inconsistent

- Issue: Most mutation hooks now invalidate `summary`, but account bulk-delete path appears to lag.
- Impact: Possible stale dashboard aggregates after certain operations.
- Recommendation: Introduce and enforce a centralized invalidation checklist/helper.
- Effort: S

3. Filter control coupling to summary loading on all dashboard pages

- Issue: Account filter disabled state depends on summary hook loading globally.
- Impact: Extra fetches and avoidable input blocking.
- Recommendation: Route-scope summary dependencies or refactor disabled criteria.
- Effort: M

4. Dead commented UI block in DateFilter

- Issue: Obsolete commented JSX remains in production file.
- Impact: Increased noise and slower maintenance/readability.
- Recommendation: Remove commented legacy block.
- Effort: S

5. Debug logging in client path

- Issue: `console.log` statements remain in filter change handler.
- Impact: Noisy production consoles and weak logging hygiene.
- Recommendation: Remove or gate with dev-only logger.
- Effort: S

### Mitigated Technical Debt

- Replaces unscoped transaction cache key with filter-aware key.
- Clears multiple existing TODOs by adding summary invalidation across mutation hooks.
- Improves date filter state handling by re-seeding local state from URL on popover open.

### Recommended Next Steps

**Immediate (This Sprint):**

- Remove debug logs and commented legacy block from date/account filter components.
- Add missing summary invalidation parity for account bulk-delete.
- Confirm and document reset/all-time product semantics.

**Short-term (Next Sprint):**

- Add integration tests for preset behavior, URL serialization, and filter interactions across pages.
- Refactor filter disable logic to avoid non-essential summary-loading coupling.

**Medium-term (Within Quarter):**

- Create shared React Query invalidation helper/policy across mutation hooks.
- Add telemetry around filter usage/preset selection to guide UX tuning.

**Long-term (Future Quarters):**

- Consider centralized dashboard filter-state module shared across header and feature pages.

---

## Testing Recommendations

### Manual Testing Checklist

- [ ] Verify each preset (today/yesterday/week/month/year/all-time) updates URL and displayed range correctly.
- [ ] Verify custom date-range selection persists after Apply and survives refresh.
- [ ] Verify Reset behavior and confirm expected default range semantics.
- [ ] Verify account filter keeps date params intact when changing account.
- [ ] Verify overview charts/cards and transactions table respond to combined account/date filters.
- [ ] Verify summary updates after create/edit/delete and bulk mutation flows.
- [ ] Verify all-zero datasets show "No data for this period" in chart card.

### Automated Testing Needs

- [ ] Component tests for `DateFilter` presets, apply/reset, and URL serialization.
- [ ] Component tests for `AccountFilter` "All Accounts" behavior and param preservation.
- [ ] Hook/integration tests for transactions query-key partitioning by filters.
- [ ] Mutation-hook tests for summary invalidation parity across all write paths.

---

## Dependencies & Prerequisites

### Required Environment Variables

- None introduced by this PR.

### External Service Dependencies

- Existing Next.js + Hono client + React Query + Radix stack only.

### Breaking Changes

- None identified.

---

## Performance Considerations

### Frontend

- Preset/date popover adds UI complexity but remains client-local.
- Filtered query-key partitioning may increase cache entries for many date/account combinations.
- Summary-loading coupling in header may trigger unnecessary requests on non-summary routes.

### API

- No endpoint logic changes in this PR.

### Database

- No schema/query changes in this PR.

---

## Security Considerations

- No authentication or authorization flow changes.
- No new secret material introduced in URL/query params.
- Query params remain user-controllable; backend validation should continue to enforce date/account constraints.

---

## Conclusion

This PR materially improves filter usability and cache correctness with preset date ranges and broader summary refresh handling, with follow-up needed to clean logging/dead code and finalize invalidation/preset semantics.

- Deployment Status: ⚠️ Needs Follow-up (recommended to close medium/low debt items before production hardening)

- Generated: February 18, 2026
- Branch: 10.SearchFilters
- Base: origin/master
- Files Changed: 22 files (+854/ -38)
