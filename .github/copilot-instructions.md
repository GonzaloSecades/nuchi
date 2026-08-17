# Copilot Instructions - nuchi

Personal finance app: a Next.js frontend and a separate Go API (chi, pgxpool,
sqlc, goose) over Dockerized PostgreSQL, with owned JWT auth and PostgreSQL
RLS. The migration from Hono/Drizzle/Neon/Clerk is complete — that stack was
removed in #84, #85 and #90, and none of it exists on disk. OpenAPI
(`openapi/nuchi.openapi.json`) is the contract source of truth.

For code review, prefer targeted findings over broad commentary. Review only
changed files plus the smallest directly related context needed to prove a
finding. Use the `.github/skills/code-review` project skill for PR reviews, and
load path-specific instructions only when the PR touches matching files.

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
- References to Hono, Drizzle, Neon or Clerk in `docs/specs/`, the parity
  fixtures, `post-migration-improvements/`, and "why this differs from legacy"
  comments. Those are the migration's historical record and are deliberately
  kept; flagging them as stale is a false positive. Active guidance (README,
  AGENTS.md, CLAUDE.md, this file) is a different matter — that should describe
  the current stack.
- Points already rebutted in a previous review round on the same PR, unless
  you have new evidence; repeating them stalls the merge protocol.
- Broad architecture suggestions, migration-era cleanup ideas, or future work
  unless the changed lines create a concrete regression. Existing debt belongs
  in `post-migration-improvements/`, not as drive-by PR review noise.

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
- Copilot reviews are requested manually. Do not assume automatic re-review on
  new pushes.
