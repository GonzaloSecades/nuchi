# Transactions API — Technical Reference

Go implementation reference for the transaction endpoints. Companion to the
contract (`openapi/nuchi.openapi.json`), which is authoritative for shapes and
status codes; this document explains the behavior the contract cannot express.

Scope: the five non-bulk operations shipped in
[#46](https://github.com/GonzaloSecades/nuchi/issues/46). Bulk create/delete is
[#47](https://github.com/GonzaloSecades/nuchi/issues/47) and will extend this
page.

## Endpoints

All mount inside the `RequireAuth` group in `backend/internal/http/router.go`.

| Method | Path | Operation | Notes |
| --- | --- | --- | --- |
| GET | `/api/transactions` | `listTransactions` | Joined projection; `from`/`to`/`accountId` filters |
| GET | `/api/transactions/{id}` | `getTransaction` | Entity projection |
| POST | `/api/transactions` | `createTransaction` | Rate limited |
| PATCH | `/api/transactions/{id}` | `updateTransaction` | Full replacement; rate limited |
| DELETE | `/api/transactions/{id}` | `deleteTransaction` | Rate limited |

## Ownership model

`transactions` has **no `user_id` column**. Ownership runs through the required
account: `transactions.account_id → accounts.id`, `accounts.user_id = caller`.
Every query joins through that account, which is also what the
`transactions_owner` RLS policy asserts.

Consequences worth knowing:

- A missing transaction and one belonging to another user are indistinguishable
  to the caller — both return `404 TRANSACTION_NOT_FOUND`, deliberately.
- `accountId` filtering an unowned account returns an **empty list**, not a 404:
  the join simply excludes it.
- Category ownership is *not* implied by the foreign key. An FK proves the row
  exists, not who owns it, and FK checks bypass RLS entirely — which is why
  migration `00004` hardened `transactions_owner`'s `WITH CHECK` to require an
  owned-or-null category, and why the handler validates it explicitly.

## Two projections

`TransactionListItem` and `Transaction` are different shapes and must stay that
way.

| Field | List | Get / Create / Update |
| --- | --- | --- |
| `id`, `date`, `payee`, `amount`, `notes`, `currency` | yes | yes |
| `accountId`, `categoryId` | yes | yes |
| `account` (name) | yes | **no** |
| `category` (name, nullable) | yes | **no** |

The names are joined data, not transaction columns: renaming a category must
change what the next list returns without rewriting any transaction. Mapped by
`toTransactionListItem` and `toTransaction` in
`backend/internal/http/transactions.go`.

## Amounts

- Signed integer **milliunits**. Positive is income, negative is expense.
  `10.5` in the UI is `10500`. **Never floats, anywhere.**
- Column is `bigint` since migration `00005`; sqlc maps it to Go `int64`.
- API accepts **±9,007,199,254,740,991** (`2^53 − 1`) — JavaScript's
  safe-integer limit, so every value returned is exact in a browser.
- The bound is deliberately **narrower than the column**. `bigint` holds more;
  values in between are reachable only by direct SQL. Do not widen the
  validator to match the column.
- The contract states the bound as `minimum`/`maximum`, but **oapi-codegen emits
  no range validation** from those keywords. `parseAmount` is the only
  enforcement. Removing it reintroduces silent wraparound on narrowing.

## Currency

Required on every write, and must be exactly `"ARS"` (`CurrencyCode` is a
single-value enum). Omitted or unknown values are rejected with a field-level
400 rather than defaulted — a client that believes it is sending USD should
fail loudly. The contract's `default: "ARS"` is decorative and is queued for
removal.

Legacy has **no currency field at all**; it is a Go-side addition, so nothing
was broken by requiring it.

## Date filtering

Implemented in `backend/internal/http/daterange.go`, all in **UTC** from a
single captured `now`.

- `to` defaults to now. `from` defaults to 30 days before now.
- **A provided `to` does not re-anchor the default `from`.** Supplying only
  `to=2026-07-20` still starts 30 days before *now*, not before that date.
- A provided `from` is start-of-day; a provided `to` is end-of-day.
- Inclusive at both ends: `date >= from AND date <= to`.
- Inclusive span capped at **366 days**. Endpoints 365 days apart span 366 days
  and are the largest accepted; 367 is rejected.

### Presence vs emptiness

`from`, `to` and `accountId` distinguish **omitted** from **present but empty**.
`url.Values.Get` returns `""` for both, so presence is carried as a `*string`
via `optionalQueryParam`. Only omission defaults; `?from=` is malformed and
returns 400, because none of these parameters sets `allowEmptyValue` (false by
default in OpenAPI 3.0.3) and each references a schema with a minimum length.

Clients must therefore **omit unset filters** rather than send `''`. See
`lib/query-params.ts`.

### The three date errors

Reproduced verbatim from the fixtures; do not paraphrase them.

| Condition | `INVALID_QUERY` message |
| --- | --- |
| Unparseable or empty date | `from and to must use yyyy-MM-dd dates.` |
| `from > to` | `from must be less than or equal to to.` |
| Span > 366 days | `Date range cannot exceed 366 days.` |

`from`/`to` are read raw from the query string rather than through the generated
params: `FromDate`/`ToDate` are `openapi_types.Date`, a struct that cannot hold
a malformed value, so binding through it would discard exactly the inputs these
messages exist for.

## `categoryId` states

| Request | Meaning |
| --- | --- |
| omitted | no category |
| `null` | no category |
| `"cat_1"` | reference, ownership validated |
| `""` | **invalid** — `400 VALIDATION_ERROR`, `categoryId` field error |

`""` is neither null nor a valid `ResourceId` (`minLength: 1`), so it fails
request validation before any lookup. A 404 would assert that a syntactically
valid reference was missing, which is a different claim.

`PATCH` is a **full replacement** despite the verb: every field is required, and
an omitted `categoryId` clears the association rather than leaving it unchanged.
A true partial PATCH is a post-parity change and needs a `{Set, Value}` wrapper.

## Ordering

`ORDER BY t.date DESC, t.id DESC`. The `id` tiebreak is not cosmetic: most rows
share midnight, so date-only ordering was nondeterministic between identical
requests. Clients could never rely on it, so pinning it changes nothing
observable while removing the flapping — and it is a precondition for cursor
pagination later.

## Rate limiting

`backend/internal/ratelimit` — 60 accepted mutations per 60 seconds, per
`(user, action)`, where action is create/update/delete. Budgets are independent.

- Mutex-guarded: Node's event loop serialized the original `Map` for free; Go
  serves requests concurrently, so an unsynchronized port would let more than 60
  through and would race.
- Clock is injected, so tests advance time instead of sleeping.
- **Rejected attempts are not recorded**, so a client hammering a limited
  endpoint cannot keep pushing its own window forward and lock itself out.
- Checked *after* auth and body validation but *before* the database work, so a
  request that goes on to 404 still consumes its attempt — matching legacy.
- `Retry-After` is `max(1, ceil(remaining_ms / 1000))`. It is observable, so the
  rounding matches exactly.

In-memory, matching legacy: state resets on restart and N replicas permit N×60.
Tracked as improvement 0003.

## Error taxonomy and precedence

| Status | Code | When |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Malformed body, bad amount, bad currency, empty `categoryId` |
| 400 | `INVALID_QUERY` | Bad or empty `from`/`to`/`accountId` |
| 401 | `UNAUTHORIZED` / `ACCESS_TOKEN_EXPIRED` | Handled by `RequireAuth` |
| 404 | `ACCOUNT_NOT_FOUND` | Referenced account missing or unowned |
| 404 | `CATEGORY_NOT_FOUND` | Referenced category missing or unowned |
| 404 | `TRANSACTION_NOT_FOUND` | Target transaction missing or unowned |
| 429 | `TRANSACTION_MUTATION_RATE_LIMITED` | Budget exhausted; carries `Retry-After` |
| 500 | `DB_ERROR` | Database failure; per-operation message, never the driver error |

**Precedence on `PATCH` is observable and inherited from legacy:** the
referenced account is validated, then the category, and only then is the
transaction matched. A request naming both a missing transaction *and* a missing
account reports `ACCOUNT_NOT_FOUND`.

## Reference validation and the FK race

Ownership checks run inside the same `WithUserTx` as the write, so validation
and mutation share one RLS-bound identity and land together or not at all.

That is **not** the same as excluding a concurrent delete. The checks are
unlocked `SELECT`s, so under READ COMMITTED another transaction may delete a
referenced row between the check and the write; the write then fails with
SQLSTATE `23503`. `referenceErrorFromForeignKeyViolation` classifies that by
constraint name and returns the same 404 the up-front check would have.

`SELECT ... FOR KEY SHARE` was considered and rejected: locking every mutation
to prevent an outcome the mapping already reports correctly is not worth the
contention. Legacy has the identical race and answers it with a 500.

The mapping switches on PostgreSQL's auto-generated constraint names
(`transactions_account_id_fkey`, `transactions_category_id_fkey`), so a
migration that renames either would silently revert this to a 500 — guarded by
`TestForeignKeyConstraintNamesMatchTheSchema_LiveDatabase`.

## Non-negotiables when changing this code

- Every owned-table query goes through `db.WithUserTx`. Never `dbgen.New(pool)`:
  outside a transaction the RLS policy sees a NULL `app.user_id` and silently
  returns **zero rows** rather than erroring.
- Identity comes only from `UserIDFromContext`. Never a body field, query
  parameter, or header.
- SQL ownership predicates stay in addition to RLS. RLS is the backstop, not the
  source of user-facing errors.
- Empty collections serialize as `[]`, never `null`.
- Driver errors are logged server-side only, never placed in a response body.

## Where the code lives

| Concern | File |
| --- | --- |
| Handlers, validation, projections | `backend/internal/http/transactions.go` |
| Date range semantics | `backend/internal/http/daterange.go` |
| Rate limiter | `backend/internal/ratelimit/mutation.go` |
| Shared resource plumbing | `backend/internal/http/resources.go` |
| Queries | `backend/internal/db/queries/transactions.sql` |
| RLS binding | `backend/internal/db/rls.go` |
| Routes | `backend/internal/http/router.go` |

## Deliberate divergences from legacy

Recorded in `post-migration-improvements/claude-backend-improvements/`:

| Entry | Divergence |
| --- | --- |
| 0006 | `amount` widened to `bigint`; cap raised from ~2.15M ARS |
| 0014 | Date filters parsed in UTC rather than the host timezone |
| 0015 | Out-of-range amount returns 400 rather than legacy's 500 |

Also intentional, driven by the contract rather than parity: `currency` required
(legacy has no such field), empty query parameters rejected, empty `categoryId`
rejected, `id` ordering tiebreak.

## Known gaps

No pagination, no idempotency keys on create, no true partial PATCH, in-memory
rate limiting only, and `date` is a `timestamp` modelling what is really a
calendar date. All queued for the post-parity hardening pass; see the registry.
