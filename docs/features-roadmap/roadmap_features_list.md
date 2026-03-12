# Nuchi Roadmap Features List

## Roadmap Context

Nuchi is currently strongest as a personal finance tracker and analytics app for manually managed data. The near-term product strategy should focus on making imported financial data trustworthy, useful, and actionable.

Two constraints shape this roadmap:

- Open banking in Argentina is not production-ready yet, so direct bank sync should be treated as a future frontier, not a current dependency.
- A separate parser service is being built to convert PDFs and CSVs into transaction-shaped data that Nuchi can ingest.

Because of that, the roadmap should prioritize aggregation, normalization, planning, and financial visibility on top of manually uploaded or parser-generated data.

## 1. Unified Import Inbox

### Why this matters

Imported data is the app's main ingestion path for now. Nuchi should become the single place where raw financial data lands before it is reviewed and persisted.

### What this feature should achieve

- Create one canonical import flow for uploaded financial data
- Support current CSV uploads and prepare the same flow for parser-service outputs
- Treat parser responses and CSV files as different sources feeding the same transaction review pipeline
- Keep the final persistence step aligned with Nuchi's existing bulk-create model

### Product impact

This feature turns imports into a first-class workflow instead of a one-off utility. It also makes the app future-ready for parser integration and later open-banking ingestion.

## 2. Import Review, Normalization, and Deduplication

### Why this matters

Imported transaction data is only useful if users can trust it. Raw imports will often contain duplicates, inconsistent merchant names, malformed rows, or partial records.

### What this feature should achieve

- Strengthen the import preview before bulk creation
- Let users validate, correct, and exclude rows before saving
- Detect likely duplicates against already imported transactions
- Normalize merchant labels, dates, and amounts into a consistent Nuchi-friendly shape
- Make the review experience safe enough for real historical uploads

### Product impact

This is the highest-value improvement for an import-led product. It reduces friction, improves data quality, and makes users more willing to import larger sets of financial history.

## 3. Transfers, Balances, and Net Worth

### Why this matters

A finance app should not only show spending. It should also answer where the user stands overall. Right now, internal money movement and account-level financial position are underrepresented.

### What this feature should achieve

- Distinguish real spending from transfers between owned accounts
- Support account balance tracking over time
- Introduce a net-worth view across assets and liabilities
- Improve the meaning of dashboard analytics by separating movement from expense
- Make historical imported data useful for personal finance overview, not just transaction logging

### Product impact

This feature upgrades Nuchi from a transaction tracker into a broader financial visibility tool. It makes imported data feel much more complete and strategic.

## 4. Budgets and Monthly Planning

### Why this matters

Once users have imported and organized their data, the next product need is planning. Historical data becomes more valuable when it can inform monthly spending decisions.

### What this feature should achieve

- Add category-level monthly budgeting
- Compare actual spending against plan
- Highlight overspending and under-spending clearly
- Support a practical monthly planning workflow around real imported spending behavior
- Build a foundation for more proactive financial guidance later

### Product impact

This feature moves Nuchi from passive reporting into active decision support. It gives users an ongoing reason to return after their data is imported.

## 5. Recurring Transactions and Forecasting

### Why this matters

Users need help understanding not only what happened, but what is likely to happen next. With enough imported history, Nuchi should start surfacing recurring patterns and future cash expectations.

### What this feature should achieve

- Detect or define recurring income and expense patterns
- Let users review and manage recurring financial events
- Generate short-term cash-flow forecasts from known recurring transactions
- Surface likely upcoming bills, income, and recurring obligations
- Build a bridge between imported history and forward-looking planning

### Product impact

This feature makes the app more predictive and useful between imports. It also creates a natural extension path for future automation once open banking becomes viable.

## Strategic Notes

### Open banking readiness

Direct bank sync should remain a long-term strategic goal, but not a dependency for the current roadmap. The product should be designed so future bank connections become just another ingestion source beside CSV uploads and parser-service outputs.

### Parser service alignment

The parser service should not introduce a separate transaction workflow. Its output should flow into the same import inbox, review experience, and bulk-create path used by CSV imports.

### Current roadmap principle

For now, Nuchi should optimize for:

- reliable ingestion
- strong data review
- better financial aggregation
- actionable planning based on imported personal-finance data
