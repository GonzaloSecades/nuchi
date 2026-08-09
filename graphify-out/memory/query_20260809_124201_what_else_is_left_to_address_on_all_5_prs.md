---
type: "query"
date: "2026-08-09T12:42:01.084147+00:00"
question: "what else is left to address on all 5 prs ?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Authentication and sessions,Summary,Transaction,Request,Contract,Pull request gates,frontend hooks,generated client/OpenAPI"]
---

# Q: what else is left to address on all 5 prs ?

## Answer

All five open PRs have green checks and zero unresolved review threads, but the batch is not cutover-ready. PR 80 must be held until issue 50 is complete: PR 78 covers accounts, PR 81 covers transactions and summary, but all six category hooks still use the Hono client. With PR 80 merged and USE_GO_API false, the custom auth pages have no matching Hono auth endpoints; with the flag true before categories migrate, category requests remain unauthenticated. Before merge, fix PR 80 verify-email stale-promise cancellation; remove PR 81 runtime Hono type import from transaction columns and update its stale description; make PR 82 require the expected 401 from the Go readiness probe. PRs 78 and 79 otherwise only need to leave draft status. Future/post-migration work includes the account detail query-key mismatch, the remaining issue 73 docs/index, CI and route-level coverage for the parity harness, and broadening parity beyond date serialization.

## Outcome

- Signal: useful

## Source Nodes

- Authentication and sessions,Summary,Transaction,Request,Contract,Pull request gates,frontend hooks,generated client/OpenAPI