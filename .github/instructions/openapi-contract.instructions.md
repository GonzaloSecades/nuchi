---
applyTo: 'openapi/**/*.json,openapi/**/*.yaml,backend/internal/openapi/**,lib/api/generated/**,lib/api/**/*.ts'
---

Review OpenAPI changes as contract-first changes. The hand-edited source is
`openapi/nuchi.openapi.json`; Go and TypeScript generated files must follow from
that contract, not be patched by hand.

Check that resource success responses preserve the `{ "data": ... }` envelope,
structured errors preserve `{ "error": { "code", "message" } }`, and auth
session/command responses keep their documented shapes. Transactions require
`currency`, defaulting to `ARS`; amounts are signed integer milliunits.

OpenAPI is 3.0.3. Nullable schemas should use the repository's existing 3.0
patterns instead of 3.1-only type arrays.

Generated output under `backend/internal/openapi/` and `lib/api/generated/` is
in scope only to confirm it is a mechanical regeneration of the contract; do not
review it line by line or ask for hand edits there.

Relevant validation is `bun run openapi:validate`, `bun run openapi:gen:go`,
and `bun run openapi:gen:ts`. Generated diffs should be mechanical.
