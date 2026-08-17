---
name: code-review
description: Review Nuchi pull requests with high signal and low token usage. Use for Copilot code review on PRs, especially changes touching money amounts, auth ownership, RLS, OpenAPI contracts, generated clients, CSV transaction imports, or TanStack Query hooks.
---

# Nuchi Code Review

Review changed lines first. Read only directly related callers, tests, contract
operations, and SQL needed to prove a finding. Do not survey the whole repo.

## Priority

Flag only concrete issues, in this order:

1. Money correctness: transaction amounts are signed integer milliunits. No
   floating-point money math, unit ambiguity, lossy rounding, or partial imports.
2. Auth and ownership: identity comes from the verified token. User-owned reads
   and writes must be explicitly scoped by `auth.userId`; RLS is a backstop.
3. Contract drift: `openapi/nuchi.openapi.json` owns API shape. Resource success
   responses use `{ "data": ... }`; errors use `{ "error": { "code", "message" } }`;
   transaction `currency` is required and defaults to `ARS`.
4. Real correctness bugs with a concrete failing scenario.
5. Missing tests for changed behavior or ticket acceptance criteria.

## Avoid

- Style, naming, formatting, import order, or subjective rewrites.
- Generated code in `backend/internal/openapi/` or `lib/api/generated/`.
- Re-litigating historical Hono/Drizzle/Neon/Clerk references in specs,
  fixtures, or post-migration notes.
- Repeating a prior Copilot comment unless new changed lines make it true.
- Asking for extra abstraction, generic cleanup, or broad refactors unrelated to
  the changed lines.

## Local Context To Prefer

- Backend behavior: `backend/internal/http/`, `backend/internal/db/queries/`,
  `backend/migrations/`, and `backend/README.md`.
- Frontend API usage: `features/*/api/`, `lib/api/`, and TanStack Query hooks.
- Contract changes: `openapi/nuchi.openapi.json`, `openapi/README.md`, generated
  outputs under generated-only paths.
- Import changes: `lib/transaction-import.ts`, `features/transactions/api/`,
  and transaction import UI under `app/(dashboard)/transactions/`.

## Output

Use `[high]`, `[medium]`, or `[low]` at the start of each comment. Include the
specific failing scenario or violated contract. Prefer one comment per root
cause.
