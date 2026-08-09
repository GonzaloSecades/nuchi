# Categories — Product Behavior

What categories mean to a user, how naming and privacy work, what deletion does,
and the messages support will encounter.

Written for product decisions and support answers. For implementation rules see
[the technical reference](../api/categories.md).

## What a category is

A category is an optional label for grouping transactions, such as Groceries,
Rent, or Transport. Categories drive the dashboard's spending breakdown, but a
transaction can exist without one.

Renaming a category changes the label shown the next time transaction and
summary data are loaded. Transactions keep the category ID; their stored money,
date, payee, and notes do not change.

## Categories are private to one user

A user can list, view, rename, and delete only their own categories. A missing
category and another user's category produce the same response:
`"Category not found."`

The app intentionally does not reveal whether another person's category
exists.

Bulk delete also avoids that disclosure. Missing, stale, and foreign IDs are
silently skipped while owned matches are deleted, so a bulk action can partially
succeed without an error.

## Names are unique without regard to case

One user cannot have both `Groceries` and `groceries`. Creating or renaming into
a case-insensitive duplicate produces:

> You already have a category with this name.

The same name is allowed for different users. Names are stored as submitted and
are not trimmed, so leading or trailing spaces can still make two names
different even though case cannot.

## Deleting a category keeps the transactions

Deleting a category does **not** delete transaction history. Every referencing
transaction survives and becomes uncategorized. The next transaction list shows
no category for those rows, and their expenses still count in dashboard totals.

They no longer appear in the named category's spending slice. Because the
current category chart excludes uncategorized expenses, deleting a heavily used
category can make the chart total fall below the Expenses figure even though no
money was removed.

There is no archive or "move these transactions to another category" step
today. Single and bulk category deletion both use the same uncategorize behavior.

## Messages a user can encounter

| Situation                                                 | Exact message                                       |
| --------------------------------------------------------- | --------------------------------------------------- |
| Name missing or empty                                     | `Name is required.`                                 |
| Name duplicates another, including a case-only difference | `You already have a category with this name.`       |
| Category missing or owned by someone else                 | `Category not found.`                               |
| Bulk delete has no selected IDs                           | `At least one id is required.`                      |
| Bulk delete contains an empty ID                          | `Ids must not be empty.`                            |
| Database problem                                          | Generic failure notice; details stay in server logs |

Duplicate-name create and rename now use the same message. The old Hono rename
path reported a generic server failure; the Go contract deliberately corrected
that inconsistency.

## What this deliberately cannot do yet

1. **No archive or restore.** Delete immediately removes the category label
   from every transaction that used it.
2. **No merge or reassignment on delete.** Transactions can only become
   uncategorized; they cannot be moved to a replacement category in the same
   action.
3. **No hierarchy.** There are no parent categories or subcategories.
4. **No budgets, limits, icons, or colors.** A category is a name only.
5. **No whitespace normalization.** Visually confusing names can differ only by
   leading or trailing spaces.

The backend hardening and product-decision queue for these gaps is the
[post-migration improvements registry](../../post-migration-improvements/codex-backend-improvements/README.md),
especially its [categories review map](../../post-migration-improvements/codex-backend-improvements/07-module-improvement-map.md#categories).

## Related

- [Technical reference](../api/categories.md)
- [Accounts — product behavior](accounts.md)
- [Transactions — product behavior](transactions.md) — optional categories and
  uncategorized transactions
- [Duplicate-update decision history](../../post-migration-improvements/claude-backend-improvements/0005-category-duplicate-update-500.md)
- [Bulk-delete partial-success history](../../post-migration-improvements/claude-backend-improvements/0004-bulk-delete-silent-ignore.md)
