# CSV Import — Technical Reference

Implementation reference for the browser CSV-import flow and its handoff to
transaction bulk create. The API contract
(`openapi/nuchi.openapi.json`) remains authoritative for request/response
shapes and status codes; this document explains the client pipeline, chunk
semantics, and error correlation the contract cannot express.

The user-facing companion is [CSV Import — Product
Behavior](../product/import.md). Bulk-create's database and ownership
invariants are covered in the [transactions technical
reference](transactions.md).

## End-to-end pipeline

Import is a frontend workflow over the existing transaction bulk-create
operation, not a separate server endpoint:

`CSV upload → column mapping → client validation → account selection → payload adaptation → sequential 500-row requests`

`react-papaparse` supplies a two-dimensional string array plus parse errors and
metadata. `prepareCSVImport` validates that complete result before import mode
can render. The importer then treats the first row as headers and every later
row as data. Users must map exactly one column to each required field—`amount`,
`date`, and `payee`—before the flow can continue. Choosing a field for a second
column clears its prior mapping.

After client validation, one selected account id is attached to every row.
`toBulkTransactionInput` adds the contract-required `ARS` currency and
normalizes absent notes/category values to `null`.

## Client validation

Upload preflight fails closed before the mapping UI:

- any Papa Parse error, aborted parse, or truncated parse rejects the file;
- an empty, blank-header, or header-only file is rejected;
- at least three columns must exist for the three required mappings;
- every data record must have the same width as the header;
- blank records inside the file are rejected, while conventional trailing
  blank records produced by a final newline are removed; and
- a UTF-8 BOM is removed from the first header cell and header whitespace is
  trimmed.

The page stores only the prepared two-dimensional data, not the unchecked
parser result. `ImportCard` still renders a recoverable empty state if it is
ever called without rows, rather than indexing `data[0]` and throwing.

`parseImportedTransactionRows` trims all three mapped values and accumulates
errors across the entire file.

- Amount must use plain decimal notation (not hexadecimal or exponent syntax),
  must be finite, is converted with
  `Math.round(amount * 1000)`, and must remain a JavaScript safe integer in
  milliunits.
- Date must round-trip exactly through date-fns using
  `yyyy-MM-dd HH:mm:ss`. Accepted input such as `2026-04-17 13:45:00` is sent
  to the API as the calendar date `2026-04-17`; the time is discarded.
- Payee must remain non-empty after trimming.

If any row has any client error, `onSubmit` is not called: no account is
selected and no API request is made. The import card shows the first five
errors, but the parser has already collected all of them.

### Client-side row numbers

The parser defaults `firstRowNumber` to 2 because row 1 is the CSV header.
Client validation therefore reports physical CSV lines: the first data record
is `Row 2`.

Do not casually unify this with API-error numbering. The two current displays
use different frames of reference, described below.

## Chunking and failure boundaries

The frontend splits validated rows with
`chunkItems(data, MAX_BULK_CREATE_TRANSACTIONS)`, where the shared limit is 500. Chunks are awaited sequentially.

One bulk-create request is atomic: every row and owned reference is validated,
then one insert runs inside one user-bound transaction. A rejected chunk writes
nothing from that chunk.

The whole file is **not** atomic. A file larger than 500 rows becomes several
independent requests. If a later chunk fails:

- every earlier successful chunk remains committed;
- the failing chunk commits nothing; and
- no later chunk is attempted.

The `submitted` counter advances only after a chunk succeeds. It is used solely
to translate a later chunk's local error indexes into file-wide data-row
numbers. There is no import id or idempotency key, so retrying the whole file
can duplicate the already committed prefix.

## Indexed validation errors

Bulk operations use operation-specific paths in
`details.fields[].path`. Indexes are zero-based within the submitted request.

| Source      | Path         | Meaning                                                          |
| ----------- | ------------ | ---------------------------------------------------------------- |
| Bulk create | `[3].amount` | `amount` failed on the fourth row in this request                |
| Bulk create | `$`          | The bare array itself is invalid, such as empty or over 500 rows |
| Bulk delete | `ids`        | The `ids` array itself is invalid                                |
| Bulk delete | `ids[i]`     | Entry `i` in `ids` is empty                                      |

CSV import only submits bulk create, but `bulkFieldErrors` intentionally decodes
both path dialects because it is the shared bulk-error utility. Bulk create
accumulates all invalid-row fields; bulk delete stops at its first unusable id.

### Malformed bodies have no `details`

