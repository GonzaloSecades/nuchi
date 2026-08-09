# Accounts API — Technical Reference

Go implementation reference for the accounts domain. Companion to the contract
(`openapi/nuchi.openapi.json`), which is authoritative for operation shapes and
status codes; this document covers the behavior the contract cannot express.

The Go account API shipped in
[#44](https://github.com/GonzaloSecades/nuchi/issues/44). Its six operations are
mounted inside the `RequireAuth` group in `backend/internal/http/router.go`.

## Ownership model

Every account row carries the authenticated user's internal UUID in `user_id`.
Handlers obtain that identity only from `UserIDFromContext` and run their SQL
inside `db.WithUserTx`, which binds the same UUID to `app.user_id` for the RLS
policy.

SQL still includes `user_id = caller` on every read and mutation. RLS is the
backstop, not a replacement for explicit ownership predicates or for stable
user-facing errors.

### Why a foreign account is a 404

Get, update, and delete match `id` and `user_id` in one statement. A missing ID
and another user's ID therefore produce the same no-row result and the same
`ACCOUNT_NOT_FOUND` response. This is deliberate non-disclosure: callers cannot
use the API to confirm that someone else's account exists.

Do not add an unscoped existence check to produce a more specific error. Apart
from leaking ownership, a check followed by a mutation would introduce a race
that the current conditional statement avoids.

Bulk delete follows the same privacy rule differently: IDs that are missing or
unowned are silently ignored, and the response contains only rows the caller
actually deleted. It never turns a skipped ID into a 404. The ergonomic cost of
that partial-success contract is recorded in
[improvement 0004](../../post-migration-improvements/claude-backend-improvements/0004-bulk-delete-silent-ignore.md).

## Case-insensitive names

`accounts.name` is PostgreSQL `citext`, with a unique index on `(user_id,
name)`. Consequences:

- `Cash` and `cash` collide for the same user.
- Two different users may each have an account named `Cash`.
- The database constraint, not an application pre-check, decides conflicts, so
  concurrent creates and renames cannot both win.
- Names are stored as submitted. `citext` folds case but does not trim
  whitespace; adding normalization would be a product and contract change.

Both create and update map PostgreSQL unique violations to
`DUPLICATE_ACCOUNT_NAME`. Keep that mapping tied to the stable error code and
constraint, never to human-readable message text.

## Deleting an account deletes its transactions

`transactions.account_id` is required and uses `ON DELETE CASCADE`. Deleting an
account—singly or in bulk—therefore permanently removes every transaction under
it. This is enforced by the foreign key, not by a second handler query.

That behavior is load-bearing for ownership: a transaction cannot exist without
an account, and its owner is derived through that account. Replacing the cascade
with application cleanup, nulling the account, or soft-deleting only one side
would break the model.

## Non-negotiables when changing this code

- Use `db.WithUserTx`; a bare `dbgen.New(pool)` has no RLS identity and can
  silently return zero rows.
- Derive identity from the verified request context only. Never accept a user
  ID from a body, path, query parameter, or header.
- Keep SQL ownership predicates in addition to RLS.
- Keep single update/delete as conditional statements rather than existence
  check plus mutation.
- Preserve database-enforced, per-user `citext` uniqueness and map conflicts by
  error code.
- Preserve the cascade when changing account or transaction foreign keys.
- Serialize empty lists and empty bulk-delete results as `[]`, never `null`.
- Log driver errors server-side only; never expose database internals.

## Where the code lives

| Concern                                          | File                                        |
| ------------------------------------------------ | ------------------------------------------- |
| Handlers, validation, projections, error mapping | `backend/internal/http/accounts.go`         |
| Queries and ownership predicates                 | `backend/internal/db/queries/accounts.sql`  |
| Table, unique index, cascade foreign key         | `backend/migrations/00002_finance_base.sql` |
| RLS policy                                       | `backend/migrations/00003_finance_rls.sql`  |
| RLS-bound transaction helper                     | `backend/internal/db/rls.go`                |
| Routes                                           | `backend/internal/http/router.go`           |
