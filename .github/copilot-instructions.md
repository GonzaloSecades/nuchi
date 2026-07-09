# Copilot Instructions — nuchi

Personal finance app mid-migration from Next.js/Hono/Drizzle/Neon/Clerk to a
separate Go API (chi, pgxpool, sqlc, goose) with Dockerized PostgreSQL, owned
JWT auth, and PostgreSQL RLS. OpenAPI (`openapi/nuchi.openapi.json`) is the
contract source of truth. Behavior parity is defined by
`docs/specs/18-go-backend-replacement/spec.md` and
`docs/specs/18-go-backend-replacement/api-parity-fixtures.md`.

## Code review: what to focus on

Review comments are valuable when they identify, in descending priority:

1. Money-math errors: transaction amounts are signed integer milliunits;
   any float math on money, rounding drift, or unit confusion is a high
   finding.
2. Ownership/auth gaps: queries missing user-ownership predicates, identity
   read from request bodies instead of the verified token, unprotected
   mutating endpoints.
3. Divergence from the parity fixtures or the OpenAPI contract: wrong status
   codes, error shapes, missing `{ "data": ... }` envelope, missing required
   `currency` (default `ARS`).
4. Real correctness bugs with a concrete failing scenario.
5. Missing tests for a ticket's stated acceptance criteria.

## Code review: what NOT to comment on

- Style, naming, formatting, import order: ESLint/Prettier/gofmt own these.
- Subjective preferences or alternative designs when the implemented one is
  documented in the spec, fixtures, or OpenAPI operation descriptions. Those
  documents decide; do not re-litigate them.
- Speculative issues phrased as "if X were the case" without verifying X in
  the repo. Check the code before asserting a failure mode.
- Generated code under `backend/internal/openapi/` and `lib/api/generated/`.
- The dummy Clerk fallback keys in CI: intentional and documented; the Clerk
  dependency is removed after Go parity (#27).
- Legacy Hono/Drizzle code under `app/api/[[...route]]` and `db/`: reference
  material scheduled for deletion; only flag changes that ADD new features
  to it.
- Points already rebutted in a previous review round on the same PR, unless
  you have new evidence; repeating them stalls the merge protocol.

## Comment format

- Tag every comment with an explicit severity: `[high]`, `[medium]`, or
  `[low]` at the start of the comment body.
- Include the concrete failing scenario or the violated spec/fixture line.
- Prefer one comment per root cause; do not fan out one issue into many
  comments.

## Repo conventions reviews should assume

- Bun is the package manager; Go 1.23 for `backend/`.
- Branch names `claude/<issue>-<slug>`; PR titles `[Issue - #<number>] ...`.
- PRs by solo maintainer via agent tooling; review threads are processed by
  an automated cycle capped at 3 iterations.
