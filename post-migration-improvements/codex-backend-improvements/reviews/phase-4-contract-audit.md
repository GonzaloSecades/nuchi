# Phase 4 contract and operations audit

Date: 2026-08-17. Scope: #110. OpenAPI is the field-level source; the linked API
guides own client examples, limits, ordering, retry, and atomicity explanations.

| Operations                                                                                                                 | Guide                                            | Phase 4 result                                                                                                                                                                                                                                                |
| -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `getHealth`, `getReadiness`                                                                                                | API changelog and deploy runbook                 | Liveness/readiness split, 200/503 examples, no dependency detail exposure                                                                                                                                                                                     |
| `registerUser`, `loginUser`, `refreshSession`, `logoutUser`, `verifyEmail`, `requestPasswordReset`, `confirmPasswordReset` | `docs/api/auth.md`                               | Session/token/cookie behavior and atomic one-time consumption documented; internal-error response additions remain gated on [#119](https://github.com/GonzaloSecades/nuchi/issues/119)                                                                        |
| account CRUD and bulk delete                                                                                               | `docs/api/accounts.md`                           | Ownership/non-disclosure, conflicts, empty results, bulk semantics documented                                                                                                                                                                                 |
| category CRUD and bulk delete                                                                                              | `docs/api/categories.md`                         | Ownership/non-disclosure, nullable transaction relationship, conflict behavior documented                                                                                                                                                                     |
| transaction CRUD, list, bulk create/delete                                                                                 | `docs/api/transactions.md`, `docs/api/import.md` | Money/date rules, ordering, body/batch/range/rate limits, per-request atomicity, multi-request non-atomicity, and retry limitations documented; body-limit contract normalization remains gated on [#125](https://github.com/GonzaloSecades/nuchi/issues/125) |
| `getSummary`                                                                                                               | `docs/api/summary.md`                            | Inclusive range, previous-period, top-three/Other, daily fill, empty data, and account filtering documented                                                                                                                                                   |

## Compatibility and generation gates

- The validator now rejects duplicate operation IDs and missing summaries.
- CI regenerates Go, TypeScript, and sqlc artifacts and fails on drift.
- Pull requests compare the contract to their base commit and reject removed
  operations/responses/schemas/properties/enum values and newly required
  parameters, request bodies, or schema properties. This focused guard does not
  replace review for semantic changes.
- `GET /api/ready` and `X-Request-ID` are additive. Rollback removes the route
  and header; deploy routing must then use the prior health behavior.

## Endpoint review status

The module guides above serve as the existing endpoint reviews. No Phase 4
change modifies auth error semantics or bulk body limits: those checklist items
link to their Claude-owned gates rather than being reimplemented here.
