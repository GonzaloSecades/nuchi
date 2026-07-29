# 0005 — Category duplicate update returns 500, create returns 409

- **Migration ticket:** #36/#29 (contract), #44+ (handlers) (https://github.com/GonzaloSecades/nuchi/issues/36)
- **Area:** api
- **Priority guess:** high

## How it was migrated

Legacy behavior: creating a category whose name duplicates an existing one
(case-insensitive, per user) returns a clean `409`; but *renaming* a
category into a duplicate name hits the unique index unhandled and surfaces
as a `500`. The spec explicitly calls this out ("Decide current mismatches
explicitly in OpenAPI instead of inheriting them accidentally, especially
category duplicate update returning 500 while category duplicate create
returns 409"). Whatever the contract froze is what the Go handlers (#44+)
must reproduce — check `openapi/nuchi.openapi.json` for the decided status
before implementing.

## Why it was done this way

Port-not-redesign: the frontend's current error handling grew around the
real behavior, and changing response semantics mid-migration would desync
the fixtures.

## The concern

A `500` for a user-caused, perfectly foreseeable situation (renaming
"Food" to "food") is wrong on every axis: it's logged as a server fault,
it gives the user a generic failure instead of "that name is taken", and it
trains monitoring to ignore real 500s. This is the clearest inherited bug in
the API surface.

## Proposed improvement

**Resolved during the migration — this entry is closed, kept for history.**

The contract was the deciding authority per `spec.md` line 104 ("Decide
current mismatches explicitly in OpenAPI instead of inheriting them
accidentally, especially category duplicate update returning `500` while
category duplicate create returns `409`"), and `updateCategory` in
`openapi/nuchi.openapi.json` declares `409` with the shared
`DuplicateCategoryNameError` shape. #45 therefore shipped `409`, matching
duplicate-name create, rather than porting the legacy `500`.

This is the one deliberate behavior divergence in #45. It is not a parity
break: parity defers to the contract where the contract made an explicit
decision, which is exactly what the spec line above instructed. Frozen by
`TestCategoriesLive_DuplicateName_OnUpdate`, which covers both the exact and
case-differing (citext) collisions and fails if the handler regresses to
`500`.

Fixtures line 321 still documents the legacy `500` as *current Hono*
behavior; that description stays accurate for the legacy stack until #27
removes it.
