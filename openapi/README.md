# OpenAPI

This directory is the hand-edited OpenAPI contract source for the Go backend replacement.

## Layout

- `nuchi.openapi.json`: source OpenAPI document.
- `oapi-codegen.yaml`: Go server type generation config.
- `backend/internal/openapi/generated.gen.go`: generated Go server types and chi bindings.
- `lib/api/generated/schema.d.ts`: generated TypeScript types (`openapi-typescript`), consumed by an `openapi-fetch` client wired up in a later ticket.

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

Generator tooling is pinned; no Java runtime is required for either side.

Go server types use [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) **v2.7.2**, pinned in `backend/go.mod` via the [build-tagged tools pattern](https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module) (`backend/tools.go`, `//go:build tools`). The command below runs the module-pinned binary from `backend/` so `go.mod` supplies the version — it does not hit the network for a version resolution the way `go run pkg@latest` would:

```bash
bun run openapi:gen:go
```

This runs `cd backend && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config ../openapi/oapi-codegen.yaml ../openapi/nuchi.openapi.json`, writing `backend/internal/openapi/generated.gen.go` (chi server bindings + `strict-server: true` typed request/response structs + models).

TypeScript types use [`openapi-typescript`](https://openapi-ts.dev/) **7.13.0** (exact-pinned devDependency). It replaces `@openapitools/openapi-generator-cli`, which required a Java runtime and is unacceptable for CI/agent environments:

```bash
bun run openapi:gen:ts
```

This writes `lib/api/generated/schema.d.ts` (the `paths`/`components` types). [`openapi-fetch`](https://openapi-ts.dev/openapi-fetch/) **0.17.0** (exact-pinned runtime dependency) consumes that `paths` type; actual client wiring (typed hooks, base URL, auth header injection) is separate follow-up work, not part of generation.

**Why 3.0.3, not 3.1:** the contract targets OpenAPI **3.0.3**, decided in #54. oapi-codegen v2.7.2 has only partial OpenAPI 3.1 support and failed outright on this contract's original 3.1-style nullable schemas (JSON-Schema-2020-12 `"type": ["string", "null"]` and `anyOf` null unions). 3.0.3 uses `nullable: true` instead (plus an `allOf` wrapper for nullable `$ref` fields, since 3.0 ignores sibling keys next to `$ref`). This is a representation-only change — no request/response wire shape changed.
