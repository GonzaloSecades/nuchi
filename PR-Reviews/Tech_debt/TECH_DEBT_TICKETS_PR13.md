# Technical Debt Tickets

- Source: `PR-Reviews/13.Upload-Transactions-Import.md`, `tech_debt_SuggestionsPR08.md`
- Ticket count: 10 (<=10)

## Priority Order

## P0 — Harden CSV upload validation and sanitization

- Problem: CSV import accepts data without robust structure/type sanitization, which can allow malformed inputs to propagate into the app and create security/stability risk.
- Evidence (quote): "CSV upload lacks input validation and sanitization. The component should validate that uploaded data has the expected structure, sanitize string inputs, validate data types (e.g., ensure amounts are numeric, dates are valid), and limit file size to prevent potential security issues or application crashes from malformed data." (`tech_debt_SuggestionsPR08.md` → `CSV for upload` / `Input validation`)
- Why it matters: Unvalidated uploaded content can cause crashes and unsafe processing paths in production.
- Scope / Proposed fix:
  - Add a CSV schema validator before mapping/rendering rows.
  - Sanitize string fields (trim/control character handling) prior to transform.
  - Enforce numeric/date parsing with explicit error collection.
  - Block submit when validation errors exist and show actionable row-level feedback.
- Acceptance criteria:
  - [ ] Invalid CSV rows are rejected with clear, field-level error messages.
  - [ ] No mutation call is executed when validation fails.
- Effort: M
- Owner suggestion: Full-stack

## P1 — Guard empty/invalid CSV parse results before rendering

- Problem: Current logic can access headers/body on empty or header-only files, causing runtime breakage in the import flow.
- Evidence (quote): "The component doesn't handle the case where the uploaded CSV might be empty or have no data rows. If data.length is 0 or 1 (only headers), accessing data[0] and data.slice(1) could cause issues." (`tech_debt_SuggestionsPR08.md` → `CSV for upload` / `Input validation`)
- Why it matters: Users can hit a broken import screen from common invalid-input scenarios.
- Scope / Proposed fix:
  - Add early guards for `data.length < 2` before computing headers/body.
  - Render an empty/error state with retry action.
  - Add defensive checks in `ImportCard` and page upload handler.
- Acceptance criteria:
  - [ ] Uploading an empty or header-only CSV does not throw and shows a user-facing error state.
  - [ ] Import controls remain disabled when no valid data rows are present.
- Effort: S
- Owner suggestion: Frontend

## P1 — Enforce strict date-format validation for import rows

- Problem: Date parsing currently assumes one format and can generate invalid transformed values for mismatched source CSVs.
- Evidence (quote): "Import parsing expects `yyyy-MM-dd HH:mm:ss`; mismatched source formats may create invalid dates during transform." (`PR-Reviews/13.Upload-Transactions-Import.md` → `Risks & Rollout Considerations` / `High-Impact Risks`)
- Why it matters: Incorrect transaction dates are correctness issues that undermine trust and reporting accuracy.
- Scope / Proposed fix:
  - Validate date fields against accepted formats before transform.
  - Surface row-level date errors and block continuation until fixed.
  - Add explicit test cases for malformed and alternate date inputs.
- Acceptance criteria:
  - [ ] Rows with invalid date formats are rejected before submit.
  - [ ] Valid supported date formats are transformed consistently.
- Effort: M
- Owner suggestion: Frontend

## P1 — Fix transaction create-mode submit label regression

- Problem: The create flow currently displays the wrong action label, which is a user-facing correctness/UX regression.
- Evidence (quote): "Create action label now reads `Edit Transaction`." (`PR-Reviews/13.Upload-Transactions-Import.md` → `Technical Debt Assessment` / `Introduced Technical Debt`)
- Why it matters: Incorrect CTA text confuses users and increases form submission mistakes.
- Scope / Proposed fix:
  - Restore create-mode label to `Create Transaction`.
  - Keep `Edit Transaction` only for edit mode.
  - Add a UI assertion test for create/edit label switching.
- Acceptance criteria:
  - [ ] New transaction form shows `Create Transaction` in create mode.
  - [ ] Edit transaction form shows `Edit Transaction` in edit mode.
- Effort: S
- Owner suggestion: Frontend

## P2 — Replace `any` typing for CSV upload result payload

- Problem: Upload result typing is currently `any`, reducing type safety and increasing maintainability risk.
- Evidence (quote): "The onUpload prop uses the any type which bypasses TypeScript's type safety." (`tech_debt_SuggestionsPR08.md` → `Missing types` / `app/(dashboard)/transactions/upload-button.tsx`)
- Why it matters: Weak typing makes parser integration errors easier to introduce and harder to detect.
- Scope / Proposed fix:
  - Replace `any` with a typed parse result interface from the CSV parser types.
  - Type `onUpload` contract in `UploadButton` and its call sites.
  - Add compile-time checks for expected `data`, `errors`, and `meta` access.