A body that cannot be decoded has no trustworthy array or row to index. Invalid
JSON, an unknown field, or a trailing second JSON value therefore returns the
generic validation envelope with `details` omitted—not an empty `fields` array
and not a fabricated `$` entry.

`bulkFieldErrors` maps that absence to `[]`. The importer then uses its generic
`Failed to import transactions` message because it has no row-specific fact to
show. Keep absence distinct from a valid, indexed validation failure.

### API-error row numbers

`formatBulkErrorSummary` converts an API index with:

`displayed row = index + successfully submitted rows + 1`

This display counts **data records**, not physical CSV lines. The first data
record is `Row 1`; index 3 in the second 500-row chunk becomes `Row 504`.
Because the CSV header is excluded, the corresponding physical CSV line is one
higher.

The offset must be the number of committed rows, not `chunkIndex * 500`: only a
successful request advances it. Removing the offset would report a failure in
row 504 as row 4 and send the user to the wrong record.

## Import invariants

- Column mapping and client parsing complete before account selection or any
  write.
- Parser errors and structurally incomplete CSV files never reach column
  mapping.
- Amounts remain signed integer milliunits and within the browser-safe range.
- Imported dates become calendar-date strings; never serialize the parsed
  `Date` with `toISOString()`.
- Every row in one import uses the account the user selected after validation.
- Each request contains at most 500 rows and is awaited before the next begins.
- A chunk is all-or-nothing, while a multi-chunk file may leave a committed
  prefix.
- Row errors retain their zero-based request index until the presentation layer
  adds the committed-row offset.
- A malformed body never invents per-row error details.
- The mutation hook invalidates transaction and summary queries after each
  successful chunk, but deliberately does not toast. The import page emits one
  success toast only after every chunk succeeds.

## Non-negotiables when changing this code

- Keep `MAX_BULK_CREATE_TRANSACTIONS`, the contract's bulk-create `maxItems`,
  and the Go handler's maximum synchronized.
- Preserve sequential awaiting unless the session/atomicity design changes.
  Parallel requests would complicate stop-on-failure behavior, row offsets,
  rate limiting, and duplicate recovery.
- Preserve strict date round-tripping. A permissive parser would silently
  reinterpret source data.
- Keep client preflight all-or-blocking: never silently drop invalid rows and
  import only the rest.
- Keep bulk paths indexed and zero-based at the API boundary. Convert to
  user-facing numbering only in `formatBulkErrorSummary`.
- Keep malformed-body `details` absent. There is no defensible row attribution
  before decoding succeeds.
- Do not claim whole-file atomicity without a new persisted import-session
  contract. A database transaction cannot span independent HTTP requests.

## Where the code lives

| Concern                                                  | File                                                        |
| -------------------------------------------------------- | ----------------------------------------------------------- |
| Upload state, sequential submission, row-offset tracking | `app/(dashboard)/transactions/page.tsx`                     |
| Column mapping and client error display                  | `app/(dashboard)/transactions/import-card.tsx`              |
| CSV-result preflight, row parsing, and normalization     | `lib/transaction-import.ts`                                 |
| 500-row splitting                                        | `lib/chunk-items.ts`                                        |
| Shared limits                                            | `lib/transaction-limits.ts`                                 |
| Contract payload adaptation                              | `features/transactions/api/transaction-payload.ts`          |
| Bulk mutation hook                                       | `features/transactions/api/use-bulk-create-transactions.ts` |
| Indexed error decoding and display numbering             | `features/transactions/api/bulk-errors.ts`                  |
| Server validation and atomic insert orchestration        | `backend/internal/http/transactions_bulk.go`                |
| Bounded strict body decoder                              | `backend/internal/http/resources.go`                        |

## Divergences and gaps

| Entry                                                                                                                                   | Note                                                                                                                     |
| --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| 0006                                                                                                                                    | Transaction amounts were widened from the legacy 32-bit column; import also enforces the JavaScript-safe milliunit range |
| 0015                                                                                                                                    | An out-of-range imported amount is rejected as a validation error rather than surfacing legacy's database failure        |
| 0016                                                                                                                                    | The Go bulk endpoint enforces the byte limit against the stream; legacy checked only declared `Content-Length`           |
| 0018                                                                                                                                    | A transaction's `date` is serialized as a UTC instant rather than a calendar date                                        |
| [Import-session proposal](../../post-migration-improvements/codex-backend-improvements/Codex-atomic-multi-chunk-transaction-imports.md) | Whole-file atomicity and replay-safe retries require a persisted import-session workflow                                 |
