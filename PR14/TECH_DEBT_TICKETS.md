# Technical Debt Tickets

- Source: `PR14/PR_SUMMARY.md`
- Ticket count: 8 (<=10)

## Priority Order

## P0 — Add hard safety gates to destructive seed workflow

- Problem: The seed workflow currently deletes core financial tables before insert and does not enforce environment restrictions.
- Evidence (quote): "Script performs destructive deletes with no environment gate." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Accidental execution against shared/prod data can cause immediate data loss.
- Scope / Proposed fix:
  - Require `NODE_ENV=development` and explicit opt-in flag to run seed.
  - Abort with non-zero exit when required guard conditions fail.
  - Log explicit warning and required flags on guard failure.
- Acceptance criteria:
  - [ ] Seed command refuses to run outside approved environment/flags.
  - [ ] Seed command runs successfully only when guards are satisfied.
- Effort: S
- Owner suggestion: Backend

## P1 — Enforce strict date validation in summary endpoint

- Problem: Summary date parsing currently accepts optional strings without guaranteed parse validity checks.
- Evidence (quote): "Optional string validation does not enforce strict parse success." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Invalid boundaries can produce incorrect windows and unstable analytics.
- Scope / Proposed fix:
  - Validate `from` and `to` using strict `yyyy-MM-dd` parsing/guards.
  - Return `400` with deterministic error message for invalid values.
  - Add endpoint tests for malformed and boundary date inputs.
- Acceptance criteria:
  - [ ] Invalid date queries return `400` with stable error payload.
  - [ ] Valid date windows continue to return correct summary metrics.
- Effort: S
- Owner suggestion: Backend

## P1 — Type tooltip payload contracts for chart components

- Problem: Chart tooltips currently use `any` payload types.
- Evidence (quote): "`any` payload typing in tooltip components." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Weak typing increases runtime failure risk during chart/data contract changes.
- Scope / Proposed fix:
  - Replace `any` with typed Recharts tooltip payload interfaces.
  - Add safe guards for missing/empty payload entries.
  - Add component tests for tooltip rendering with expected/empty payloads.
- Acceptance criteria:
  - [ ] Tooltip components compile with no `any` payload typing.
  - [ ] Tooltips render safely with valid and empty payload scenarios.
- Effort: S
- Owner suggestion: Frontend

## P1 — Remove debug summary route from production module

- Problem: Non-product debug endpoint remains mounted in summary route file.
- Evidence (quote): "`/asd` path is non-product behavior in live router." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Extra undocumented surface area increases maintenance and audit overhead.
- Scope / Proposed fix:
  - Remove `.get('/asd', ...)` from summary module.
  - Add route-surface test to assert only supported summary paths.
  - Update API docs to reflect final route shape.
- Acceptance criteria:
  - [ ] `/api/summary/asd` is not exposed.
  - [ ] Summary routes match documented contract.
- Effort: S
- Owner suggestion: Backend

## P2 — Optimize `fillMissingDays` lookup strategy

- Problem: Missing-day fill currently uses repeated linear search over active day entries.
- Evidence (quote): "O(n*m) lookup strategy." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Larger windows can add avoidable CPU cost in request processing.
- Scope / Proposed fix:
  - Build a date-keyed map of active days before interval traversal.
  - Resolve each day via O(1) lookup.
  - Preserve output ordering and shape.
- Acceptance criteria:
  - [ ] `fillMissingDays` scales linearly with interval size.
  - [ ] Output is identical to current implementation for equivalent inputs.
- Effort: S
- Owner suggestion: Backend

## P2 — Add dependency governance gate for visualization stack

- Problem: Newly added chart dependencies increase compatibility/security maintenance burden without a formal gate.
- Evidence (quote): "Dependency governance for newly added chart stack is not formalized." (`Technical Debt Assessment` → `Introduced Technical Debt`)
- Why it matters: Library drift or vulnerable upgrades can destabilize production dashboards.
- Scope / Proposed fix:
  - Add CI dependency audit/check step for frontend deps.
  - Define acceptable risk thresholds and merge-blocking policy.
  - Add periodic dependency review cadence.
- Acceptance criteria:
  - [ ] CI reports and enforces dependency health checks.
  - [ ] Dependency policy and ownership are documented.
- Effort: M
- Owner suggestion: DevOps

## P2 — Add summary contract and visualization correctness tests

- Problem: Analytics output now drives multiple visualizations but lacks explicit contract regression coverage.
- Evidence (quote): "Add tests for summary amount/percentage semantics." (`Technical Debt Assessment` → `Recommended Next Steps`)
- Why it matters: Changes to summary payload semantics can silently break legends/cards/tooltips.
- Scope / Proposed fix:
  - Add endpoint integration tests for totals, category rollups, and day series.
  - Add hook tests for conversion and cache behavior.
  - Add chart component tests for legend ordering and displayed values.
- Acceptance criteria:
  - [ ] Contract tests detect regressions in summary payload semantics.
  - [ ] Visualization tests verify legend ordering and formatted values.
- Effort: M
- Owner suggestion: Full-stack

## P3 — Track remediation ownership explicitly per ticket in repository docs

- Problem: Owner requested clearer remediation accountability across tech debt items.
- Evidence (quote): "Owner requested debt ownership mapping to make remediation accountability explicit." (`Owner Additions`)
- Why it matters: Explicit ownership reduces ticket drift and follow-up delays.
- Scope / Proposed fix:
  - Add owner/team field to debt ticket template in docs.
  - Map each open PR14 debt ticket to a responsible role/team.
  - Review ownership mapping during PR checklist.
- Acceptance criteria:
  - [ ] All active PR14 debt tickets include an explicit owner mapping.
  - [ ] Ownership check is part of PR review checklist.
- Effort: S
- Owner suggestion: Unknown
