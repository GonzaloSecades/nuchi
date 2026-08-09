# Accounts — Product Behavior

What accounts mean to a user, how naming and privacy work, what deletion does,
and the messages support will encounter.

Written for product decisions and support answers. For implementation rules see
[the technical reference](../api/accounts.md).

## What an account is

An account is a named container for transactions: Cash, Checking, Savings, or
any label the user chooses. Every transaction belongs to exactly one account,
and that relationship is also how the app decides who owns the transaction.

Accounts do not currently calculate or store a separate opening balance. The
dashboard and transaction views derive money activity from the transactions
inside the account.

## Accounts are private to one user

A user can list, view, rename, and delete only their own accounts. Asking for an
account that does not exist and asking for another user's account produce the
same response: `"Account not found."`

That ambiguity is deliberate. The app does not confirm that another person's
account exists.

When a bulk delete contains a stale, missing, or foreign ID, that ID is skipped
without an error. Owned matches are still deleted. This protects privacy but
means a bulk action can partially succeed; the response identifies only what
was actually removed.

## Names are unique without regard to case

One user cannot have both `Cash` and `cash`. Trying to create or rename into a
case-insensitive duplicate produces:

> You already have an account with this name.

Another user may use the same name—the rule is per person, not global.

Names are not trimmed or otherwise normalized. Leading and trailing spaces are
significant, so `Cash` and `␠Cash␠` are currently different names (`␠` marks a
space). The UI should not rely on whitespace to create distinctions people
cannot see easily.

## Deleting an account deletes its transaction history

Deleting an account permanently deletes every transaction assigned to it. The
dashboard totals and charts change accordingly. Bulk deletion has the same
effect for each account it removes.

This is not what category deletion does: deleting a category keeps its
transactions and makes them uncategorized. For accounts there is no equivalent
"unassigned" state, because every transaction must have an account.

There is no archive, restore, or automatic reassignment step today. Support
should treat account deletion as destructive, not as hiding an account from the
list.

## Messages a user can encounter

| Situation                                                 | Exact message                                       |
| --------------------------------------------------------- | --------------------------------------------------- |
| Name missing or empty                                     | `Name is required.`                                 |
| Name duplicates another, including a case-only difference | `You already have an account with this name.`       |
| Account missing or owned by someone else                  | `Account not found.`                                |
| Bulk delete has no selected IDs                           | `At least one id is required.`                      |
| Bulk delete contains an empty ID                          | `Ids must not be empty.`                            |
| Database problem                                          | Generic failure notice; details stay in server logs |

## What this deliberately cannot do yet

1. **No safe archive or restore.** Delete is permanent and cascades through all
   transactions in the account.
2. **No reassignment during deletion.** Transactions cannot be moved to another
   account as part of the delete action.
3. **No bank connection or synchronization.** The account is a manual named
   container today.
4. **No account type, institution, opening balance, or per-account currency.**
5. **No whitespace normalization.** Visually confusing names can differ only by
   leading or trailing spaces.

The backend hardening and product-decision queue for these gaps is the
[post-migration improvements registry](../../post-migration-improvements/codex-backend-improvements/README.md),
especially its [accounts review map](../../post-migration-improvements/codex-backend-improvements/07-module-improvement-map.md#accounts).

## Related

- [Technical reference](../api/accounts.md)
- [Categories — product behavior](categories.md)
- [Transactions — product behavior](transactions.md) — why every transaction
  requires an account
- [Bulk-delete partial-success history](../../post-migration-improvements/claude-backend-improvements/0004-bulk-delete-silent-ignore.md)
