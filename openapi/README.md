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

## Generate

Generation commands are wired but deferred in #35 because the repo does not yet pin generator tools and #29 owns the full resource contract.

Go server types:

```bash
bun run openapi:gen:go
```

TypeScript fetch client and types:

```bash
bun run openapi:gen:ts
```

Run generation after #29 fills the contract and generator versions are pinned or approved for network use.
