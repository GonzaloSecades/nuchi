---
type: "query"
date: "2026-07-29T04:37:40.405569+00:00"
question: "Should Nuchi create shared Claude and Codex skills around Graphify and CodeGraph after the Go migration?"
contributor: "graphify"
source_nodes: ["Architecture", "Delivery Roadmap and Quality Gates", "API Contract and Documentation Standard", "Nuchi Codex Guide", "PR Review Cycle", "TestFinanceRLS", "migration_upgrade_test.go"]
---

# Q: Should Nuchi create shared Claude and Codex skills around Graphify and CodeGraph after the Go migration?

## Answer

Yes, but as a small quality-gated workflow layer rather than one autonomous co-engineer persona. Keep AGENTS.md as the concise shared project contract and import it from CLAUDE.md. Use Graphify for semantic architecture, decisions, documentation, and historical context; use CodeGraph for live symbol navigation, callers, impact, and affected tests. Start with three focused skills: orient, implement/change-impact, and review/verify. Route to one graph first and use the other only when the task crosses semantic and structural concerns. Treat both graphs as disposable indexes, always verify critical auth, ownership, money, migration, and OpenAPI claims against source and tests. Pilot with baseline versus Graphify-only versus CodeGraph-only versus combined evals before automating hooks or creating a specialized orchestrator agent.

## Source Nodes

- Architecture
- Delivery Roadmap and Quality Gates
- API Contract and Documentation Standard
- Nuchi Codex Guide
- PR Review Cycle
- TestFinanceRLS
- migration_upgrade_test.go