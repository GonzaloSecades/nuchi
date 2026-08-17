---
applyTo: 'lib/transaction-import.ts,lib/transaction-import.test.ts,features/transactions/**,app/(dashboard)/transactions/**'
---

Review transaction import changes for all-or-nothing financial safety. CSV
parser errors, empty/header-only files, blank internal rows, and uneven records
must be rejected before column mapping. Row values should be strictly typed and
validated before bulk creation.

Bulk creation must not partially import rows. Account/category ownership is
validated before insert, and database `WITH CHECK` RLS should fail loudly and
atomically if an unowned reference slips through.

Amounts remain signed integer milliunits. Dates remain plain `yyyy-MM-dd`;
avoid timezone conversion that changes the user's selected date.

Relevant focused tests live around `lib/transaction-import.test.ts` and
`features/transactions/api/*.test.ts`.
