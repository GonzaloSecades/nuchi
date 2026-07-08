# OpenAPI

This directory is the hand-edited OpenAPI contract source for the Go backend replacement.

## Layout

- `nuchi.openapi.json`: source OpenAPI document.
- `oapi-codegen.yaml`: Go server type generation config.
- `backend/internal/openapi/generated.gen.go`: generated Go server types and chi bindings.
- `lib/api/generated/typescript-fetch/`: generated TypeScript fetch client and types.

Generated files stay in generated-only paths. Business logic belongs outside those paths.

## Validate

```bash
bun run openapi:validate
```

The validator is intentionally local and dependency-free so contract edits can be checked before generator tooling is pinned.

## Shared Contract

- API errors use the structured `{ "error": { "code": "...", "message": "..." } }` shape defined by `ApiErrorResponse`.
- App resource success responses should preserve the existing `{ "data": ... }` envelope where practical.
- Auth success responses are separate command/session shapes and are not wrapped in the app resource envelope.
- App resource endpoints use Bearer access-token auth. Refresh and logout use the HttpOnly refresh-token cookie documented by `refreshTokenCookie`.

## Contract Coverage

`nuchi.openapi.json` covers the full #29 contract surface:

- health: `GET /health`
- auth: register, login, refresh, logout, verify email, request password reset, confirm password reset
- accounts: list, get, create, update, delete, bulk delete
- categories: list, get, create, update, delete, bulk delete
- transactions: list, get, create, update, delete, bulk create, bulk delete
- summary: dashboard summary with date/account filters

Resource behavior follows [`docs/specs/18-go-backend-replacement/api-parity-fixtures.md`](../docs/specs/18-go-backend-replacement/api-parity-fixtures.md) unless the OpenAPI operation description calls out an intentional migration change.

Intentional migration changes represented in the contract:

- Clerk route auth is replaced by owned auth endpoints.
- App resource endpoints use Bearer JWT access tokens.
- Login/refresh set an HttpOnly refresh-token cookie; refresh/logout consume it.
- API errors use the structured shared error format instead of the mixed current Hono string/Zod shapes.
- Transactions include required `currency`, defaulting to `ARS`.
- Category duplicate update returns structured `409` like duplicate create instead of preserving the current Hono `500` mismatch.

Current parity decisions preserved in the contract:

- App resource success responses use `{ "data": ... }`.
- Auth success responses are not app resource envelopes.
- Transaction amounts remain signed integer milliunits.
- Transaction and summary filters keep `from`, `to`, and optional `accountId`.
- Date filters require `yyyy-MM-dd`, are inclusive, and reject ranges over 366 days.
- Bulk create validates all transaction rows and references before inserting.
- Account, category, and transaction bulk delete ignore missing or unowned IDs and return only deleted owned IDs.
- Transaction mutation endpoints keep the current per-user/action `60 requests / 60 seconds` rate limit contract with `Retry-After`.

## Generate

Generation commands are wired and documented, but generated outputs remain deferred for #29. The repo does not yet pin generator versions, and this ticket is OpenAPI-first contract work rather than generated-client/server-type churn.

Go server types:

```bash
bun run openapi:gen:go
```

TypeScript fetch client and types:

```bash
bun run openapi:gen:ts
```

Run generation after generator versions are pinned or network-installed tools are explicitly approved for the ticket doing generation.
