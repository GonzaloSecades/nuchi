---
type: "query"
date: "2026-08-08T21:22:01.110043+00:00"
question: "3 more prs landed. Review them all"
contributor: "graphify"
outcome: "useful"
source_nodes: ["GetSummary", "WithUserTx", "goApiRewrites", "authenticated-fetch.ts", "token-store.ts"]
---

# Q: 3 more prs landed. Review them all

## Answer

Reviewed PRs #75, #76, and #77 at heads 780a6ce, 0c50b44, and 1096ad1. PR #75 has READ COMMITTED snapshot inconsistency and valid-value int64 overflow risks. PR #76 is broadly sound but should reject URL userinfo. PR #77 has blocking fresh-session refresh, Request header preservation, Request body retry, and server-global token-store issues, plus apiErrorCode mismatch.

## Outcome

- Signal: useful

## Source Nodes

- GetSummary
- WithUserTx
- goApiRewrites
- authenticated-fetch.ts
- token-store.ts