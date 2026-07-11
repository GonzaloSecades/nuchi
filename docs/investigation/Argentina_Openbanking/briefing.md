# Briefing — Argentina Open Banking Investigation

**Date opened:** 2026-07-10
**Owner:** main session (PM). Research may be delegated to the `researcher`
agent on refreshes.
**Status:** active — regime under construction, re-check quarterly or on news.

## Why Nuchi cares

Nuchi is a personal finance app for an Argentina-based user. Today all
transaction data is entered or imported manually. If the Sistema de Finanzas
Abiertas (SFA) becomes operational, consented programmatic access to bank,
wallet, and state (ANSES/ARCA) data could replace manual import as the primary
ingestion path — a major product and architecture consideration for the Go
backend (ingestion endpoints, consent model, token-scoped data access).

## Research questions

1. What exactly did the current (Milei) administration promise and deliver on
   open banking? (Answered: Decree 353/2025 — see findings.)
2. Who regulates it, and under what model (mandatory vs. voluntary,
   consent-based)? (Answered: BCRA, voluntary + express consent.)
3. What is the implementation status and expected timeline? (In progress:
   technical groups during 2026; launch estimated 2026–2027.)
4. Are there published technical/API standards Nuchi could build against?
   (Open: none published as of 2026-07-10.)
5. What is the regulatory perimeter — would a personal finance app need to
   register with the BCRA to receive data? (Open: undefined until BCRA issues
   implementing regulation.)

## What to check on each refresh

- BCRA Comunicaciones "A" index (bcra.gob.ar) for anything implementing the
  SFA — none existed as of the June 2026 indexes.
- BCRA news/press page and the next Objectives & Plans document (published
  late December each year).
- Boletín Oficial for decrees/resolutions citing Decree 353/2025.
- Cámara Argentina Fintech and major outlets (Infobae, La Nación, Ámbito,
  El Cronista, iProUP) for technical-group outputs and pilot announcements.
- Whether API standards, sandbox, or an entity registry have been published.

## Refresh protocol

Same flow as a code update: branch → update `findings.md` (dated entries,
sources) → regenerate `summary.md` → bump README "Last refreshed" →
`graphify update .` → PR → human merge review.
