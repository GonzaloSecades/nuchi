# CSV Import — Product Behavior

What the transaction CSV importer accepts, what users see when a row fails,
and what support must know about large files that stop partway through.

Written for product decisions and support answers. For implementation details
see [the technical reference](../api/import.md).

## What a user imports

The import screen reads a CSV and asks the user to map one column to each of
three required fields:

- Amount
- Date
- Payee

Each field can be mapped only once. Continue stays disabled until all three are
mapped. After the file passes validation, the user chooses one account; every
transaction from that import is assigned to it. CSV import does not set notes
or categories, and imported transactions use ARS.

## The accepted date format is exact

Dates must use:

`yyyy-MM-dd HH:mm:ss`

For example, `2026-04-17 13:45:00` is accepted. `17/04/2026`,
`2026-04-17`, and a value without seconds are not.

The time is accepted only as part of the CSV format. The saved transaction is
for the calendar day, so the example above is stored as `2026-04-17`.

Amounts must be valid finite numbers. They are converted to thousandths of a
peso, so `12.34` becomes `12340` milliunits. Payees cannot be blank. Leading
and trailing spaces are removed from all three mapped values.

## Validation happens before anything is saved

The whole file is checked before the account picker or upload begins. If any
value is invalid, **nothing is imported**.

The error panel lists up to the first five invalid values. If there are more,
it asks the user to fix those first and continue again to reveal the remaining
problems. Fixing one row never causes the other valid rows to be imported in
the background.

## Large files are saved in groups of 500

The app sends at most 500 transactions at a time. Each group is all-or-nothing:
if one row in that group fails server validation, none of that group is saved.

The entire file is not all-or-nothing. Groups are sent in order, so earlier
groups can already be saved when a later one fails. Once a group fails, the app
stops and does not attempt the remaining groups.

For a 1,200-row file, if the second group fails:

- rows 1–500 remain saved;
- rows 501–1,000 are not saved; and
- rows 1,001–1,200 were never attempted.

The screen reports an import failure even though the earlier prefix remains in
the transaction list. Re-importing the full file can duplicate that prefix;
there is no duplicate detection or replay protection yet. Support should help
the user correct and import only the unsaved remainder.

## How displayed row numbers work

There are two numbering systems today.

### Errors found before upload

The CSV header is row 1, so the first data record is shown as **Row 2**. These
numbers match the physical line numbers in the CSV file.

### Errors returned while saving

The first data record is shown as **Row 1**, with the header excluded. The app
keeps counting across 500-row groups, so the fourth record in the second group
is shown as **Row 504**, not Row 4.

To find a save-time error in the original CSV, add one for the header: displayed
Row 504 is physical CSV line 505.

This difference is confusing but current behavior. A support answer should
first establish whether the user is looking at the red validation panel or a
save-error toast.

## Messages a user can encounter

| Situation                                                  | What they see / what it means                                          |
| ---------------------------------------------------------- | ---------------------------------------------------------------------- |
| Missing or invalid amount                                  | Row-level amount message; no upload starts                             |
| Date not in the exact format                               | Row-level date message showing `yyyy-MM-dd HH:mm:ss`; no upload starts |
| Blank payee                                                | Row-level payee message; no upload starts                              |
| More than five client errors                               | First five are listed, with a count/message for the remainder          |
| No account selected                                        | `You must select an account to import transactions`                    |
| Server identifies bad rows                                 | A toast lists up to three row/field problems and counts any others     |
| Server cannot identify a row, or the request/network fails | `Failed to import transactions`                                        |
| Every group succeeds                                       | `Transactions imported successfully`                                   |

## Current limitations

1. **No whole-file rollback.** A later failure leaves earlier 500-row groups
   saved.
2. **No duplicate protection.** Retrying a file or a committed prefix can make
   duplicate transactions.
3. **One account per file.** A single import cannot distribute rows among
   several accounts.
4. **No categories or notes from CSV.** Those must be added after import.
5. **Two row-number conventions.** Preflight errors count the header; save-time
   errors do not.

## Related

- [Technical reference](../api/import.md)
- [Transactions — Product Behavior](transactions.md)
- [Future atomic import design](../../post-migration-improvements/codex-backend-improvements/Codex-atomic-multi-chunk-transaction-imports.md)
