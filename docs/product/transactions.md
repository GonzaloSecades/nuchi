# Transactions — Product Behavior

What transactions do from a user's point of view: the rules the app enforces,
the limits it imposes, the messages people will see, and what it deliberately
cannot do yet.

Written for product decisions and support answers. For endpoint shapes and
implementation rules see [the technical reference](../api/transactions.md).

## What a transaction is

One movement of money, recorded against exactly one account, optionally tagged
with one category.

| Field | Required | Notes |
| --- | --- | --- |
| Amount | yes | Signed. Positive is income, negative is an expense |
| Payee | yes | Free text — who was paid, or who paid you |
| Date | yes | Any calendar day, past or future. Not pre-filled |
| Account | yes | Must be one of the user's own accounts |
| Category | no | Optional; if set, must be one of the user's own categories |
| Notes | no | Free text |
| Currency | yes | ARS only today |

**A transaction cannot exist without an account.** That is also how ownership
works: a transaction belongs to whoever owns its account. Deleting an account
deletes its transactions. Deleting a *category* does not — those transactions
survive and simply become uncategorized.

## Money is stored exactly, never as a decimal

Amounts are held as whole **milliunits** — thousandths of a peso. `$10.50`
becomes `10500`. This is why the app never shows a rounding artifact: there is
no floating-point arithmetic anywhere in the money path.

Sign carries meaning rather than living in a separate field. Income is positive,
an expense is negative, and the summary derives income/expense/remaining by
looking at the sign.

### The amount ceiling

A single transaction can be at most **±9,007,199,254,740.991 ARS** — about nine
trillion pesos. Beyond that the app refuses the entry rather than storing a
wrong number.

This ceiling was raised deliberately. It was previously ~2,147,483 ARS, which at
current rates is roughly **USD 1,400** — below a month's rent in parts of Buenos
Aires, a medical bill, or a used-car payment. That was a real defect for an
Argentine finance app, not a theoretical limit, and it was fixed before the
transaction API shipped.

The new ceiling is set by what a browser can represent exactly. It is not a
storage limit; it is a "we will never show you a number we cannot round-trip
faithfully" limit.

## The transaction list

### Default view

With no filters, the list shows **the last 30 days**, newest first.

### Filters

- **Date range.** Both ends are inclusive: a range of `2026-07-10` to
  `2026-07-19` includes anything on either of those days.
- **Account.** Narrows to one account.

Two behaviors worth knowing for support:

- Setting only an end date does **not** move the start date to 30 days before
  it. The range still begins 30 days before today. Someone asking "why am I
  seeing transactions after my end date's month" is usually hitting this.
- The longest range is **366 days**. Anything wider is refused rather than
  silently truncated, because an unbounded range on a large history is slow and
  the app would rather say so.

### Ordering

Newest first by date. Transactions sharing the same day appear in a stable
order — repeating the same request returns the same order, which was not
previously guaranteed.

### Filtering by an account that isn't yours

Returns an empty list, not an error. The app does not confirm or deny that
another user's account exists.

## Categories are optional, and must be yours

A transaction can be left uncategorized, and clearing a category on an existing
transaction is allowed.

Attaching a category that belongs to someone else fails with "Category not
found" — the same message as a category that does not exist at all. The app
never reveals that another user's data exists.

The same applies to accounts.

## Editing replaces the whole transaction

Saving an edit submits **every** field, not just the changed ones. Practically:
if the form clears the category, the category is cleared. There is no "leave
this field alone" edit today — a partial edit API is a planned change.

## Rate limiting

A user may make **60 changes per minute**, counted separately for creating,
editing, deleting, bulk-creating and bulk-deleting. So 60 creates and 60 deletes
in the same minute is fine; 61 creates is not. A CSV import consumes the
bulk-create budget only, so importing does not lock a user out of editing.

When the limit is hit the app reports how many seconds to wait. This exists to
protect the database from a runaway import or a stuck retry loop, not to
restrict normal use — 60 manual entries in a minute is far beyond human pace.

Two current limitations, both known:

- The count resets if the server restarts.
- With multiple server instances the effective limit is higher, since each keeps
  its own count.

Neither matters at current scale; both are queued for the hardening pass.

## Messages a user can encounter

| Situation | What they see |
| --- | --- |
| Amount too large, fractional, or missing | Field error on the amount |
| Payee, date, or account missing | Field error on that field |
| Category or account not theirs, or deleted | "Account not found" / "Category not found" |
| Editing or deleting something already gone | "Transaction not found" |
| Bad date filter | "from and to must use yyyy-MM-dd dates." |
| Reversed date filter | "from must be less than or equal to to." |
| Range over a year | "Date range cannot exceed 366 days." |
| Too many changes too fast | Rate-limit notice with a wait time |
| Database problem | Generic failure notice; details go to server logs only |

Error responses never contain database internals.

## Currency

Every transaction is ARS. The field exists and is required so multi-currency can
arrive later without a data migration, but only ARS is accepted today. Sending
anything else is rejected rather than quietly stored, so no transaction can end
up mislabeled.

## Importing many at once

A CSV import posts transactions in batches of up to **500 rows**. Two rules
matter for support:

- **A batch is all-or-nothing.** If any row in a batch is invalid, nothing from
  that batch is saved — there is no partial import leaving half the rows behind.
- **A large import is several batches, and each is independent.** A 1,200-row
  file is three requests, so an early batch can succeed and a later one fail.
  The user is left with the successful batches saved. Re-importing the whole
  file would duplicate those rows, since there is no duplicate detection yet.

When a batch is rejected, every problem row is reported at once with its
position in the file, rather than one error per attempt.

There are also total-payload caps — roughly 1 MB per import batch and 100 KB per
bulk delete. A batch of 500 rows fits comfortably under these with ordinary
data, but they are reachable legitimately: payee and notes have no length limit
of their own, so around 2 KB of text per row is enough to cross the batch cap.
A user who hits one has a batch that is genuinely too large, not necessarily a
malformed one, and the fix is a smaller batch rather than different data.

## What this deliberately cannot do yet

Ordered roughly by how likely a user is to feel it:

1. **No pagination.** A long date range returns every matching transaction in
   one response. Fine for a personal history; the first thing to fix as data
   grows.
2. **No partial edit.** Editing always replaces the whole record.
3. **No duplicate protection on retry.** If a create is retried after a network
   failure, two identical transactions can result — and for a CSV import, a
   whole re-uploaded file duplicates everything already saved. Idempotency keys
   are planned.
4. **Single currency.**
5. **No per-user timezone.** Day boundaries are UTC, so a filter near midnight
   is judged in UTC rather than in the user's local day.
6. **No recurring transactions, splits, or transfers between accounts.** A
   transfer is currently two independent transactions.

## Related

- [Technical reference](../api/transactions.md)
- [Roadmap](../features-roadmap/roadmap_features_list.md) — import inbox and
  parser-service ingestion both land on top of transactions
- `post-migration-improvements/claude-backend-improvements/` — the queue behind
  the "cannot do yet" list
