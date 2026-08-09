# Categories API — Technical Reference

Go implementation reference for the categories domain. Companion to the
contract (`openapi/nuchi.openapi.json`), which is authoritative for operation
shapes and status codes; this document covers the behavior the contract cannot
express.

The Go category API shipped in
[#45](https://github.com/GonzaloSecades/nuchi/issues/45). Its six operations are
mounted inside the `RequireAuth` group in `backend/internal/http/router.go`.

## Ownership model

Every category row carries the authenticated user's internal UUID in `user_id`.
Handlers take that identity only from `UserIDFromContext` and execute through
`db.WithUserTx`, which binds the same UUID for PostgreSQL RLS.

Every query also includes an ownership predicate. Get, update, and delete match
`id` and `user_id` together, so a missing category and a foreign category are
indistinguishable and both return `CATEGORY_NOT_FOUND`. This is intentional: a
caller must not be able to discover whether another user's category exists.

Do not replace the conditional update/delete with an unscoped existence check.
That would both leak information and introduce a check-then-write race.

Bulk delete silently ignores missing and unowned IDs and returns only the
categories actually deleted. It never 404s for an individual skipped ID. See
[improvement 0004](../../post-migration-improvements/claude-backend-improvements/0004-bulk-delete-silent-ignore.md)
for the privacy/ergonomics tradeoff.

## Case-insensitive names

`categories.name` is PostgreSQL `citext`, with a unique index on `(user_id,
name)`. `Groceries` and `groceries` therefore collide for one user, while
different users may use the same name.

The unique index is the concurrency-safe decision point. Do not add a preflight
duplicate query or compare lowercased strings in Go. Names are stored exactly as
submitted: case comparison is insensitive, but whitespace is not trimmed.

Both create and update map unique violations to
`DUPLICATE_CATEGORY_NAME`. Update returning a structured duplicate conflict is
an explicit contract decision rather than a copy of legacy Hono's accidental
server error; the history is retained in
[improvement 0005](../../post-migration-improvements/claude-backend-improvements/0005-category-duplicate-update-500.md).

## Deleting a category keeps its transactions

`transactions.category_id` is optional and uses `ON DELETE SET NULL`. Deleting
a category—singly or in bulk—does not remove transaction history. Referencing
transactions survive and become uncategorized.

The foreign key performs this transition; the handler does not issue a second
update. This is deliberately opposite to account deletion, where the required
account relationship cascades transaction deletion. Any migration changing
either foreign key must preserve that distinction or make an explicit product
and contract change.

## Non-negotiables when changing this code

- Use `db.WithUserTx`; a bare `dbgen.New(pool)` has no RLS identity and can
  silently return zero rows.
- Derive identity only from the verified request context, never caller input.
- Keep SQL ownership predicates as well as RLS.
- Keep single update/delete as conditional statements rather than existence
  check plus mutation.
- Preserve database-enforced, per-user `citext` uniqueness and stable error
  codes on both create and update.
- Preserve `ON DELETE SET NULL`; deleting a category must not delete its
  transactions.
- Serialize empty lists and empty bulk-delete results as `[]`, never `null`.
- Log driver errors server-side only; never expose database internals.

## Where the code lives

| Concern                                          | File                                                           |
| ------------------------------------------------ | -------------------------------------------------------------- |
| Handlers, validation, projections, error mapping | `backend/internal/http/categories.go`                          |
| Queries and ownership predicates                 | `backend/internal/db/queries/categories.sql`                   |
| Table, unique index, set-null foreign key        | `backend/migrations/00002_finance_base.sql`                    |
| Category RLS policy                              | `backend/migrations/00003_finance_rls.sql`                     |
| Transaction category-ownership RLS check         | `backend/migrations/00004_transactions_category_owner_rls.sql` |
| RLS-bound transaction helper                     | `backend/internal/db/rls.go`                                   |
| Routes                                           | `backend/internal/http/router.go`                              |
