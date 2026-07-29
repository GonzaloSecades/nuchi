---
type: "query"
date: "2026-07-29T04:12:53.549237+00:00"
question: "check the graphify outcome for nuchi and tell me how can we use it properly"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Architecture", "schema.ts", "transactions.ts", "use-create-transaction.ts", "InsertTransactionSchema", "generated.gen.go", "newAuthTestEnv()", "createApiError()"]
---

# Q: check the graphify outcome for nuchi and tell me how can we use it properly

## Answer

Expanded from original query via graph vocab: [architecture, feature, route, auth, database, schema, transaction, workflow, health, summary, query, hono]. The Nuchi graph is structurally healthy and useful for symbol-level navigation: 2,733 nodes and 4,724 edges, with no dangling, missing, self-loop, duplicate, or collapsed edges. Concrete path/explain/affected tests successfully connected use-create-transaction.ts, InsertTransactionSchema, transactions.ts, and downstream importers. It is not yet reliable for high-level community navigation: 542 communities include 399 singletons, 1,233 nodes have degree <=1, community labels are visibly stale after incremental updates, and 827 nodes (30.3%) come from generated OpenAPI/sqlc/TypeScript artifacts or Drizzle snapshots. The current manifest hashes match source files, but it retains 10 now-ignored .codex entries; the hook is not installed. Recommended workflow: curate generated artifacts with .graphifyignore, prune with graphify update . --force, run graphify cluster-only . to regenerate communities and labels, optionally install graphify hook install, then prefer exact explain/path/affected commands or query with --context call/import and a small token budget. Verify INFERRED edges against source locations and save useful outcomes to work memory.

## Outcome

- Signal: useful

## Source Nodes

- Architecture
- schema.ts
- transactions.ts
- use-create-transaction.ts
- InsertTransactionSchema
- generated.gen.go
- newAuthTestEnv()
- createApiError()