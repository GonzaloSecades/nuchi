# Differential parity harness

Drives the same input at both stacks and compares what comes back.

**This can only be built while both stacks exist.** After
[#27](https://github.com/GonzaloSecades/nuchi/issues/27) removes Hono there is
nothing left to compare against, and any divergence still present becomes
simply "how the app behaves". That is the whole reason this lands before the
cutover rather than after it.

## The gap it closes

Two oracles already exist and neither one compares the stacks:

- `docs/specs/18-go-backend-replacement/api-parity-fixtures.md` records what
  Hono does. It is a document; nothing executes it.
- The Go test suite asserts Go against the contract.

Both pass while disagreeing with each other. A difference that the contract does
not pin down — a timezone, an ordering tiebreak, a rounding step — sits in the
gap between them, invisible to either, until `USE_GO_API` flips and it becomes a
production bug on cutover day.

That is not hypothetical: the first thing this harness was pointed at, it found
one. See "Date serialization" below.

## Running it

Needs Postgres and a Go API. From the repo root:

```bash
wsl -d Ubuntu-20.04 -u root -e sh -c "service postgresql start; sleep 7200"
```

Then the API, from `backend/`:

```bash
DATABASE_URL='postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable' AUTH_JWT_SECRET='local-parity-secret-at-least-32-bytes-long' APP_BASE_URL='http://localhost:3000' SMTP_ADDR='localhost:1025' MAIL_FROM='nuchi@localhost' go run ./cmd/api
```

Then the harness:

```bash
ADMIN_DATABASE_URL='postgres://postgres:postgres@localhost:5432/nuchi?sslmode=disable' bun test tests/parity/
```

Without those variables every test **skips** rather than passes. A differential
test that silently passes because it could not reach either stack is worse than
no test at all — it looks like evidence. Check for `skip` in the output before
believing a green run.

`ADMIN_DATABASE_URL` is the admin role deliberately: the harness seeds and reads
across users, and the app role is subject to RLS, which would return zero rows
and make every comparison trivially equal.

## How each side is represented

| Stack | How it is driven                                             |
| ----- | ------------------------------------------------------------ |
| Go    | Over HTTP, as a real client, with a real session             |
| Hono  | Through **node-postgres**, the driver its Drizzle setup uses |

The Go side is fully end-to-end: the harness registers a user, verifies it with
SQL (the emailed token is stored hashed, so the raw value only exists in an
email, and depending on a mail catcher would tie a data comparison to SMTP),
logs in, and calls the API with the resulting Bearer token.

**The Hono side is the honest limitation here.** Its routes authenticate through
Clerk, which is not scriptable from a test, so this compares at the data layer
rather than through Hono's handlers. For the class of difference that motivated
the harness — how stored values become JSON — that is faithful, because the
handler passes those values straight through. It would _not_ catch a difference
introduced inside a Hono handler, such as an ordering or aggregation choice.

Closing that gap means either mocking `@hono/clerk-auth` and calling the Hono app
in-process with `app.request()`, or standing up a Clerk test session. Both are
viable; neither was needed for what this found first. That is the seam to extend
at, and it is the reason the support layer is a separate module.

## Known divergences

`divergences.ts` lists the deliberate ones as data, each pointing at where it is
argued — a numbered entry in
`post-migration-improvements/claude-backend-improvements/`, or a "Divergences"
section under `docs/api/`.

Keeping that list is what makes the harness usable. Without it a run is a wall
of red that has to be re-triaged by hand every time, and the one real regression
hides among the decisions we already made on purpose. **Anything reported that
is not on the list is a finding.**

## Date serialization

The first finding, and currently the only entry marked `expected: false`.

`transactions.date` is `timestamp without time zone`. Both stacks read the same
bytes and disagree about what timezone those bytes are in:

|                      | value                      |
| -------------------- | -------------------------- |
| stored               | `2026-08-07 00:00:00`      |
| node-postgres (Hono) | `2026-08-07T03:00:00.000Z` |
| Go                   | `2026-08-07T00:00:00Z`     |

A browser renders those as **different calendar days** anywhere west of
Greenwich. Every transaction date shows a day early — including for Argentina,
which is the app's own market and its default currency.

Registry entry 0014 covers the _filter_ side of the same underlying modelling
problem, and `docs/api/transactions.md` notes under "Known gaps" that `date` is
a `timestamp` modelling what is really a calendar date. The response side does
not appear to be recorded anywhere.

It is deliberately **not** marked as an expected divergence. It is a regression
against Hono rather than a decision, and marking it expected would retire the
only test that reports it. The test asserts the difference _is present_ rather
than asserting the two agree: an assertion that fails today, inside a parity
freeze that forbids fixing it, gets ignored and then deleted. Written this way
it documents the defect, proves it is still there, and fails loudly the day
someone changes either side — which is exactly when someone should look.
