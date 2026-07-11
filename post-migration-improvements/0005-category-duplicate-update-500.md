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

Make duplicate-name update return `409` with the same error shape as
duplicate-name create — the Go query layer already surfaces the unique
violation (SQLSTATE 23505) cleanly, so the handler change is trivial. Needs
a coordinated OpenAPI + fixtures + frontend-toast update, hence post-parity.
Should be among the first optimization tickets: user-visible, low effort,
removes a false-alarm class from error monitoring.
