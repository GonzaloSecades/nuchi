# Investigation: Argentina Open Banking (Sistema de Finanzas Abiertas)

Tracks the real-world status of Argentina's open banking / open finance
initiative as product context for Nuchi. Argentina created the **Sistema de
Finanzas Abiertas (SFA)** by Decree 353/2025 (May 2025), with the BCRA as
application authority; as of July 2026 it is in the infrastructure-definition
phase (technical working groups), with an expected operational launch in
2026–2027.

## Files

- [briefing.md](briefing.md) — scope, research questions, and the refresh
  protocol for this investigation.
- [findings.md](findings.md) — detailed, sourced findings (regulation,
  timeline, actors, technical model).
- [summary.md](summary.md) — executive summary and implications for Nuchi.

## Update flow

This directory is treated as code and follows the same loop as migration
tickets:

1. Changes land via a `claude/...` branch and a PR — never directly on
   `master`.
2. Each refresh updates `findings.md` (append/revise with dates and sources),
   regenerates `summary.md`, and bumps the "Last refreshed" line below.
3. Run `graphify update .` before handing in, so the knowledge graph tracks
   the investigation.
4. Merge only on Gonzalo's explicit approval, per repo rules.

**Last refreshed:** 2026-07-10
