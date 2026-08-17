# 0005-pattern verification — predictable conflicts must not become 500s

Date: 2026-08-17. Scope: #111 verification note only; no behavior change.

## Original pattern

Registry entry
[`0005-category-duplicate-update-500.md`](../claude-backend-improvements/0005-category-duplicate-update-500.md)
recorded a legacy mismatch: duplicate category create returned 409, while
renaming onto the same case-insensitive unique name escaped as 500.

## Current Go/OpenAPI verification

| Surface                      | Create conflict               | Update conflict               | Evidence                                                                                                                                     |
| ---------------------------- | ----------------------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Accounts `(user_id, name)`   | 409 `DUPLICATE_ACCOUNT_NAME`  | 409 `DUPLICATE_ACCOUNT_NAME`  | `isUniqueViolation` branches in `accounts.go`; exact and case-insensitive live tests for create/update                                       |
| Categories `(user_id, name)` | 409 `DUPLICATE_CATEGORY_NAME` | 409 `DUPLICATE_CATEGORY_NAME` | `isUniqueViolation` branches in `categories.go`; `TestCategoriesLive_DuplicateName_OnCreate` and `TestCategoriesLive_DuplicateName_OnUpdate` |

The OpenAPI contract declares the same 409 response components for each
create/update pair. PostgreSQL `citext` plus the per-user unique indexes make
case-only collisions take the same handler branch. Missing or unowned update
targets still map to their non-disclosing 404 before any conflict response.

Auth registration has a create-only email uniqueness command and already maps
its expected conflict to 409; there is no email-update operation with the 0005
shape. Transactions have no user-editable unique field. No remaining current
operation has the specific “known unique conflict on update becomes generic
500” mismatch.

## Conclusion

The 0005 pattern is resolved and regression-protected; no Phase 5 code change
is warranted. Future unique constraints must ship with an explicit SQLSTATE-
to-contract mapping and matching create/update conflict tests before they are
considered complete.
