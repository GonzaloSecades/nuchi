# Tech Debt Backlog

## Scanned Sources

- `PR14/TECH_DEBT_TICKETS.md`
- `PR-15/TECH_DEBT_TICKETS.md`
- `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`
- `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR06.md`
- `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR07.md`
- `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`
- `PR14/PR_SUMMARY.md`
- `PR-15/PR_SUMMARY.md`
- `PR-Reviews/05.Accounts-setups.md`
- `PR-Reviews/06.Categories-setup.md`
- `PR-Reviews/07.Transactions-Feature.md`
- `PR-Reviews/13.Upload-Transactions-Import.md`

## Resolved Since Review

- `Account bulk delete summary invalidation`
  - Status: resolved
  - Evidence: `useBulkDeleteAccounts` now invalidates `['summary']` in [features/accounts/api/use-bulk-delete-accounts.ts](/home/gonzalo/projects/nuchi/features/accounts/api/use-bulk-delete-accounts.ts)
  - Source: `PR-15/TECH_DEBT_TICKETS.md`

- `Transaction list query key missing filters`
  - Status: resolved
  - Evidence: transaction query key now includes `from`, `to`, and `accountId` in [features/transactions/api/use-get-transactions.ts](/home/gonzalo/projects/nuchi/features/transactions/api/use-get-transactions.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR07.md`

- `Summary invalidation TODOs in transaction mutations`
  - Status: resolved
  - Evidence: transaction create/edit/delete/bulk hooks invalidate `['summary']` in [features/transactions/api/use-create-transaction.ts](/home/gonzalo/projects/nuchi/features/transactions/api/use-create-transaction.ts), [features/transactions/api/use-edit-transaction.ts](/home/gonzalo/projects/nuchi/features/transactions/api/use-edit-transaction.ts), [features/transactions/api/use-delete-transaction.ts](/home/gonzalo/projects/nuchi/features/transactions/api/use-delete-transaction.ts), and [features/transactions/api/use-bulk-delete-transactions.ts](/home/gonzalo/projects/nuchi/features/transactions/api/use-bulk-delete-transactions.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR07.md`

- `Debug logs in account filter`
  - Status: resolved
  - Evidence: no `console.log` remains in [components/account-filter.tsx](/home/gonzalo/projects/nuchi/components/account-filter.tsx)
  - Source: `PR-15/TECH_DEBT_TICKETS.md`

- `fillMissingDays O(n*m) lookup`
  - Status: resolved
  - Evidence: `fillMissingDays` now uses a `Map` for keyed lookups in [lib/utils.ts](/home/gonzalo/projects/nuchi/lib/utils.ts)
  - Source: `PR14/TECH_DEBT_TICKETS.md`

## Open Tickets

### P0

- `Enforce ownership validation for transaction create and bulk-create`
  - Why now: this is the highest-risk auth bug still open; users can submit foreign `accountId` or `categoryId`.
  - Evidence: create and bulk-create insert directly without ownership checks in [app/api/[[...route]]/transactions.ts](/home/gonzalo/projects/nuchi/app/api/[[...route]]/transactions.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR07.md`

- `Harden CSV import validation and fail closed on malformed input`
  - Why now: current import flow trusts parsed rows too much and can produce invalid payloads.
  - Evidence: upload and import contracts still use `any`, no row schema validation exists, and date/amount conversion is unchecked in [app/(dashboard)/transactions/upload-button.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/upload-button.tsx) and [app/(dashboard)/transactions/import-card.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/import-card.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Add empty/header-only CSV guards before rendering import UI`
  - Why now: malformed uploads can break the import screen before the user can recover.
  - Evidence: `ImportCard` assumes `data[0]` exists and immediately computes `headers` and `body` in [app/(dashboard)/transactions/import-card.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/import-card.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Add CSRF protection for mutating endpoints`
  - Why now: all authenticated mutations remain vulnerable to cross-site request forgery.
  - Evidence: no CSRF middleware, token issuance, or verification exists across `app/api/[[...route]]/*.ts`
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`

- `Add rate limiting on mutating APIs`
  - Why now: write-heavy endpoints are unthrottled and easy to abuse.
  - Evidence: no rate limiting middleware exists for accounts, categories, or transactions routes under [app/api/[[...route]]](/home/gonzalo/projects/nuchi/app/api/[[...route]])
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`, `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR06.md`

### P1

- `Strictly validate summary date params and return 400 for invalid ranges`
  - Why now: analytics endpoints still parse arbitrary strings and can compute incorrect windows.
  - Evidence: `summary.ts` accepts optional strings and uses `parse(...)` without rejecting invalid dates in [app/api/[[...route]]/summary.ts](/home/gonzalo/projects/nuchi/app/api/[[...route]]/summary.ts)
  - Source: `PR14/TECH_DEBT_TICKETS.md`

- `Remove debug summary endpoint`
  - Why now: undocumented production surface area is still exposed.
  - Evidence: `.get('/asd', ...)` remains in [app/api/[[...route]]/summary.ts](/home/gonzalo/projects/nuchi/app/api/[[...route]]/summary.ts)
  - Source: `PR14/TECH_DEBT_TICKETS.md`

- `Fix import create-mode CTA regression`
  - Why now: this is a user-facing correctness bug with trivial remediation.
  - Evidence: create mode still renders `Edit Transaction` in [features/transactions/components/transaction-form.tsx](/home/gonzalo/projects/nuchi/features/transactions/components/transaction-form.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Replace remaining unsafe any contracts in import and chart tooltip code`
  - Why now: weak typing hides integration bugs in the areas currently changing the most.
  - Evidence: `any` remains in [app/(dashboard)/transactions/upload-button.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/upload-button.tsx), [app/(dashboard)/transactions/import-card.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/import-card.tsx), and [components/category-tooltip.tsx](/home/gonzalo/projects/nuchi/components/category-tooltip.tsx)
  - Source: `PR14/TECH_DEBT_TICKETS.md`, `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Decouple account filter disabled state from summary loading`
  - Why now: header controls trigger unnecessary coupling and can block interaction on non-summary pages.
  - Evidence: `AccountFilter` still depends on `useGetSummary().isLoading` in [components/account-filter.tsx](/home/gonzalo/projects/nuchi/components/account-filter.tsx)
  - Source: `PR-15/TECH_DEBT_TICKETS.md`

- `Replace hardcoded All time lower bound`
  - Why now: historical data before 2000 is silently excluded.
  - Evidence: `allTime` still uses `new Date(2000, 0, 1)` in [components/date-filter.tsx](/home/gonzalo/projects/nuchi/components/date-filter.tsx)
  - Source: `PR-15/TECH_DEBT_TICKETS.md`

- `Trim and validate account/category names`
  - Why now: duplicate-looking values and oversized names are still allowed.
  - Evidence: forms and insert schemas still use raw `name` values without trim/min/max guards in [features/accounts/components/account-form.tsx](/home/gonzalo/projects/nuchi/features/accounts/components/account-form.tsx), [features/categories/components/category-form.tsx](/home/gonzalo/projects/nuchi/features/categories/components/category-form.tsx), and [db/schema.ts](/home/gonzalo/projects/nuchi/db/schema.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`, `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR06.md`

- `Start automated coverage for critical money paths`
  - Why now: the repo still has no tests, so every fix above ships with regression risk.
  - Evidence: no test files are present in the repository
  - Source: all scanned debt sets

### P2

- `Chunk large CSV imports and add size guardrails`
  - Why now: current imports submit one unbounded payload and can degrade badly on large files.
  - Evidence: `onSubmitImport` passes the full array straight into bulk-create in [app/(dashboard)/transactions/page.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/page.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Add user guidance to the import mapping flow`
  - Why now: the import UX is functional but under-explained, which increases support load.
  - Evidence: `ImportCard` renders controls without instructional copy in [app/(dashboard)/transactions/import-card.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/import-card.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Normalize import view layout with list view`
  - Why now: inconsistent layout is low-risk but visible polish debt.
  - Evidence: import mode returns `ImportCard` without the shared page container in [app/(dashboard)/transactions/page.tsx](/home/gonzalo/projects/nuchi/app/(dashboard)/transactions/page.tsx)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR13.md`

- `Centralize API error handling and remove ad hoc server logging`
  - Why now: route error handling is still inconsistent and one raw `console.log` remains.
  - Evidence: ad hoc `catch` blocks dominate route files and `console.log(e)` remains in [app/api/[[...route]]/transactions.ts](/home/gonzalo/projects/nuchi/app/api/[[...route]]/transactions.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`

- `Add audit trail for destructive financial changes`
  - Why now: support and incident response remain blind after deletes or edits.
  - Evidence: no audit table or event writing exists in current schema/routes
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`

- `Introduce soft deletes / restore flows for accounts and categories`
  - Why now: accidental deletes remain irreversible.
  - Evidence: schema has no `deleted_at` or restore path in [db/schema.ts](/home/gonzalo/projects/nuchi/db/schema.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`, `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR06.md`

- `Add pagination to accounts listing`
  - Why now: not urgent for current scale, but still open product debt.
  - Evidence: accounts list endpoint returns all rows without cursor/limit in [app/api/[[...route]]/accounts.ts](/home/gonzalo/projects/nuchi/app/api/[[...route]]/accounts.ts)
  - Source: `PR-Reviews/Tech_debt/TECH_DEBT_TICKETS_PR05.md`

## Roadmap

### Phase 1: Security and Data Integrity

- Add ownership validation to transaction create and bulk-create.
- Add strict CSV schema validation, empty-file guards, and date parsing errors.
- Add CSRF protection for mutating routes.
- Add rate limiting for accounts, categories, and transaction write endpoints.
- Remove `/api/summary/asd` and hard-fail invalid summary dates.

### Phase 2: User-Facing Correctness and Developer Safety

- Fix transaction create-mode label.
- Replace remaining `any` contracts in import flow and category tooltip.
- Decouple account filter from summary loading.
- Replace the hardcoded `All time` lower bound with backend-derived history start.
- Add trim/min/max validation for account and category names.
- Remove remaining raw server console logging and standardize API error helpers.

### Phase 3: Regression Coverage

- Add API tests first for transactions create/bulk-create ownership, summary invalid dates, and import validation.
- Add hook/component tests for filter behavior, import mapping, and tooltip contracts.
- Add a minimal end-to-end smoke path for accounts, categories, transactions, and CSV import.

### Phase 4: Operational Resilience and UX Improvements

- Add audit events for create/edit/delete/bulk-delete flows.
- Introduce soft deletes plus restore for accounts and categories.
- Add chunked import submission, file-size caps, and partial-failure reporting.
- Add import guidance copy and align import layout with the normal transactions view.
- Add pagination to accounts once real dataset size justifies it.
