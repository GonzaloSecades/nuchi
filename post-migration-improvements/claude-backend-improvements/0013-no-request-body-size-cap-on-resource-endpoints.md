# 0013 — Resource endpoints accept unbounded request bodies

- **Migration ticket:** #44 (https://github.com/GonzaloSecades/nuchi/issues/44)
- **Area:** http / input validation
- **Priority guess:** medium

## How it was migrated

The authenticated owned-resource endpoints (`/api/accounts/*` in #44;
categories, transactions, and summary follow the same helper in #45-#48)
decode request bodies through `decodeResourceBody` in
`backend/internal/http/resources.go`, which applies no `http.MaxBytesReader`
cap. A body of any size is read and decoded as long as it is well-formed
JSON with no unknown fields.

The unauthenticated auth endpoints are unaffected: they keep the 4 KiB
`maxAuthBodyBytes` cap from #41.

## Why it was done this way

#44 first shipped a 64 KiB cap on these endpoints. Review rejected it as a
parity break, correctly:

- Neither the legacy Hono validators (`app/api/[[...route]]/accounts.ts`)
  nor the contract's `AccountInput` / `BulkDeleteRequest` schemas declare
  `maxLength`, `maxItems`, or any byte-size bound.
- A contract-valid body just over the cap therefore returned
  `400 VALIDATION_ERROR` where Hono accepts and processes it — an
  observable behavior change introduced by the migration.
- The frontend's 500-item chunk size (`lib/chunk-items.ts`) is a client
  implementation detail, not an API constraint, so it cannot justify a
  server-side limit.

Parity freezes observable behavior, and an undocumented limit is squarely
observable. Two live tests now freeze the permissive behavior
(`create: large body is accepted, not capped` and
`bulk-delete: large id list is accepted, not capped` in
`accounts_live_test.go`) so a cap cannot creep back in unnoticed.

## The concern

An authenticated client can stream an arbitrarily large body into the JSON
decoder, and a bulk-delete can carry an unbounded `ids` array that becomes an
unbounded `id = ANY($1)` parameter. Memory is allocated proportional to the
body before any application-level validation runs. The server's `ReadTimeout`
(`cmd/api/main.go`) bounds the *time* a single request may spend streaming,
not the *bytes* it may allocate, and requiring authentication raises the cost
of abuse without eliminating it — a single compromised or careless client can
still pressure the process.

This is inherited from the legacy implementation, which has the same
property, so it is not a regression. It is a latent availability issue that
the port faithfully preserved.

## Proposed improvement

After parity: declare real limits in the OpenAPI contract first
(`maxLength` on `AccountInput.name` and the other resource name fields,
`maxItems` on every `BulkDeleteRequest.ids`), regenerate both sides, then
enforce them in the handlers with a documented `413` or `400` response. With
the bound in the contract, the cap stops being undocumented behavior and the
frozen-permissive tests above get updated in the same change.

A defense-in-depth byte cap well above any legitimate contract-valid body
(so it can never reject a conforming request) could land alongside it.
