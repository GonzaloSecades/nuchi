# Graph Report - nuchi  (2026-11-18)

## Corpus Check
- 364 files · ~230,606 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 11 file(s) not represented in the graph (top: (none) 6, .example 1, .ico 1)

## Summary
- 3378 nodes · 8185 edges · 315 communities (160 shown, 155 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 577 edges (avg confidence: 0.88)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f177ade4`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- testing.T
- dependencies
- Current API Parity Fixtures
- cn
- transactions/page.tsx
- github.com/jackc/pgx/v5/pgxpool.Pool
- PR Overview: CSV Transaction Import Flow with Account Assignment
- unwrap
- session.tsx
- edit-transaction-sheet.tsx
- oapi-codegen Configuration
- PR Overview: <Descriptive Title Based on PR>
- 2. Prioritized workstreams
- time.Time
- Transactions API — Technical Reference
- scripts
- ProjectPolish Context
- generated.gen.go
- github.com/jackc/pgx/v5/pgtype.UUID
- Nuchi Roadmap Features List
- pgUserID
- DataTable
- post-migration-improvements/README.md
- Button
- Nuchi Project Context
- Nuchi Go Backend
- Next.js App Router
- newAuthTestEnv
- compilerOptions
- components.json
- What You Must Do When Invoked
- codex-backend-improvements/README.md
- Nuchi
- devDependencies
- utils.ts
- API Parity Source Files
- Go Backend Replacement Spec
- config_test.go
- Nuchi Current Product Design System
- transaction-payload.ts
- newTransactionsTestEnv
- transaction-date.ts
- Nuchi Codex Guide
- Nuchi Domain Model
- auth_live_test.go
- Findings — Argentina Open Banking / Sistema de Finanzas Abiertas
- Roadmap
- context.Context
- mustParseURL
- client.ts
- Current UI map
- Queries
- Queries
- Summary — Product Behavior
- UnauthorizedErrorJSONResponse
- Current implemented tokens
- Authentication API — Technical Reference
- Queries
- nuchi-postgres-data Volume
- use-bulk-delete-transactions.ts
- Transactions — Product Behavior
- Bun Package Manager
- dev-local.ts
- Clerk Auth
- components/ui Primitives
- 4. Positive foundations to preserve
- OpenAPI
- Summary API — Technical Reference
- Drizzle ORM
- Auth
- Hono API
- Neon/Postgres
- Globe Icon
- Vercel Deployment Platform
- Next.js Framework
- Window Icon
- Nuchi Brand Identity
- TestMigration00004_NullsPreexistingCrossOwnerCategory_LiveDatabase
- File Icon
- createAuthenticatedFetch
- go-api-rewrite.ts
- postcss.config.mjs
- CSV Import — Product Behavior
- Copilot PR Review Prompt Agent Instructions
- Delivery Roadmap and Quality Gates
- TestFinanceRLS_LiveDatabase
- CSV Import Feature (Docs)
- Authentication — Product Behavior
- useBulkCreateTransactions Hook
- PR13: CSV Transaction Import Flow
- ImportCard Component
- ImportTable Component
- react-papaparse CSV Parser Dependency
- TableHeadSelect Component
- UploadButton Component
- useSelectAccount Hook
- Budgets and Monthly Planning (Roadmap Feature 4)
- Import Review, Normalization, and Deduplication (Roadmap Feature 2)
- Open Banking / Direct Bank Sync (Future)
- External Parser Service (PDF/CSV to Transactions)
- Recurring Transactions and Forecasting (Roadmap Feature 5)
- Transfers, Balances, and Net Worth (Roadmap Feature 3)
- Unified Import Inbox (Roadmap Feature 1)
- Tech Debt Ticket Generation Agent Instructions
- Tech Debt Remediation Roadmap
- TanStack Query
- Transaction Milliunits
- Typed Hono Client
- Backend Configuration
- Health Endpoint
- Nuchi Go Backend README
- Target Backend Architecture
- package.json
- Architecture
- Troubleshooting
- Local Setup
- Mailpit Docker Service
- Nuchi Project Context (Canonical Docs)
- Accounts API
- Bulk Delete Missing IDs Behavior
- Categories API
- GitHub Issue #34 API Parity Fixtures
- Shared API Behavior
- Summary API
- Summary Calculations
- Transaction Bulk Create
- Transaction Date Filters
- Transactions API
- API Parity Requirements
- ARS Default Currency
- Dockerized PostgreSQL
- Generated OpenAPI Clients
- Separate Go API
- Go Outside Docker Development
- Go Persistence Stack
- GitHub Issue #18 Go Backend Replacement
- GitHub Issue #21 Schema Migrations
- GitHub Issue #22 Owned Auth
- GitHub Issue #28 Spec Document
- GitHub Issue #29 OpenAPI Contract
- JWT Access Tokens
- Legacy Backend Cleanup
- Milliunit Money Invariant
- Owned Email/Password Auth
- Refresh Token Cookies
- Same-Origin /api Proxy
- next
- Improvement Proposal — `<title>`
- graphify reference: extra exports and benchmark
- AmountInput
- Color-system drift
- Observability Readiness
- Module Improvement Map
- Codex — Atomic Multi-Chunk Transaction Imports
- Endpoint Review — `<operationId>`
- Current UI inconsistencies
- State and behavior inconsistencies
- Accounts API — Technical Reference
- Accounts — Product Behavior
- .ListCategories
- Current State and Constraints
- Security Standard
- Performance and Query Engineering
- ProjectPolish PR Summary
- .QueryRow
- PR Review Cycle
- Spacing, size, radius, and elevation drift
- API Contract and Documentation Standard
- useOpenAccount
- graphify reference: query, path, explain
- Responsive inconsistencies
- Nuchi Code Review
- schema.d.ts
- 0001 — transactions.date is timestamp without time zone
- react
- 0003 — Transaction rate limiting is in-memory
- 0007 — JWT signing is HS256 with a single static secret
- 0008 — Access tokens cannot be revoked mid-life
- 0009 — No refresh-token reuse detection or session listing
- 0010 — Auth operations do not declare 500 responses in the contract
- 0011 — No resend endpoint; email delivery is fire-and-forget
- 0013 — Resource endpoints accept unbounded request bodies
- 0014 — Date filters are parsed in UTC, not the host timezone
- 0015 — Out-of-range amounts return 400, where legacy returned 500
- 0017 — The summary category chart silently excludes uncategorized spending
- 0018 — A transaction's date is serialized as a UTC instant, not a calendar date
- .BulkCreateTransactions
- .BulkDeleteAccounts
- .BulkDeleteCategories
- .BulkDeleteTransactions
- .ConfirmPasswordReset
- .CreateAccount
- .CreateCategory
- .CreateTransaction
- Accessibility inconsistencies
- Duplicated and parallel patterns
- Q: got ticket #41 done by claude, please review the pushed changes and add comments beside the copilot ones, as you see fit
- Q: recheck the fixes and give another pass for reviews
- Q: give it a last review of all the files changend in the pr the comments and all that is in place right now in the pr leave comments for medim-high and high issues mid-low and low forget it we mande 3 iterations so far. if something is too big then addres it consitently here so it so i can anlyze it
- Q: Review PR #64 against post-merge PR #62 comment and issue #63
- Q: Review briefing 42 email verification and password reset design and confirm best-effort send, silent rate cap, TTL defaults, and plain-text future-frontend links
- Q: Recheck latest PR 65 changes and comments for merge readiness
- Q: check the graphify outcome for nuchi and tell me how can we use it properly
- Q: would it be a good complement or suplement adding colbymchenry/codegraph?
- Q: recheck the changes and doanother review
- Q: check the tickets going foward, what we shall continue with after this pr merges ?
- Q: check the changes introduced for your reviews and recheck the pr if is there more to approach
- Q: last comment address if you have anything more to say label it for a future improvement post migration and name it Codex-{title}
- Q: 3 more prs landed. Review them all
- Q: address the mas comments accordingly in the prs
- Q: what else is left to address on all 5 prs ?
- ValidationErrorJSONResponse
- Q: orchestrate everything to have a clean and smooth merge to master make nice merge comments after that check if everything is ok and give me a final merge summary
- .LoginUser
- .RegisterUser
- .RequestPasswordReset
- .UpdateAccount
- .UpdateCategory
- .UpdateTransaction
- .VerifyEmail
- 0004 — Bulk-delete silently ignores missing/unowned IDs
- Codex Backend Improvements
- tech_debt_instructions.md
- graphify reference: add a URL and watch a folder
- graphify reference: commit hook and native CLAUDE.md integration
- graphify reference: incremental update and cluster-only
- ChartEnum
- SpendingPieEnum
- Typography drift
- Q: Review PR 65 email verification and password reset changes
- Q: Should Nuchi create shared Claude and Codex skills around Graphify and CodeGraph after the Go migration?
- .DeleteAccount
- internal/openapi/README.md
- generated/README.md
- .DeleteCategory
- .DeleteTransaction
- .GetAccount
- .GetCategory
- .GetHealth
- .GetTransaction
- .ListAccounts
- .LogoutUser
- graphify reference: GitHub clone and cross-repo merge
- graphify reference: transcribe video and audio
- extraction-spec.md
- github.com/GonzaloSecades/nuchi/backend
- AuthServer
- main
- ServerInterfaceWrapper
- Nuchi — Claude Code Guide
- net/http.ResponseWriter
- date-filter.tsx
- Post-Migration Improvements
- transaction-route-utils.ts
- Copilot Instructions — nuchi
- NewRouter
- Generated OpenAPI Client Types
- Hand-Edited OpenAPI Contract Source
- GitHub Issue #35 Generator Tooling Deferred
- nuchi.openapi.json
- OpenAPI Directory README
- TypeScript Fetch Client Generation
- Clerk Authentication
- CSV Transaction Import Feature
- Dashboard Analytics
- Drizzle ORM with Neon Postgres
- Hono API Layer
- Local Postgres
- Mailpit Local SMTP
- Next.js 16 App Router Stack
- Nuchi Application
- Nuchi Personal Finance App
- Nuchi README
- TanStack Query Server State
- Local Docker Postgres Dev Setup
- CSV Import Chunked Bulk-Create Submission
- Tech Debt Backlog (Consolidated)
- github.com/jackc/pgx/v5/pgtype.Text
- app/layout.tsx
- InvalidRefreshTokenErrorJSONResponse
- import-card.tsx

## God Nodes (most connected - your core abstractions)
1. `cn()` - 96 edges
2. `Button()` - 63 edges
3. `unwrap()` - 50 edges
4. `newTransactionsTestEnv()` - 45 edges
5. `newAuthTestEnv()` - 42 edges
6. `lucide-react` - 41 edges
7. `react` - 40 edges
8. `ServerInterface` - 36 edges
9. `createTestAccount()` - 34 edges
10. `ServerInterfaceWrapper` - 33 edges

## Surprising Connections (you probably didn't know these)
- `UX-018 — Color is not consistently reinforced as a state signal` --references--> `AmountInput()`  [INFERRED]
  design/audit/DESIGN-AUDIT.md → components/amount-input.tsx
- `Disabled amount input is only partially disabled` --references--> `AmountInput()`  [INFERRED]
  design/current/INCONSISTENCIES.md → components/amount-input.tsx
- `Borders and focus treatment` --references--> `AmountInput()`  [INFERRED]
  design/current/TOKENS.md → components/amount-input.tsx
- `12. Feedback patterns` --references--> `AuthError()`  [INFERRED]
  design/current/DESIGN.md → components/auth/auth-error.tsx
- `Several state surfaces represent errors differently` --references--> `AuthError()`  [INFERRED]
  design/current/INCONSISTENCIES.md → components/auth/auth-error.tsx

## Import Cycles
- None detected.

## Communities (315 total, 155 thin omitted)

### Community 0 - "testing.T"
Cohesion: 0.09
Nodes (54): IssueAccessToken(), TestIssueAccessToken_ClaimsAreMinimal(), TestIssueAndVerifyAccessToken_RoundTrip(), TestVerifyAccessToken_ExpiredRejected(), TestVerifyAccessToken_GarbageTokenRejected(), TestVerifyAccessToken_MissingExpRejected(), TestVerifyAccessToken_NonExpiryFailuresStayGeneric(), TestVerifyAccessToken_NonUUIDSubjectRejected() (+46 more)

### Community 1 - "dependencies"
Cohesion: 0.06
Nodes (36): dependencies, class-variance-authority, clsx, date-fns, @hookform/resolvers, lucide-react, next, next-themes (+28 more)

### Community 2 - "Current API Parity Fixtures"
Cohesion: 0.06
Nodes (30): Accounts, Auth Redirects And API Auth Errors, Categories, CSV Import To Bulk Create, Current API Parity Fixtures, `DELETE /api/accounts/:id`, `DELETE /api/categories/:id`, `DELETE /api/transactions/:id` (+22 more)

### Community 3 - "cn"
Cohesion: 0.09
Nodes (38): Actions(), Props, columns, ResponseType, Actions(), Props, columns, ResponseType (+30 more)

### Community 4 - "transactions/page.tsx"
Cohesion: 0.07
Nodes (67): AccountsPage(), CategoriesPage(), DashboardPage(), SettingsPage(), ImportCard(), TransactionsPage(), TransactionsPageWrapper(), VARIANTS (+59 more)

### Community 5 - "github.com/jackc/pgx/v5/pgxpool.Pool"
Cohesion: 0.09
Nodes (43): assertCascadingUserFK(), assertUniqueColumn(), TestAuthBaseSchema_LiveDatabase(), NewPool(), TestNewPool_InvalidURL(), TestNewPool_LiveDatabase(), TestNewPool_UnreachableServer(), TestFinanceRLSConfiguration_LiveDatabase() (+35 more)

### Community 6 - "PR Overview: CSV Transaction Import Flow with Account Assignment"
Cohesion: 0.06
Nodes (31): 1. Frontend - Data/State (Next.js, React hooks, mutation hooks), 2. Frontend - UI Components (Shadcn/Radix UI, table rendering), 3. State Management (Custom hooks, promise-based dialog flow), 4. Developer Experience & Configuration (Dependency and repo hygiene), API, Automated Testing Needs, Breaking Changes, Business Value (+23 more)

### Community 7 - "unwrap"
Cohesion: 0.14
Nodes (25): RequestType, ResponseType, RequestType, ResponseType, ResponseType, useDeleteAccount(), RequestType, ResponseType (+17 more)

### Community 8 - "session.tsx"
Cohesion: 0.09
Nodes (26): Dependencies, FetchLike, accessTokenStore, clearAccessToken, createAccessTokenStore(), getAccessToken, setAccessToken, subscribeToAccessToken (+18 more)

### Community 9 - "edit-transaction-sheet.tsx"
Cohesion: 0.17
Nodes (28): Sheet(), SheetContent(), SheetDescription(), SheetFooter(), SheetHeader(), SheetOverlay(), SheetPortal(), SheetTitle() (+20 more)

### Community 10 - "oapi-codegen Configuration"
Cohesion: 0.40
Nodes (5): chi-server Generation, Generated Go Output, Model Generation, oapi-codegen Configuration, Go Server Type Generation

### Community 11 - "PR Overview: <Descriptive Title Based on PR>"
Cohesion: 0.07
Nodes (26): Automated Testing Needs, Breaking Changes, Business Value, Conclusion, Dependencies & Prerequisites, Deployment Considerations, External Service Dependencies, High-Impact Risks (+18 more)

### Community 12 - "2. Prioritized workstreams"
Cohesion: 0.06
Nodes (34): 1. How to read this map, 2. Prioritized workstreams, 3. What should be preserved, 4. Phased sequence, 5. Program-level guardrails, Nuchi Redesign Opportunity Map, Opportunity A1 — Distinguish failed reads from valid empty and zero data, Opportunity A2 — Create a coherent loading, empty, and outcome language (+26 more)

### Community 13 - "time.Time"
Cohesion: 0.10
Nodes (33): calendarDaysBetween(), endOfDay(), parseCalendarDate(), parseDateRange(), present(), TestParseDateRange_BoundariesAreWholeDays(), TestParseDateRange_Defaults(), TestParseDateRange_InvalidInputs() (+25 more)

### Community 14 - "Transactions API — Technical Reference"
Cohesion: 0.06
Nodes (33): All-or-nothing is structural, Amounts, Atomicity is per request, not per import, Bulk operations, Byte limits and the 413, `categoryId` states, Currency, Date filtering (+25 more)

### Community 15 - "scripts"
Cohesion: 0.14
Nodes (14): scripts, build, db:reset, dev, dev:next, format, format:check, lint (+6 more)

### Community 16 - "ProjectPolish Context"
Cohesion: 0.20
Nodes (9): Follow-Up Memory, Implemented Scope, Last Verified, Local Database, Memory Index, ProjectPolish Context, Restart Point, Safety Rules (+1 more)

### Community 17 - "generated.gen.go"
Cohesion: 0.01
Nodes (107): AccountNotFoundErrorJSONResponse, AccountSummary, ApiErrorResponse, BulkCreateValidationErrorJSONResponse, BulkDeleteValidationErrorJSONResponse, Category, CategoryNotFoundErrorJSONResponse, CategorySummary (+99 more)

### Community 18 - "github.com/jackc/pgx/v5/pgtype.UUID"
Cohesion: 0.13
Nodes (12): Queries, ConsumeRefreshTokenRow, CountRecentEmailVerificationTokensParams, CountRecentPasswordResetTokensParams, CreateEmailVerificationTokenParams, CreatePasswordResetTokenParams, CreateRefreshTokenParams, CreateUserParams (+4 more)

### Community 19 - "Nuchi Roadmap Features List"
Cohesion: 0.07
Nodes (26): 1. Unified Import Inbox, 2. Import Review, Normalization, and Deduplication, 3. Transfers, Balances, and Net Worth, 4. Budgets and Monthly Planning, 5. Recurring Transactions and Forecasting, Current roadmap principle, Nuchi Roadmap Features List, Open banking readiness (+18 more)

### Community 20 - "pgUserID"
Cohesion: 0.06
Nodes (62): accountNotFoundError(), constraintName(), duplicateAccountNameError(), ResourceServer, isUniqueViolation(), toAccount(), toAccountSummary(), validateAccountName() (+54 more)

### Community 21 - "DataTable"
Cohesion: 0.13
Nodes (27): ImportTable(), Props, DataTable(), DataTableProps, Table(), TableBody(), TableCaption(), TableCell() (+19 more)

### Community 22 - "post-migration-improvements/README.md"
Cohesion: 0.08
Nodes (21): Case-insensitive names, Categories API — Technical Reference, Deleting a category keeps its transactions, Non-negotiables when changing this code, Ownership model, Where the code lives, 0002 — Finance tables use text cuid IDs; UUID default is v4, How it was migrated (+13 more)

### Community 23 - "Button"
Cohesion: 0.10
Nodes (60): ForgotPasswordPage(), formSchema, FormValues, formSchema, FormValues, ResetPassword(), ResetPasswordPage(), formSchema (+52 more)

### Community 24 - "Nuchi Project Context"
Cohesion: 0.08
Nodes (23): Accounts, Accounts, API Shape, Canonical Architecture, Categories, Categories, Client-to-Database Flow, Core Stack (+15 more)

### Community 25 - "Nuchi Go Backend"
Cohesion: 0.22
Nodes (9): Configuration, Database, Health Check, Local Run, Migrations (goose), Nuchi Go Backend, RLS, Typed queries (sqlc) (+1 more)

### Community 28 - "newAuthTestEnv"
Cohesion: 0.11
Nodes (42): decodeAPIError(), newAuthTestEnv(), refreshCookieFromResponse(), TestAuthLive_Login_UnknownEmailUnauthorized(), TestAuthLive_Login_UnknownFieldRejected(), TestAuthLive_Login_UnverifiedUserForbidden(), TestAuthLive_Login_VerifiedUserSucceeds(), TestAuthLive_Login_WrongPasswordAndUnknownEmailShareShape() (+34 more)

### Community 29 - "compilerOptions"
Cohesion: 0.11
Nodes (18): compilerOptions, allowJs, esModuleInterop, incremental, isolatedModules, jsx, lib, module (+10 more)

### Community 30 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 31 - "What You Must Do When Invoked"
Cohesion: 0.08
Nodes (24): For /graphify add and --watch, For /graphify query, For the commit hook and native CLAUDE.md integration, For --update and --cluster-only, /graphify, Honesty Rules, Interpreter guard for subcommands, Part A - Structural extraction for code files (+16 more)

### Community 32 - "codex-backend-improvements/README.md"
Cohesion: 0.11
Nodes (15): Categories are private to one user, Categories — Product Behavior, Deleting a category keeps the transactions, Messages a user can encounter, Names are unique without regard to case, Related, What a category is, What this deliberately cannot do yet (+7 more)

### Community 33 - "Nuchi"
Cohesion: 0.12
Nodes (16): Backend Commands, Canonical Docs, Contributing And Workflow, Database, Docker And Local Services, Frontend Commands, Go API proxy configuration, Graphify (+8 more)

### Community 34 - "devDependencies"
Cohesion: 0.12
Nodes (16): devDependencies, eslint, eslint-config-next, eslint-config-prettier, openapi-typescript, postcss, prettier, prettier-plugin-tailwindcss (+8 more)

### Community 35 - "utils.ts"
Cohesion: 0.10
Nodes (25): Props, BarVariant(), Props, CategoryTooltip(), CategoryTooltipPayload, CategoryTooltipProps, CustomTooltip(), CustomTooltipProps (+17 more)

### Community 37 - "Go Backend Replacement Spec"
Cohesion: 0.11
Nodes (17): API Contract, Auth And Sessions, Child Issue Plan, Currency And Money, Database And RLS, Go Backend Replacement Spec, Goals, Implementation Order (+9 more)

### Community 38 - "config_test.go"
Cohesion: 0.22
Nodes (22): getEnv(), getEnvBool(), Load(), clearAuthEnv(), TestLoad_BareMailFromAccepted(), TestLoad_Defaults(), TestLoad_InvalidAccessTokenTTLFailsFast(), TestLoad_InvalidCookieSecureFailsFast() (+14 more)

### Community 39 - "Nuchi Current Product Design System"
Cohesion: 0.09
Nodes (22): 10. Component states, 11. Form behavior, 12. Feedback patterns, 13. Navigation patterns, 14. Responsive behavior, 15. Accessibility behavior, 16. Motion and transition patterns, 1. Design principles currently evident (+14 more)

### Community 40 - "transaction-payload.ts"
Cohesion: 0.17
Nodes (14): RequestType, ResponseType, DEFAULT_CURRENCY, emptyToNull(), ImportedTransactionInput, baseValues, toBulkTransactionInput(), toTransactionInput() (+6 more)

### Community 41 - "newTransactionsTestEnv"
Cohesion: 0.16
Nodes (44): createTestAccount(), createTestCategory(), getSummary(), seedSummaryTransaction(), TestSummaryLive_AccountFilter(), TestSummaryLive_CategoryBreakdown(), TestSummaryLive_ChangeComparesThePrecedingWindow(), TestSummaryLive_ChangeIs100WhenPreviousPeriodIsEmpty() (+36 more)

### Community 42 - "transaction-date.ts"
Cohesion: 0.28
Nodes (10): CalendarDate, calendarDateFromApi(), calendarDateFromLocalDate(), localDateFromApi(), localDateFromCalendarDate(), pad(), picker, useGetTransactions() (+2 more)

### Community 43 - "Nuchi Codex Guide"
Cohesion: 0.20
Nodes (10): Commands, Env, graphify, Nuchi Codex Guide, Pull Requests Hard Rules, Reference, Repo Rules, Risk Areas (resolved — kept as a record) (+2 more)

### Community 45 - "auth_live_test.go"
Cohesion: 0.07
Nodes (47): argonParams, decodeHash(), DummyVerify(), HashPassword(), mustHashPassword(), TestDecodeHash_OperationalBounds(), TestDummyVerify_DoesNotPanicOrLeak(), TestHashPassword_EncodesPHCString() (+39 more)

### Community 46 - "Findings — Argentina Open Banking / Sistema de Finanzas Abiertas"
Cohesion: 0.10
Nodes (18): Briefing — Argentina Open Banking Investigation, Refresh protocol, Research questions, What to check on each refresh, Why Nuchi cares, 1. Background: no framework before 2025, 2. The promise delivered: Decree 353/2025 creates the SFA, 3. Late 2025: BCRA turns it into an active program (+10 more)

### Community 47 - "Roadmap"
Cohesion: 0.17
Nodes (12): Open Tickets, P0, P1, P2, Phase 1: Security and Data Integrity, Phase 2: User-Facing Correctness and Developer Safety, Phase 3: Regression Coverage, Phase 4: Operational Resilience and UX Improvements (+4 more)

### Community 48 - "context.Context"
Cohesion: 0.17
Nodes (3): Queries, Mailer, User

### Community 49 - "mustParseURL"
Cohesion: 0.18
Nodes (13): buildLink(), passwordResetEmailBody(), mustParseURL(), startFakeSMTPServer(), TestBuildLink_EscapesTokenQueryParam(), TestBuildLink_PreservesBaseURLHostAndScheme(), TestSMTPMailer_ConnectionDeadlineEnforced(), TestSMTPMailer_SendPasswordResetEmail() (+5 more)

### Community 50 - "client.ts"
Cohesion: 0.09
Nodes (20): validationError(), API_BASE_URL, apiErrorCode(), ApiErrorEnvelope, apiOrigin, toApiError(), ApiError, ApiErrorDetails (+12 more)

### Community 51 - "Current UI map"
Cohesion: 0.11
Nodes (18): Application shells, Authentication shell, Current UI map, Dashboard navigation, Dashboard shell, Empty, Error, Filters (+10 more)

### Community 52 - "Queries"
Cohesion: 0.24
Nodes (4): Queries, BulkDeleteTransactionsParams, DeleteTransactionParams, GetTransactionParams

### Community 53 - "Queries"
Cohesion: 0.16
Nodes (7): Queries, BulkDeleteAccountsParams, CreateAccountParams, DeleteAccountParams, GetAccountParams, ListOwnedAccountIDsParams, UpdateAccountNameParams

### Community 54 - "Summary — Product Behavior"
Cohesion: 0.11
Nodes (16): "Compared to previous period" means the period immediately before, Filtering by one account, "Other", Related, Summary — Product Behavior, The category chart does not add up to the expenses total, The daily chart always covers every day, The period (+8 more)

### Community 55 - "UnauthorizedErrorJSONResponse"
Cohesion: 0.05
Nodes (22): UnauthorizedErrorJSONResponse, BulkCreateTransactions401JSONResponse, BulkDeleteAccounts401JSONResponse, BulkDeleteCategories401JSONResponse, BulkDeleteTransactions401JSONResponse, CreateAccount401JSONResponse, CreateCategory401JSONResponse, CreateTransaction401JSONResponse (+14 more)

### Community 56 - "Current implemented tokens"
Cohesion: 0.11
Nodes (17): Borders and focus treatment, Breakpoints and responsive values, Color tokens, Current implemented tokens, Direct Tailwind palette values actually used, Families, Hard-coded color values outside the token aliases, Icons and motion (+9 more)

### Community 57 - "Authentication API — Technical Reference"
Cohesion: 0.11
Nodes (17): Access token: short-lived and memory-only, Authentication API — Technical Reference, Divergences and gaps, Email delivery boundary, Email verification, Enumeration safety, Logout, Non-negotiables when changing this code (+9 more)

### Community 58 - "Queries"
Cohesion: 0.16
Nodes (7): Queries, BulkDeleteCategoriesParams, CreateCategoryParams, DeleteCategoryParams, GetCategoryParams, ListOwnedCategoryIDsParams, UpdateCategoryNameParams

### Community 60 - "use-bulk-delete-transactions.ts"
Cohesion: 0.18
Nodes (14): API-error row numbers, Chunking and failure boundaries, CSV Import — Technical Reference, Divergences and gaps, End-to-end pipeline, Import invariants, Indexed validation errors, Malformed bodies have no `details` (+6 more)

### Community 61 - "Transactions — Product Behavior"
Cohesion: 0.12
Nodes (17): Categories are optional, and must be yours, Currency, Default view, Editing replaces the whole transaction, Filtering by an account that isn't yours, Filters, Importing many at once, Messages a user can encounter (+9 more)

### Community 63 - "dev-local.ts"
Cohesion: 0.07
Nodes (32): HEADER_FILTER_FILES, PROBE, ProbeResult, probeZone(), resolved, runIn(), Zone, zones (+24 more)

### Community 66 - "4. Positive foundations to preserve"
Cohesion: 0.14
Nodes (13): 1. Evidence boundary and limitations, 2. Executive audit summary, 3. Severity overview, 4. Positive foundations to preserve, 6. Evaluation-dimension coverage, 7. Overall assessment, Nuchi Current Application UX/UI Design Audit, PF-01 — Clear product-level shell and route structure (+5 more)

### Community 69 - "OpenAPI"
Cohesion: 0.29
Nodes (6): Contract Coverage, Generate, Layout, OpenAPI, Shared Contract, Validate

### Community 70 - "Summary API — Technical Reference"
Cohesion: 0.14
Nodes (14): Category breakdown, Daily series, Date handling is shared, not reimplemented, Divergences and gaps, Empty state, Four queries, one transaction, Percentage change, Summary API — Technical Reference (+6 more)

### Community 72 - "Auth"
Cohesion: 0.25
Nodes (8): Auth, curl examples, Email verification and password reset, JWT access tokens, Mailpit (local email testing), Password hashing, Refresh tokens and the session cookie, Registration and login

### Community 75 - "Globe Icon"
Cohesion: 0.67
Nodes (4): Next.js Application, Public Directory, Globe Icon, Web / Global UI Theme

### Community 80 - "TestMigration00004_NullsPreexistingCrossOwnerCategory_LiveDatabase"
Cohesion: 0.31
Nodes (12): adminDatabaseURL(), connect(), execUpgrade(), gooseBinary(), insertUpgradeUser(), pgIdent(), runGoose(), setSessionAppUser() (+4 more)

### Community 81 - "File Icon"
Cohesion: 0.67
Nodes (3): Document/File Metaphor, File Icon, UI Icon

### Community 82 - "createAuthenticatedFetch"
Cohesion: 0.22
Nodes (10): createAuthenticatedFetch(), isAuthRequest(), refresh(), refreshOnce(), requestPath(), readErrorCode(), expired(), jsonResponse() (+2 more)

### Community 83 - "go-api-rewrite.ts"
Cohesion: 0.43
Nodes (5): DEFAULT_GO_API_URL, goApiRewrites(), normalizeGoApiUrl(), RewriteEnv, nextConfig

### Community 86 - "CSV Import — Product Behavior"
Cohesion: 0.18
Nodes (11): CSV Import — Product Behavior, Current limitations, Errors found before upload, Errors returned while saving, How displayed row numbers work, Large files are saved in groups of 500, Messages a user can encounter, Related (+3 more)

### Community 88 - "Delivery Roadmap and Quality Gates"
Cohesion: 0.18
Nodes (10): Delivery Roadmap and Quality Gates, Evidence storage convention, Phase 0 — Rebaseline after migration, Phase 1 — Security and correctness foundation, Phase 2 — Transaction and resilience hardening, Phase 3 — Query and runtime performance, Phase 4 — Documentation and observability readiness, Phase 5 — Behavior and schema evolution (+2 more)

### Community 89 - "TestFinanceRLS_LiveDatabase"
Cohesion: 0.42
Nodes (9): assertRLSViolation(), assertRowCount(), insertTestAccount(), insertTestCategory(), insertTestTransaction(), insertTestUser(), setAppUser(), TestFinanceRLS_LiveDatabase() (+1 more)

### Community 91 - "Authentication — Product Behavior"
Cohesion: 0.20
Nodes (9): Authentication — Product Behavior, Local testing, Resetting a password signs you out everywhere, Signing out, Signing up does not sign you in, Staying signed in, The reset form always says the same thing, The screens (+1 more)

### Community 121 - "Target Backend Architecture"
Cohesion: 0.20
Nodes (9): Architecture acceptance tests, Configuration, Core rule, Endpoint policy registry, Error boundary, Package boundaries, Request flow, Target Backend Architecture (+1 more)

### Community 122 - "package.json"
Cohesion: 0.07
Nodes (29): eslintConfig, ignoreScripts, name, private, trustedDependencies, version, eslint, eslint-config-next (+21 more)

### Community 124 - "Architecture"
Cohesion: 0.50
Nodes (4): Architecture, Current App Shape, Legacy And Reference Code, Target Go Backend Shape

### Community 125 - "Troubleshooting"
Cohesion: 0.50
Nodes (4): Docker Engine Not Reachable, GitHub Project Board Requires Scope, Graphify Warnings, Troubleshooting

### Community 126 - "Local Setup"
Cohesion: 0.50
Nodes (4): Environment, Install Dependencies, Local Setup, Prerequisites

### Community 161 - "next"
Cohesion: 0.09
Nodes (27): DashboardLayout(), Props, SessionGuard(), Header(), HeaderLogo(), NavButton(), Props, Navigation() (+19 more)

### Community 162 - "Improvement Proposal — `<title>`"
Cohesion: 0.20
Nodes (9): Alternatives and tradeoffs, Contract and compatibility, Decision record, Improvement Proposal — `<title>`, Invariants, Problem and evidence, Proposed design, Risks (+1 more)

### Community 163 - "graphify reference: extra exports and benchmark"
Cohesion: 0.22
Nodes (8): graphify reference: extra exports and benchmark, Step 6b - Wiki (only if --wiki flag), Step 7 - Neo4j export (only if --neo4j or --neo4j-push flag), Step 7a - FalkorDB export (only if --falkordb or --falkordb-push flag), Step 7b - SVG export (only if --svg flag), Step 7c - GraphML export (only if --graphml flag), Step 7d - MCP server (only if --mcp flag), Step 8 - Token reduction benchmark (only if total_words > 5000)

### Community 164 - "AmountInput"
Cohesion: 0.44
Nodes (7): AmountInput(), Props, Tooltip(), TooltipContent(), TooltipProvider(), TooltipTrigger(), react-currency-input-field

### Community 165 - "Color-system drift"
Cohesion: 0.22
Nodes (9): Auth colors form a separate palette, Chart tokens are disconnected from chart rendering, Color-system drift, Dark styling exists without a complete activation path, Header supporting text is another direct value, Near-identical blues are preserved separately, React Select border bypasses semantic borders, Semantic tokens and direct palette utilities coexist (+1 more)

### Community 166 - "Observability Readiness"
Cohesion: 0.22
Nodes (8): Context and correlation, Health, readiness, and diagnostics, Instrumentation seam, Metrics vocabulary, Observability Readiness, Readiness gate before selecting a tool, Structured logging, Trace boundaries

### Community 167 - "Module Improvement Map"
Cohesion: 0.22
Nodes (9): Accounts, Authentication and sessions, Categories, Cross-module priority table, Database and migrations, Module Improvement Map, Platform and HTTP, Summary (+1 more)

### Community 168 - "Codex — Atomic Multi-Chunk Transaction Imports"
Cohesion: 0.22
Nodes (9): Alternatives and tradeoffs, Codex — Atomic Multi-Chunk Transaction Imports, Contract and compatibility, Decision record, Invariants, Problem and evidence, Proposed design, Risks (+1 more)

### Community 169 - "Endpoint Review — `<operationId>`"
Cohesion: 0.22
Nodes (8): Contract, Decision, Documentation and observability readiness, Endpoint Review — `<operationId>`, Identity, Performance, Robustness, Security

### Community 170 - "Current UI inconsistencies"
Cohesion: 0.25
Nodes (7): Current UI inconsistencies, Duplicated component roles, Empty-state patterns drift, Interaction-pattern inconsistencies, Reading this document, Several state surfaces represent errors differently, Two unrelated select systems

### Community 171 - "State and behavior inconsistencies"
Cohesion: 0.25
Nodes (8): Bulk-delete confirmation is domain-specific in a generic component, Dashboard query errors are visually indistinguishable from valid emptiness, Disabled amount input is only partially disabled, Edit transaction form receives create-state semantics, Loading-state semantics are uneven, State and behavior inconsistencies, Toast wording and capitalization drift, Visible copy contains isolated errors

### Community 172 - "Accounts API — Technical Reference"
Cohesion: 0.25
Nodes (7): Accounts API — Technical Reference, Case-insensitive names, Deleting an account deletes its transactions, Non-negotiables when changing this code, Ownership model, Where the code lives, Why a foreign account is a 404

### Community 173 - "Accounts — Product Behavior"
Cohesion: 0.25
Nodes (8): Accounts are private to one user, Accounts — Product Behavior, Deleting an account deletes its transaction history, Messages a user can encounter, Names are unique without regard to case, Related, What an account is, What this deliberately cannot do yet

### Community 174 - ".ListCategories"
Cohesion: 0.21
Nodes (4): ListCategoriesRequestObject, ListCategoriesResponseObject, RefreshSessionRequestObject, RefreshSessionResponseObject

### Community 175 - "Current State and Constraints"
Cohesion: 0.25
Nodes (7): Current State and Constraints, Deferred behavior and schema decisions already recorded, Evidence that must be refreshed before implementation, Existing strengths to preserve, Gaps this project must close, Migration boundary, Snapshot

### Community 176 - "Security Standard"
Cohesion: 0.25
Nodes (7): Authorization and ownership, Database and secret controls, Identity and session requirements, Input and resource controls, Security review evidence, Security Standard, Security tests required per resource module

### Community 177 - "Performance and Query Engineering"
Cohesion: 0.25
Nodes (7): Index program, Performance acceptance gate, Performance and Query Engineering, Pool and runtime tuning, Query rules, Service objectives and budgets, Summary strategy

### Community 178 - "ProjectPolish PR Summary"
Cohesion: 0.22
Nodes (8): Follow-Up, Key Files, Main Changes, Operational Notes, ProjectPolish PR Summary, Purpose, Tests Added, Verification To Re-run

### Community 179 - ".QueryRow"
Cohesion: 0.29
Nodes (3): containsSQL(), faultingPool, faultingTx

### Community 180 - "PR Review Cycle"
Cohesion: 0.29
Nodes (6): Cycle (one iteration), Inputs, Iteration tracking, Merge protocol, Never, PR Review Cycle

### Community 181 - "Spacing, size, radius, and elevation drift"
Cohesion: 0.29
Nodes (7): Cards may receive two shadow systems, Equivalent maximum width expressed two ways, Focus rings differ, Page and component padding mix abstractions, Radius roles are not consistent by component size, Spacing, size, radius, and elevation drift, Two chart-height conventions

### Community 182 - "API Contract and Documentation Standard"
Cohesion: 0.29
Nodes (6): API Contract and Documentation Standard, Contract workflow, Documentation acceptance gate, Internal documentation, Operational documentation, Required operation documentation

### Community 183 - "useOpenAccount"
Cohesion: 0.47
Nodes (4): AccountColumn(), Props, OpenAccountState, useOpenAccount

### Community 184 - "graphify reference: query, path, explain"
Cohesion: 0.33
Nodes (5): For /graphify explain, For /graphify path, graphify reference: query, path, explain, Step 0 — Constrained query expansion (REQUIRED before traversal), Step 1 — Traversal

### Community 185 - "Responsive inconsistencies"
Cohesion: 0.33
Nodes (6): JavaScript and CSS breakpoint boundaries overlap, Overlay width rules differ by overlay type, React Select owns a one-off breakpoint, Responsive decisions mix render-time and CSS-time approaches, Responsive inconsistencies, Tables have overflow but no alternate mobile presentation

### Community 186 - "Nuchi Code Review"
Cohesion: 0.33
Nodes (5): Avoid, Local Context To Prefer, Nuchi Code Review, Output, Priority

### Community 187 - "schema.d.ts"
Cohesion: 0.33
Nodes (5): components, $defs, operations, paths, webhooks

### Community 188 - "0001 — transactions.date is timestamp without time zone"
Cohesion: 0.33
Nodes (5): 0001 — transactions.date is timestamp without time zone, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 189 - "react"
Cohesion: 0.22
Nodes (16): Props, Select(), Dialog(), DialogContent(), DialogDescription(), DialogFooter(), DialogHeader(), DialogOverlay() (+8 more)

### Community 190 - "0003 — Transaction rate limiting is in-memory"
Cohesion: 0.33
Nodes (5): 0003 — Transaction rate limiting is in-memory, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 191 - "0007 — JWT signing is HS256 with a single static secret"
Cohesion: 0.33
Nodes (5): 0007 — JWT signing is HS256 with a single static secret, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 192 - "0008 — Access tokens cannot be revoked mid-life"
Cohesion: 0.33
Nodes (5): 0008 — Access tokens cannot be revoked mid-life, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 193 - "0009 — No refresh-token reuse detection or session listing"
Cohesion: 0.33
Nodes (5): 0009 — No refresh-token reuse detection or session listing, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 194 - "0010 — Auth operations do not declare 500 responses in the contract"
Cohesion: 0.33
Nodes (5): 0010 — Auth operations do not declare 500 responses in the contract, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 195 - "0011 — No resend endpoint; email delivery is fire-and-forget"
Cohesion: 0.33
Nodes (5): 0011 — No resend endpoint; email delivery is fire-and-forget, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 196 - "0013 — Resource endpoints accept unbounded request bodies"
Cohesion: 0.33
Nodes (5): 0013 — Resource endpoints accept unbounded request bodies, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 197 - "0014 — Date filters are parsed in UTC, not the host timezone"
Cohesion: 0.33
Nodes (5): 0014 — Date filters are parsed in UTC, not the host timezone, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 198 - "0015 — Out-of-range amounts return 400, where legacy returned 500"
Cohesion: 0.33
Nodes (5): 0015 — Out-of-range amounts return 400, where legacy returned 500, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 199 - "0017 — The summary category chart silently excludes uncategorized spending"
Cohesion: 0.33
Nodes (5): 0017 — The summary category chart silently excludes uncategorized spending, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 200 - "0018 — A transaction's date is serialized as a UTC instant, not a calendar date"
Cohesion: 0.33
Nodes (5): 0018 — A transaction's date is serialized as a UTC instant, not a calendar date, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 209 - "Accessibility inconsistencies"
Cohesion: 0.40
Nodes (5): Accessibility inconsistencies, Clickable table cells are not keyboard controls, Color is part of several state distinctions, Good form ARIA is not matched by all custom controls, Icon-only controls do not all have accessible names

### Community 210 - "Duplicated and parallel patterns"
Cohesion: 0.40
Nodes (5): Account and category CRUD are near copies, Auth form composition is repeated, Duplicated and parallel patterns, List-page loading shells are repeated, Loading indicators are not a component

### Community 211 - "Q: got ticket #41 done by claude, please review the pushed changes and add comments beside the copilot ones, as you see fit"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: got ticket #41 done by claude, please review the pushed changes and add comments beside the copilot ones, as you see fit, Source Nodes

### Community 212 - "Q: recheck the fixes and give another pass for reviews"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: recheck the fixes and give another pass for reviews, Source Nodes

### Community 213 - "Q: give it a last review of all the files changend in the pr the comments and all that is in place right now in the pr leave comments for medim-high and high issues mid-low and low forget it we mande 3 iterations so far. if something is too big then addres it consitently here so it so i can anlyze it"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: give it a last review of all the files changend in the pr the comments and all that is in place right now in the pr leave comments for medim-high and high issues mid-low and low forget it we mande 3 iterations so far. if something is too big then addres it consitently here so it so i can anlyze it, Source Nodes

### Community 214 - "Q: Review PR #64 against post-merge PR #62 comment and issue #63"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Review PR #64 against post-merge PR #62 comment and issue #63, Source Nodes

### Community 215 - "Q: Review briefing 42 email verification and password reset design and confirm best-effort send, silent rate cap, TTL defaults, and plain-text future-frontend links"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Review briefing 42 email verification and password reset design and confirm best-effort send, silent rate cap, TTL defaults, and plain-text future-frontend links, Source Nodes

### Community 216 - "Q: Recheck latest PR 65 changes and comments for merge readiness"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Recheck latest PR 65 changes and comments for merge readiness, Source Nodes

### Community 217 - "Q: check the graphify outcome for nuchi and tell me how can we use it properly"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: check the graphify outcome for nuchi and tell me how can we use it properly, Source Nodes

### Community 218 - "Q: would it be a good complement or suplement adding colbymchenry/codegraph?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: would it be a good complement or suplement adding colbymchenry/codegraph?, Source Nodes

### Community 219 - "Q: recheck the changes and doanother review"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: recheck the changes and doanother review, Source Nodes

### Community 220 - "Q: check the tickets going foward, what we shall continue with after this pr merges ?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: check the tickets going foward, what we shall continue with after this pr merges ?, Source Nodes

### Community 221 - "Q: check the changes introduced for your reviews and recheck the pr if is there more to approach"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: check the changes introduced for your reviews and recheck the pr if is there more to approach, Source Nodes

### Community 222 - "Q: last comment address if you have anything more to say label it for a future improvement post migration and name it Codex-{title}"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: last comment address if you have anything more to say label it for a future improvement post migration and name it Codex-{title}, Source Nodes

### Community 223 - "Q: 3 more prs landed. Review them all"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: 3 more prs landed. Review them all, Source Nodes

### Community 224 - "Q: address the mas comments accordingly in the prs"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: address the mas comments accordingly in the prs, Source Nodes

### Community 225 - "Q: what else is left to address on all 5 prs ?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: what else is left to address on all 5 prs ?, Source Nodes

### Community 226 - "ValidationErrorJSONResponse"
Cohesion: 0.05
Nodes (20): ValidationErrorJSONResponse, BulkDeleteAccounts400JSONResponse, BulkDeleteCategories400JSONResponse, ConfirmPasswordReset400JSONResponse, CreateAccount400JSONResponse, CreateCategory400JSONResponse, CreateTransaction400JSONResponse, DeleteAccount400JSONResponse (+12 more)

### Community 227 - "Q: orchestrate everything to have a clean and smooth merge to master make nice merge comments after that check if everything is ok and give me a final merge summary"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: orchestrate everything to have a clean and smooth merge to master make nice merge comments after that check if everything is ok and give me a final merge summary, Source Nodes

### Community 235 - "0004 — Bulk-delete silently ignores missing/unowned IDs"
Cohesion: 0.40
Nodes (5): 0004 — Bulk-delete silently ignores missing/unowned IDs, How it was migrated, Proposed improvement, The concern, Why it was done this way

### Community 236 - "Codex Backend Improvements"
Cohesion: 0.40
Nodes (5): Codex Backend Improvements, Decision hierarchy, Definition of done for an endpoint, How to use this set, Source set

### Community 237 - "tech_debt_instructions.md"
Cohesion: 0.50
Nodes (3): <P0|P1|P2|P3> — <Short ticket title>, Priority Order, Technical Debt Tickets

### Community 238 - "graphify reference: add a URL and watch a folder"
Cohesion: 0.50
Nodes (3): For /graphify add, For --watch, graphify reference: add a URL and watch a folder

### Community 239 - "graphify reference: commit hook and native CLAUDE.md integration"
Cohesion: 0.50
Nodes (3): For git commit hook, For native CLAUDE.md integration, graphify reference: commit hook and native CLAUDE.md integration

### Community 240 - "graphify reference: incremental update and cluster-only"
Cohesion: 0.50
Nodes (3): For --cluster-only, For --update (incremental re-extraction), graphify reference: incremental update and cluster-only

### Community 241 - "ChartEnum"
Cohesion: 0.50
Nodes (4): ChartEnum, AREA, BAR, LINE

### Community 242 - "SpendingPieEnum"
Cohesion: 0.50
Nodes (4): SpendingPieEnum, PIE, RADAR, RADIAL

### Community 243 - "Typography drift"
Cohesion: 0.50
Nodes (4): Chart and tooltip typography is local, Input text changes at `md`, while React Select changes one pixel earlier, Semantic roles do not map to one type recipe, Typography drift

### Community 244 - "Q: Review PR 65 email verification and password reset changes"
Cohesion: 0.50
Nodes (3): Answer, Q: Review PR 65 email verification and password reset changes, Source Nodes

### Community 245 - "Q: Should Nuchi create shared Claude and Codex skills around Graphify and CodeGraph after the Go migration?"
Cohesion: 0.50
Nodes (3): Answer, Q: Should Nuchi create shared Claude and Codex skills around Graphify and CodeGraph after the Go migration?, Source Nodes

### Community 293 - "AuthServer"
Cohesion: 0.09
Nodes (25): GenerateToken(), HashToken(), TestGenerateToken_ProducesMatchingHash(), New(), conflictEmailError(), decodeAuthBody(), decodeJSONBody(), emailNotVerifiedError() (+17 more)

### Community 315 - "main"
Cohesion: 0.11
Nodes (16): databaseHost(), main(), getEnvDuration(), Config, authTestEnv, NewAuthServer(), NewResourceServer(), CapturedEmail (+8 more)

### Community 316 - "ServerInterfaceWrapper"
Cohesion: 0.04
Nodes (27): DatabaseErrorJSONResponse, NewStrictHandler(), NewStrictHandlerWithOptions(), BulkCreateTransactions500JSONResponse, BulkDeleteAccounts500JSONResponse, BulkDeleteCategories500JSONResponse, BulkDeleteTransactions500JSONResponse, CreateAccount500JSONResponse (+19 more)

### Community 332 - "Nuchi — Claude Code Guide"
Cohesion: 0.20
Nodes (9): Commands, Graphify, Hard Invariants, Legacy Code, Model Orchestration, Nuchi — Claude Code Guide, Post-Migration Improvements, Project (+1 more)

### Community 338 - "net/http.ResponseWriter"
Cohesion: 0.05
Nodes (10): healthHandler(), contextEchoProbe(), authServerMethods, resourceServerMethods, BulkDeleteRequest, DeleteAccount500JSONResponse, GetHealth200JSONResponse, ServerInterface (+2 more)

### Community 348 - "date-filter.tsx"
Cohesion: 0.13
Nodes (19): DateFilter(), Preset, PresetKey, PRESETS, Props, buttonVariants, Calendar(), CalendarDayButton() (+11 more)

### Community 372 - "Post-Migration Improvements"
Cohesion: 0.40
Nodes (5): Entry template, Index (claude-backend-improvements/), Layout, Post-Migration Improvements, The rule

### Community 388 - "transaction-route-utils.ts"
Cohesion: 0.09
Nodes (21): resolveApiBaseUrl(), ResolveApiBaseUrlOptions, Same-origin, always, MAX_BULK_CREATE_BODY_BYTES, MAX_BULK_CREATE_TRANSACTIONS, MAX_BULK_DELETE_BODY_BYTES, MAX_BULK_DELETE_TRANSACTIONS, checkTransactionMutationRateLimit() (+13 more)

### Community 396 - "Copilot Instructions — nuchi"
Cohesion: 0.33
Nodes (5): Code review: what NOT to comment on, Code review: what to focus on, Comment format, Copilot Instructions — nuchi, Repo conventions reviews should assume

### Community 435 - "NewRouter"
Cohesion: 0.07
Nodes (21): bearerToken(), probeAccountVisibility(), RequireAuth(), writeAccessTokenExpiredError(), optionalQueryParam(), withListTransactionsParams(), withResourceID(), withSummaryParams() (+13 more)

### Community 536 - "github.com/jackc/pgx/v5/pgtype.Text"
Cohesion: 0.18
Nodes (11): Queries, CreateTransactionParams, GetCategorySpendingParams, GetDailyTotalsParams, GetDailyTotalsRow, GetPeriodTotalsParams, GetPeriodTotalsRow, ListTransactionsParams (+3 more)

### Community 570 - "app/layout.tsx"
Cohesion: 0.21
Nodes (10): geistMono, geistSans, metadata, RootLayout(), Toaster(), getQueryClient(), makeQueryClient(), Props (+2 more)

### Community 576 - "InvalidRefreshTokenErrorJSONResponse"
Cohesion: 0.40
Nodes (3): InvalidRefreshTokenErrorJSONResponse, LogoutUser401JSONResponse, RefreshSession401JSONResponse

### Community 578 - "import-card.tsx"
Cohesion: 0.11
Nodes (22): Props, requiredOptions, SelectedColumnsState, CSVReaderRenderProps, Props, Client-side row numbers, Client validation, formSchema (+14 more)

## Knowledge Gaps
- **1093 isolated node(s):** `formSchema`, `FormValues`, `formSchema`, `FormValues`, `formSchema` (+1088 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1274 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **155 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `CSV Import — Technical Reference` connect `use-bulk-delete-transactions.ts` to `codex-backend-improvements/README.md`, `import-card.tsx`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **Why does `Resolved during the migration (#46)` connect `Transactions API — Technical Reference` to `import-card.tsx`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **What connects `formSchema`, `FormValues`, `formSchema` to the rest of the system?**
  _1093 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.09059029807130334 - nodes in this community are weakly interconnected._
- **Should `dependencies` be split into smaller, more focused modules?**
  _Cohesion score 0.05555555555555555 - nodes in this community are weakly interconnected._
- **Should `Current API Parity Fixtures` be split into smaller, more focused modules?**
  _Cohesion score 0.06451612903225806 - nodes in this community are weakly interconnected._
- **Should `cn` be split into smaller, more focused modules?**
  _Cohesion score 0.09480519480519481 - nodes in this community are weakly interconnected._