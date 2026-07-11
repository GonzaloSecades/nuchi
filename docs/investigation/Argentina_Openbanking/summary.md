# Summary — Argentina Open Banking, as of 2026-07-10

## TL;DR

The promise was kept on paper, not yet in APIs. The Milei administration
created Argentina's first open finance framework — the **Sistema de Finanzas
Abiertas (SFA)** — via **Decree 353/2025** (effective 2025-05-23). The BCRA is
the application authority and spent late 2025 promoting the system (Bausili in
October, Werning at the Fintech Forum in November) as a credit-revival tool:
voluntary, express-consent sharing of bank, wallet, and eventually state
(ANSES/ARCA) data. Its 2026 plan is to run **technical working groups to
define the infrastructure**. As of July 2026 there are **no published API
standards, no implementing BCRA Comunicación, and no official launch date**;
sector estimates say operational in **2026–2027**.

## What this means for Nuchi

1. **No integration target exists yet.** Nothing to build against; manual
   entry and file import remain the ingestion story for now.
2. **The architecture bets age well.** A consent-first, token-scoped Go API
   with RLS matches the SFA's express-consent model; if/when standards land,
   consented ingestion becomes an additive feature (new data source), not a
   redesign.
3. **Regulatory perimeter is the open question.** The decree names
   BCRA-registered financial entities as participants. Whether a personal
   finance app can plug in directly, must partner with a registered entity,
   or falls outside the perimeter won't be known until the BCRA issues
   implementing regulation.
4. **Milliunit/ARS invariants are unaffected** — the SFA is about data
   access, not payments or currency.

## Watch list (triggers a refresh of this investigation)

- First BCRA **Comunicación "A"** implementing the SFA (standards, registry,
  or calendar).
- Outputs of the 2026 technical working groups.
- BCRA *Objetivos y Planes 2027* (expected late December 2026).
- Any pilot/sandbox announcement or fintech-chamber standardization work.

Full detail and sources: [findings.md](findings.md). Scope and refresh
protocol: [briefing.md](briefing.md).
