# 0015 — Out-of-range amounts return 400, where legacy returned 500

- **Migration ticket:** #46 (https://github.com/GonzaloSecades/nuchi/issues/46)
- **Area:** api
- **Priority guess:** low

## How it was migrated

`parseAmount` (`backend/internal/http/transactions.go`) rejects any amount
outside ±(2^53−1) milliunits with `400 VALIDATION_ERROR` and a field-level
message, before the value reaches the database.

Legacy has no such check. A JavaScript number goes straight to PostgreSQL,
the `int4` column raises `22003 numeric_value_out_of_range`, and the route's
bare `catch` turns that into `500 DB_ERROR`.

## Why it was done this way

The check itself is not optional. The generated `TransactionInput.Amount` is a
Go `int64`; any narrowing conversion of an out-of-range value wraps silently,
turning a large income into a negative expense with no error anywhere. Legacy
never had that failure mode, because JavaScript numbers reached the database
intact and were rejected there. Porting "faithfully" without a bounds check
would therefore have *introduced* a data-corruption bug that legacy does not
have — which is squarely the internal-hardening lane, not a parity question.

Given the check must exist, only the status code was a real choice, and 400
won: the contract declares `400 ValidationError` on these operations, and
returning a server error for a client error the server has already
identified and named is indefensible. It also stops a routine user mistake
from polluting 500-rate monitoring.

## The concern

This is nevertheless an **observable difference**: a client sending an
oversized amount gets `400 VALIDATION_ERROR` from Go and `500 DB_ERROR` from
Hono. The blast radius is small — the values involved are rejected either
way, so no request that used to succeed now fails — but the status code and
error body differ, and a client branching on 4xx-versus-5xx would take a
different path.

Note also that the bound the API enforces is **narrower than the column**
after migration 00005: `bigint` accepts far more than ±(2^53−1), and values
in between are reachable only by direct SQL. That is deliberate, so every
amount the API returns is exact in JavaScript. See entry 0006.

## Proposed improvement

Nothing to undo — the Go behavior is the correct one. Two follow-ups make it
tidier:

- The contract's `minimum`/`maximum` on the amount fields are documentation
  only: oapi-codegen emits no range validation from them, so the handler
  check is the sole enforcement. If the project later adopts a request
  validation middleware, wire it up so the contract and the code cannot
  drift apart silently.
- Update `docs/specs/18-go-backend-replacement/api-parity-fixtures.md` to
  describe the 400 once the legacy stack is gone in #27; until then the
  fixture correctly documents Hono's 500.
