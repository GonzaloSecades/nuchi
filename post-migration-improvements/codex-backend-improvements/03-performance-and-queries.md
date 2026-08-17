# Performance and Query Engineering

“Top performance possible” must mean the fastest design that preserves
correctness, isolation, maintainability, and an agreed cost envelope. Every
claim needs a reproducible measurement; intuition alone does not approve an
index or query rewrite.

## Service objectives and budgets

Before optimization work begins, define p50/p95/p99 latency and error-rate
objectives per operation class: auth commands, single-resource CRUD, lists,
bulk mutations, and summary. Define the tested concurrency, database size,
hardware tier, and cold/warm-cache conditions beside the numbers.

Each endpoint also gets budgets for:

- database round trips;
- rows scanned and returned;
- response bytes;
- allocations where material;
- maximum body and batch size; and
- database/handler timeout.

Initial budgets are hypotheses. Store benchmark output and query plans so
regressions compare like with like.

## Query rules

- Select only response/service fields; never use `SELECT *` in maintained SQL.
- Use set-based validation and mutations for bulk work. Query count must not
  scale linearly with batch length.
- Add deterministic ordering for every list. Introduce bounded pagination
  before the transaction table can grow without bound; prefer keyset cursors
  such as `(date, id)` over deep offsets.
- Keep ownership predicates in SQL even with RLS so intent, plans, and tests
  remain explicit.
- Avoid read-before-write when one conditional `INSERT`, `UPDATE`, or `DELETE
  ... RETURNING` can provide the same safe result. Keep separate checks only
  when the public error distinction is contractual.
- Treat PostgreSQL integer-to-Go conversion and summary accumulation as
  overflow-sensitive. Use wide types and explicit bounds for milliunits.
- Pass request context into every pgx/sqlc call so cancellation reaches the
  database.
- Prefer a small stable set of prepared query shapes. Optional filters must be
  plan-reviewed rather than assembled into unbounded SQL variants.

## Index program

The current Go schema gives a sound baseline:

- unique `(user_id, name)` plus `user_id` indexes for accounts/categories;
- `(account_id, date DESC)` for transaction lists and ranges; and
- `category_id` for category relations.

Do not add speculative indexes. For each candidate, record the target query,
production-shaped `EXPLAIN (ANALYZE, BUFFERS)` before/after evidence, write and
storage cost, selectivity, and redundant-index analysis.

Likely review points—not preapproved changes—include:

- deterministic list support with `id` appended to the transaction index;
- summary access across all user accounts, where ownership is reached through
  an account join and date range;
- category expense aggregation by owned accounts, date, amount sign, and
  category; and
- token lookup/expiry cleanup indexes for auth workflows.

Validate RLS-enabled plans using the production runtime role. A plan measured
as a bypass-capable owner is not representative.

## Summary strategy

The legacy summary performs two totals queries plus category and daily-series
queries. The Go implementation should first preserve semantics, then benchmark
alternatives:

- combine current/previous totals with conditional aggregation;
- push top-three-plus-Other aggregation into SQL when it reduces transferred
  rows without obscuring correctness;
- generate missing days in SQL or Go based on measured cost and clarity; and
- consider cached/pre-aggregated summaries only after direct indexed queries
  miss the service objective at realistic scale.

Any caching proposal must define key dimensions (`user`, date range, account,
currency, contract version), invalidation on every mutation path, staleness,
memory bounds, and cross-user isolation. No cache is preferable to a cache
without a correctness proof.

## Pool and runtime tuning

Configure pgxpool explicitly from deployment capacity: max/min connections,
acquire timeout, max connection lifetime/idle time, and health checks. The
maximum is derived from database connection capacity across all replicas, not
CPU count alone. Export pool saturation and acquisition latency later through
the observability boundary.

Set HTTP read-header, read, write, idle, header-size, and graceful-shutdown
limits. Profile before micro-optimizing JSON or allocations; database plans,
round trips, unbounded responses, and contention are the first targets.

## Performance acceptance gate

An endpoint cannot pass with only a green unit test. Required evidence:

- benchmark scenario and dataset generator/version;
- p50/p95/p99 and throughput before/after when changing performance behavior;
- number of SQL statements and rows affected at min/typical/max input;
- query plans for new or materially changed queries;
- pool-saturation and cancellation behavior under load;
- no cross-user or functional regression; and
- an explicit conclusion against its operation-class objective.

## Phase 3 operation-class budgets

These are the approved engineering budgets for the versioned local/CI baseline.
They are not production promises; #66 must bind them to a host size,
concurrency, and database capacity before launch.

| Class | Representative operations | Database round trips | Warm p95 objective | Bound |
| --- | --- | ---: | ---: | --- |
| Health | `getHealth` | 0 | 25 ms | constant work |
| Auth command | register/login/refresh/reset | 1-4, constant | 1.5 s | Argon2/mail handoff class documented separately |
| Single-resource read/write | account/category/transaction CRUD | 1-3, constant | 250 ms | one resource |
| Bounded list page | `listTransactions?limit=...` | 1 | 500 ms | 1-500 returned rows |
| Compatibility list | `listTransactions` without `limit` | 1 | 1 s | 366 days; temporary unbounded response |
| Bulk mutation | transaction bulk create/delete | constant with batch size | 1 s | 500 rows and documented byte cap |
| Summary | `getSummary` | 4 | 2 s | 366 days, six accounts, 100k-owner-row baseline |

Infrastructure errors must remain below 1% under the eventual agreed load;
validation, authorization, conflict, and rate-limit responses are not counted
as infrastructure faults. A class fails if its query count grows with input
rows even when latency happens to pass.

## Phase 3 list-pagination delivery

`listTransactions` now offers additive `(date,id)` keyset pagination:

- `limit` is 1-500 and requests one extra row to decide whether a continuation
  exists;
- `nextCursor` is an opaque URL-safe encoding of the last returned date/id;
- `cursor` requires `limit`, is validated before database work, and applies a
  strict descending tuple boundary;
- date/account filters and RLS identity stay independent of the cursor; and
- omitting `limit` preserves the existing unbounded response so the current UI
  is not silently truncated before it adopts pagination.

Phase 0's 100k-row, two-tenant warm sample returned 50,142 rows in 684 ms and
spilled an 8.5 MB external merge sort. The first `limit=100` page returned 101
database rows in 673 ms on the same disposable database: response size became
bounded and the sort became a 37 kB top-N heapsort, but PostgreSQL still scanned
the 50,142 qualifying rows. Pagination therefore fixes transfer/memory growth,
not the all-account scan cost.

Appending `id DESC` to the `(account_id,date DESC)` index remains a measured
candidate for account-filtered pages, not part of this PR. The local migration
ledger was already at version 6 while fresh `origin/master` ended at 5, proving
another schema-owning branch has the next migration number. Phase 3 must
remeasure after those branches merge instead of creating an out-of-order goose
migration or claiming another branch's schema as evidence.

No pgxpool limit is changed from a single-host plan. Pool maximum/minimum and
lifetime settings require #66's database connection budget across all replicas;
tuning them from CPU count or one serial plan would be speculation.