- Acceptance criteria:
  - [ ] `upload-button.tsx` has no `any` in upload-result props.
  - [ ] TypeScript catches incompatible upload result shapes at compile time.
- Effort: S
- Owner suggestion: Frontend

## P2 — Replace `any` typing for import submit payload

- Problem: Import submission contract uses `any`, weakening confidence in transformed transaction data shape.
- Evidence (quote): "The onSubmit prop uses the any type which bypasses TypeScript's type safety." (`tech_debt_SuggestionsPR08.md` → `Missing types` / `app/(dashboard)/transactions/import-card.tsx`)
- Why it matters: Unclear payload typing increases runtime error risk and slows refactors.
- Scope / Proposed fix:
  - Define a typed import row model aligned to transaction insert payload.
  - Use strict `onSubmit` type signatures in `ImportCard` and callers.
  - Add type tests or compile checks for transform output fields.
- Acceptance criteria:
  - [ ] `import-card.tsx` has a strongly typed `onSubmit` payload contract.
  - [ ] Transform output compiles only when required fields are present and typed.
- Effort: S
- Owner suggestion: Frontend

## P2 — Add batch/chunk strategy for bulk import submission

- Problem: Current import submits all rows in a single mutation request, which can degrade reliability and performance with larger files.
- Evidence (quote): "All mapped rows are submitted in one bulk mutation." (`PR-Reviews/13.Upload-Transactions-Import.md` → `Technical Debt Assessment` / `Introduced Technical Debt`)
- Why it matters: Large single payloads can fail more often and are harder to recover from on partial errors.
- Scope / Proposed fix:
  - Implement client-side chunking with configurable batch size.
  - Add progress, retry, and partial-failure reporting.
  - Keep successful chunk commits while surfacing failed chunk details.
- Acceptance criteria:
  - [ ] Imports are sent in bounded batches rather than one unbounded payload.
  - [ ] Failed batches can be retried without re-importing successful batches.
- Effort: M
- Owner suggestion: Full-stack

## P3 — Add explicit row/file size guardrails for imports

- Problem: Import currently lacks explicit size limits, increasing risk of sluggish UX for oversized files.
- Evidence (quote): "LOW PRIORITY: - There's no limit on the size or number of rows in the CSV file that can be uploaded. Large CSV files could cause performance issues or even crash the browser by consuming too much memory." (`tech_debt_SuggestionsPR08.md` → `CSV for upload` / `Input validation`)
- Why it matters: Preventable client performance degradation can hurt reliability in edge cases.
- Scope / Proposed fix:
  - Define max file size and max row thresholds.
  - Reject oversized uploads with clear user feedback.
  - Document limits in import UI helper text.
- Acceptance criteria:
  - [ ] Oversized CSV files are rejected before parsing with a clear message.
  - [ ] Accepted files under threshold process without warnings.
- Effort: S
- Owner suggestion: Frontend

## P3 — Add in-context guidance text for CSV column mapping

- Problem: Users currently get little instruction on required mapping steps and skip behavior.
- Evidence (quote): "The ImportCard lacks user guidance. Consider adding instructional text explaining that users need to map CSV columns to transaction fields before importing." (`tech_debt_SuggestionsPR08.md` → `Guidance suggestion`)
- Why it matters: Clear instructions reduce mis-mapping errors and support overhead.
- Scope / Proposed fix:
  - Add concise helper copy above the import table.
  - Explicitly list required fields and explain `Skip`.
  - Include examples or tooltip for accepted date format.
- Acceptance criteria:
  - [ ] Import view contains instructional copy for mapping workflow.
  - [ ] Required fields and skip behavior are clearly documented in the UI.
- Effort: S
- Owner suggestion: Frontend

## P3 — Normalize import-view container layout with list view

- Problem: Import mode rendering is visually inconsistent with list mode due to missing shared wrapper.
- Evidence (quote): "When the IMPORT variant is active, the ImportCard is not wrapped in the same layout container (mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10) as the LIST variant. This creates an inconsistent layout and spacing." (`tech_debt_SuggestionsPR08.md` → `Ui improvements` / `app/(dashboard)/transactions/page.tsx`)
- Why it matters: Layout inconsistency creates avoidable UX friction and visual quality issues.
- Scope / Proposed fix:
  - Wrap import variant in the same page container classes used by list mode.
  - Verify spacing/alignment parity across breakpoints.
  - Add a snapshot/visual regression check for both variants.
- Acceptance criteria:
  - [ ] Import and list variants share consistent container width, spacing, and vertical offset.
  - [ ] Mobile and desktop layouts match expected design behavior for both variants.
- Effort: S
- Owner suggestion: Frontend
