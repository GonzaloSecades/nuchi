---
type: "query"
date: "2026-08-08T12:24:07.201967+00:00"
question: "check the tickets going foward, what we shall continue with after this pr merges ?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Go Backend Replacement Spec", "Summary", "summary.ts", "use-get-summary.ts", "Legacy Code"]
---

# Q: check the tickets going foward, what we shall continue with after this pr merges ?

## Answer

Expanded from original query via graph vocab: [migration, summary, backend, replacement, frontend, client, hooks, auth, cleanup, ticket, issue, legacy]. Live GitHub state and the Go Backend Replacement Spec agree that PR 74 closes issue 47 and issue 48, Implement summary API parity, is the only next sequential ticket. Issue 48 is high-risk attended work. It is implementation-ready because summary.sql and sqlc generated queries, the shared date-range parser, OpenAPI GetSummary types, legacy summary.ts, frontend use-get-summary hook, and parity fixtures already exist. After 48, continue strictly with 30 Next same-origin rewrite, 49 generated client wiring, 50 TanStack hook migration, 51 custom auth UI, and 27 legacy teardown. Post-migration issues 66 production-shaped environment and 73 bi-format documentation wait until 27 closes.

## Outcome

- Signal: useful

## Source Nodes

- Go Backend Replacement Spec
- Summary
- summary.ts
- use-get-summary.ts
- Legacy Code