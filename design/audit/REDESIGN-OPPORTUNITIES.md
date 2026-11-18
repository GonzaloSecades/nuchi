# Nuchi Redesign Opportunity Map

**Document purpose:** Prioritized opportunity map derived from `DESIGN-AUDIT.md`  
**Scope:** Current-implementation improvement opportunities, not a redesign specification

## 1. How to read this map

This document converts the audit findings into coherent workstreams. It does not prescribe replacement components, visual concepts, mockups, implementation code, or timelines. Each opportunity remains directional and references stable audit issue IDs.

Prioritization follows three criteria, in order:

1. User risk: potential for misunderstanding system state, losing access to an action, or misjudging a destructive operation.
2. Cross-screen leverage: the number of routes, workflows, and state types affected.
3. Design-system scalability: the degree to which a shared decision can prevent future drift.

The evidence boundary remains the same as the audit: `design/current/DESIGN.md`, `design/current/UI-MAP.md`, `design/current/COMPONENT-INVENTORY.md`, and `design/current/INCONSISTENCIES.md` only. Validation questions identify what should be tested or confirmed later; they do not imply that testing has already occurred.

## 2. Prioritized workstreams

### Workstream A — Trustworthy system status and recovery

#### Opportunity A1 — Distinguish failed reads from valid empty and zero data

**Opportunity:** Establish a product-wide meaning for successful data, zero data, empty data, unavailable data, and failed data retrieval.

**Audit issues addressed:** UX-001, UX-009

**Why it matters:** Financial summaries and management tables must communicate whether the displayed state is authoritative. The current evidence shows that failures can look like valid zeros, empty charts, “No results,” or blank defaults.

**Directional scope:** Overview summaries and charts; account, category, and transaction lists; edit-data loading; state copy; recovery paths; semantic announcement of failure.

**Dependencies:** Agreement on data-state definitions; identification of which screens can retry, refresh, preserve prior data, or redirect; alignment with existing toast and inline feedback conventions.

**Risks / guardrails:** Do not convert every empty state into an error. Preserve legitimate “No results” and “No data for this period” meanings. Do not imply data loss when the evidence only establishes a read failure.

**Suggested sequence:** First in the overall program; define state meanings before unifying their visual presentation.

**Validation questions:**

- Can users tell whether a zero balance is current and successfully loaded?
- Can users distinguish no matching rows from a failed list request?
- Is a recovery action available and appropriate to the failed scope?
- Do assistive technologies receive the same status distinction?

#### Opportunity A2 — Create a coherent loading, empty, and outcome language

**Opportunity:** Align the semantics and hierarchy of loading, empty, success, and error feedback across full-screen, card, sheet, table, form, and action contexts.

**Audit issues addressed:** UX-009, UX-013, UX-017

**Why it matters:** Current async feedback is present but fragmented. Consistent state language improves orientation, accessible announcement, and perceived stability without forcing every context into one visual treatment.

**Directional scope:** Session and auth bootstrap; list and chart loading; sheet loading; pending buttons; table/chart/CSV empty states; auth outcomes; mutation toasts; stable chart-region occupation.

**Dependencies:** Opportunity A1’s state definitions; product-language rules; accessibility review of status and alert semantics.

**Risks / guardrails:** Preserve context-appropriate differences between a button spinner, a page-level state, and a destructive-action result. Avoid making transient states visually dominant. Do not treat all absence of data as equivalent.

**Suggested sequence:** Immediately after A1; establish semantics before consolidating repeated presentations.

**Validation questions:**

- Does each loading state identify what is loading and at what scope?
- Does changing chart type preserve a stable surrounding layout?
- Do empty states explain whether filtering, missing data, or setup is the cause?
- Are status and alert announcements timely without becoming repetitive?

### Workstream B — Accessible and predictable interaction semantics

#### Opportunity B1 — Make every visible action operable and explicitly named

**Opportunity:** Apply one interaction-accessibility contract to icon controls, clickable table content, row actions, reset actions, and composite controls.

**Audit issues addressed:** UX-002, UX-010

**Why it matters:** Core operations should not depend on pointer-only cell behavior or unlabeled icon recognition. The current Form and Radix foundations demonstrate a stronger standard that local controls should match.

**Directional scope:** Mobile menu trigger; row action triggers; transaction account/category cell actions; date reset; amount sign reversal; focus order; visible and non-visible names; keyboard activation.

**Dependencies:** Definition of the intended primary action for each row/cell; accessible-name content decisions; verification of focus behavior within tables and overlays.

**Risks / guardrails:** Do not create multiple competing focus targets for the same action. Preserve efficient pointer interaction while adding equivalent keyboard and assistive-technology behavior. Avoid assigning button semantics to non-actions.

**Suggested sequence:** Parallel with Workstream A and before broader visual-system consolidation.

**Validation questions:**

- Can every documented pointer action be reached and activated by keyboard?
- Does each icon-only control have a meaningful name that describes its action?
- Is the primary row action discoverable without accidental activation?
- Does focus move predictably when an action opens a sheet, menu, or popover?

#### Opportunity B2 — Reinforce domain states without relying on color alone

**Opportunity:** Define how income/expense and categorized/uncategorized meaning is redundantly communicated across controls, tables, summaries, and charts.

**Audit issues addressed:** UX-018, UX-011

**Why it matters:** Color is a useful supporting signal but currently receives uneven reinforcement. Equivalent meaning should remain available to users who cannot reliably distinguish the existing green/rose treatments.

**Directional scope:** Amount entry; transaction badges and cells; chart legends/tooltips; summary values; signs, labels, icons, or position where appropriate.

**Dependencies:** Domain terminology; review of where sign or label already provides sufficient reinforcement; later rendered contrast validation.

**Risks / guardrails:** Do not add redundant visual noise to every occurrence. Preserve rapid scanning. Avoid using inconsistent symbols for the same financial state.

**Suggested sequence:** After the interaction-accessibility contract and alongside semantic color work.

**Validation questions:**

- Is each important state distinguishable without color?
- Do tables, inputs, summaries, and charts use compatible domain language?
- Can users scan positive, negative, and uncategorized states quickly?
- Are visual cues understandable without requiring a legend in routine contexts?

### Workstream C — Safe and unambiguous form actions

#### Opportunity C1 — Clarify create, edit, pending, and destructive modes

**Opportunity:** Establish explicit experience contracts for form mode, action labels, available destructive actions, and complete composite-control disabling.

**Audit issues addressed:** UX-003, UX-006, UX-010, UX-017

**Why it matters:** The edit-transaction flow currently communicates create semantics, and the amount control remains partly operable while pending. Users need to know what object they are changing and whether input is still accepted.

**Directional scope:** Transaction create/edit sheets; account/category create/edit patterns; form titles and submit labels; pending states; delete entry points; composite controls; success wording.

**Dependencies:** Product-language conventions; row-action hierarchy; explicit definition of form modes and pending-state behavior.

**Risks / guardrails:** Preserve domain-specific validation and optional fields. Do not expose destructive actions merely for visual consistency. Avoid duplicating the same destructive action without a clear hierarchy.

**Suggested sequence:** High priority after status semantics; address transaction mode first, then apply the contract across CRUD forms.

**Validation questions:**

- Can users identify whether a form creates or edits before submitting?
- Do pending forms prevent all value-changing interactions consistently?
- Are save, create, cancel, and delete actions named and ordered predictably?
- Does a successful action clearly state what changed?

#### Opportunity C2 — Make destructive confirmation domain-correct

**Opportunity:** Define the information every destructive confirmation must communicate for accounts, categories, transactions, and bulk selections.

**Audit issues addressed:** UX-004, UX-010, UX-017

**Why it matters:** A destructive dialog that names the wrong entity undermines informed consent at the point of highest consequence.

**Directional scope:** Single-row deletion; bulk deletion; object name/type; selected count; consequence language; pending and completion feedback.

**Dependencies:** Confirmed deletion consequences for each domain; product-language rules; decision on which consequences require stronger explanation.

**Risks / guardrails:** Do not invent downstream consequences not established by product behavior. Keep confirmation proportional to risk. Preserve efficient deletion for understood, reversible, or low-impact cases if applicable.

**Suggested sequence:** Immediate correction within the High-risk set, followed by a reusable confirmation-content standard.

**Validation questions:**

- Does the confirmation name the correct entity and quantity?
- Can users understand the consequence before confirming?
- Is the destructive action visually and verbally distinct from cancellation?
- Does post-action feedback identify what was deleted?

### Workstream D — Responsive operational workflows

#### Opportunity D1 — Define narrow-screen information priority for data management

**Opportunity:** Establish which transaction, account, and category information and actions remain primary at each responsive range.

**Audit issues addressed:** UX-005, UX-010

**Why it matters:** Horizontal overflow preserves data but does not resolve prioritization, action discoverability, or row comparison on small screens. Core management tasks need an intentional narrow-screen reading order.

**Directional scope:** Transaction, account, and category lists; row identity; financial values; selection; edit/action access; sorting/filtering context; pagination; bulk actions.

**Dependencies:** Task-priority research or product-owner decisions; row-action hierarchy; accessibility requirements for any alternate presentation.

**Risks / guardrails:** Do not remove information users need to verify financial records. Preserve semantic relationships, selection state, and efficient desktop comparison. Avoid creating separate mobile behavior that diverges functionally from desktop.

**Suggested sequence:** Begin after High-risk action semantics are defined; address Transactions first because it has the greatest column and interaction density.

**Validation questions:**

- Which fields are required to identify and verify a transaction at a glance?
- Can users find row actions without horizontal exploration?
- Can users compare records and maintain row context on narrow screens?
- Are sorting, selection, pagination, and bulk actions still understandable?

#### Opportunity D2 — Unify responsive boundaries and adaptation rules

**Opportunity:** Define one breakpoint contract and decide when the product changes component structure versus adapting layout through styling.

**Audit issues addressed:** UX-007, UX-008, UX-015

**Why it matters:** Inclusive JavaScript media queries overlap with CSS minimum-width rules at 640 px and 1024 px, while React Select carries a separate 767 px rule. A shared contract reduces boundary ambiguity and future responsive drift.

**Directional scope:** Navigation; date filter; header filters; auth split; form-control text; select systems; sheets/dialogs; dashboard grids and padding.

**Dependencies:** Inventory of current breakpoint-dependent behaviors; agreement on structural versus presentational adaptation; hydration and accessibility review for render-time replacement.

**Risks / guardrails:** Do not force every component to use the same adaptation technique when structural change is justified. Preserve 16 px narrow-screen input text where it supports mobile input usability. Test exact boundary widths, not only representative devices.

**Suggested sequence:** Define after D1’s content priorities, then apply to the header and date filter before lower-risk layout conventions.

**Validation questions:**

- At exact boundary widths, is only one layout assumption active?
- Do navigation and date-filter changes preserve state and focus?
- Do all form controls follow the intended text-size transition?
- Are overlay widths appropriate to task complexity across viewport sizes?

### Workstream E — Consistent control and data interaction patterns

#### Opportunity E1 — Establish a shared selection-control contract

**Opportunity:** Align the behavior and presentation expectations of fixed-option and creatable selection controls without erasing their capability differences.

**Audit issues addressed:** UX-008, UX-012

**Why it matters:** Users currently encounter two separate focus, menu, height, theming, responsive-text, and empty-state systems for similar selection tasks.

**Directional scope:** Account/date filters where applicable; chart selectors; CSV mapping; transaction account/category creation; import account selection; labels, errors, disabled states, and menu behavior.

**Dependencies:** Capability matrix for fixed, searchable, and creatable selection; keyboard/accessibility verification; semantic color and focus conventions.

**Risks / guardrails:** Do not reduce the transaction workflow’s ability to create account/category values. Do not assume identical underlying controls are required; consistency should be judged by user-facing behavior and semantics.

**Suggested sequence:** After form-mode and accessibility contracts; before broad control-density normalization.

**Validation questions:**

- Are keyboard navigation, focus indication, disabled behavior, and error messaging predictable across selects?
- Is the ability to create a new value clearly distinguished from selecting an existing one?
- Do menu placement and empty states behave consistently in sheets and dialogs?
- Do text size and control height remain coherent at responsive boundaries?

#### Opportunity E2 — Define one row interaction hierarchy

**Opportunity:** Clarify the relationship among direct cell interaction, primary row editing, contextual menus, and form-level destructive actions.

**Audit issues addressed:** UX-002, UX-010, UX-004

**Why it matters:** Different cells and domains currently open different edit experiences, while row menus and some forms provide overlapping actions. A transferable model improves discoverability and reduces accidental or missed actions.

**Directional scope:** Account, category, and transaction rows; uncategorized transaction behavior; row menus; edit sheets; single and bulk deletion.

**Dependencies:** Accessibility contract; destructive-action contract; mobile content priorities.

**Risks / guardrails:** Preserve useful shortcuts only when their affordance is explicit. Do not make an entire row interactive if it contains nested controls with conflicting behavior. Maintain semantic table integrity.

**Suggested sequence:** Define before revising the narrow-screen table experience so both share the same action model.

**Validation questions:**

- Can users predict which action occurs from a cell, row, or menu?
- Is there one obvious route to edit, with secondary actions clearly subordinate?
- Are direct interactions keyboard accessible and non-conflicting with selection?
- Does the hierarchy remain understandable on narrow screens?

### Workstream F — Scalable visual and content foundations

#### Opportunity F1 — Clarify semantic color ownership and domain exceptions

**Opportunity:** Establish which current semantic roles are authoritative, which direct colors represent legitimate domain meaning, and which values remain explicit inconsistencies pending migration.

**Audit issues addressed:** UX-011, UX-018

**Why it matters:** Current semantic variables do not govern auth colors, header support text, React Select borders, chart palettes, tooltips, or all domain states. Without clear ownership, theme and brand changes can remain partial.

**Directional scope:** Global semantic roles; dashboard brand surfaces; auth text; input borders; domain positive/negative states; chart series and tooltips; inactive dark and unused sidebar token families.

**Dependencies:** Rendered contrast assessment; domain-state cue principles; decision on whether dark mode is an active product requirement or only legacy/incomplete infrastructure.

**Risks / guardrails:** Do not silently collapse near-identical blues or remove intentional income/expense distinctions. Do not present the existing dark variables as a complete supported theme without validating all hard-coded surfaces.

**Suggested sequence:** After state and accessibility semantics; before any broad visual refresh or theme expansion.

**Validation questions:**

- Does each semantic role have a clear product meaning and scope?
- Which domain colors must remain distinct for comprehension?
- Do all states retain sufficient non-color reinforcement?
- If a theme changes, which current surfaces still fail to follow it?

#### Opportunity F2 — Define role-based typography, control density, focus, and surface hierarchy

**Opportunity:** Convert locally assembled type, size, focus, spacing, elevation, and overlay conventions into documented role-level guidance.

**Audit issues addressed:** UX-012, UX-015, UX-013

**Why it matters:** The current application has sound primitive scales but no shared recipes for screen titles, status headings, summary values, data controls, chart text, focus treatment, or overlay geometry.

**Directional scope:** Type roles; line hierarchy; control heights; focus visibility; page/card spacing; overlay width principles; card/elevation roles; visualization-region stability; small-control radius behavior.

**Dependencies:** Responsive boundary contract; semantic color ownership; later rendered review of contrast, alignment, and density.

**Risks / guardrails:** Do not normalize values solely for numerical uniformity. Preserve differences that communicate hierarchy or suit task complexity. Do not infer visual defects that the evidence documents cannot confirm.

**Suggested sequence:** After high-risk behavior and state work; use current repeated values as evidence, not as proof of intentional roles.

**Validation questions:**

- Can users distinguish screen, section, card, status, and data hierarchy consistently?
- Are focus states equally visible across primitive and custom controls?
- Do controls within one form feel related without forcing identical dimensions where capability differs?
- Do overlay and card surfaces communicate containment consistently?

#### Opportunity F3 — Establish a product-language standard

**Opportunity:** Define voice, capitalization, tense, grammar, and domain-noun conventions for actions and system outcomes.

**Audit issues addressed:** UX-017, UX-003, UX-004, UX-009

**Why it matters:** Copy is part of mode, consequence, and feedback clarity. Current defects range from local grammar errors to inconsistent destructive and mutation wording.

**Directional scope:** Form titles; button labels; pending labels; empty states; confirmations; success/error toasts; auth outcomes; domain terminology.

**Dependencies:** State taxonomy; form-mode contract; destructive-action contract; confirmed domain vocabulary.

**Risks / guardrails:** Do not rewrite specific, useful guidance into generic language. Preserve the privacy-protective forgot-password response. Avoid changing product behavior through copy alone.

**Suggested sequence:** Define early enough to support Workstreams A and C; complete the cross-product copy audit after behavioral contracts are settled.

**Validation questions:**

- Do action labels state the object and intended result clearly?
- Are error messages grammatical, actionable, and proportional to the failure?
- Are create/edit/pending/success verbs consistent across domains?
- Does terminology remain stable from table to sheet to confirmation to toast?

### Workstream G — Workflow-level reuse and governance

#### Opportunity G1 — Govern repeated CRUD, auth, loading, and list structures

**Opportunity:** Define reusable workflow patterns above the primitive level while preserving domain-specific validation and language.

**Audit issues addressed:** UX-014, UX-009, UX-017, UX-015

**Why it matters:** Account/category structures, list loaders, auth forms, and spinners are already near copies and have begun to drift. Shared experience governance can reduce divergence without flattening genuine differences.

**Directional scope:** Account/category forms and sheets; column/action conventions; list-page loading frames; auth form rhythm and outcome panels; loading indicators; layout and feedback contracts.

**Dependencies:** State taxonomy; form-mode and content standards; role-based layout guidance; inventory of justified domain differences.

**Risks / guardrails:** Do not create abstraction for its own sake. Preserve route-specific validation, success outcomes, and transaction complexity. Measure success by consistent user behavior, not by minimizing component count.

**Suggested sequence:** Begin after core state and action contracts are defined; use accounts/categories as the first comparison because their current structures are closest.

**Validation questions:**

- Which repeated structures are truly equivalent from the user’s perspective?
- Which differences are domain requirements rather than drift?
- Can a correction to loading, confirmation, or spacing propagate safely across all intended screens?
- Does reuse make content more accurate rather than more generic?

### Workstream H — Analytical-state continuity

#### Opportunity H1 — Define which analytical view choices should persist

**Opportunity:** Establish a rule for persistent, shareable, and ephemeral analytical controls.

**Audit issues addressed:** UX-016

**Why it matters:** Account/date filters persist through URL state, while chart-type choices reset as local state. The difference may be appropriate, but it is not documented as a product rule.

**Directional scope:** Overview account/date context; transaction and category visualization choices; reload, navigation, back/forward behavior, and shared links.

**Dependencies:** Understanding of how users revisit or share dashboard views; privacy considerations for URL state; state-model alignment with existing filters.

**Risks / guardrails:** Do not persist every preference automatically. Avoid cluttering URLs with low-value state. Preserve the current reliable account/date context behavior.

**Suggested sequence:** Later phase, after higher-risk state, accessibility, form, and responsive work.

**Validation questions:**

- Do users expect chart representation to survive reload or navigation?
- Is the selected representation important when sharing a filtered view?
- Would persistence create confusing stale preferences?
- Does browser back/forward behavior remain understandable?

## 3. What should be preserved

The following current-product foundations should remain explicit guardrails for future work:

- The clear separation between authentication and dashboard shells.
- The existing dashboard route order: Overview, Transactions, Accounts, Categories, Settings.
- The recognizable dashboard organization of summary cards, analytical cards, and management-list cards.
- URL-backed account and date filters that preserve each other’s state.
- Form labels, generated descriptions/error relationships, `aria-invalid`, and Zod-backed validation.
- Radix-provided focus management, keyboard behavior, and state attributes for overlays and controls.
- The visually hidden title already used for the mobile navigation sheet.
- Semantic table structure, checked/indeterminate selection, sorting, pagination, and overflow containment.
- Disabled pending controls and inline submit feedback where they are already complete.
- Replacement success states for authentication workflows.
- The privacy-preserving forgot-password response that does not reveal whether an email address exists.
- Confirmation before documented delete operations.
- Explicit mobile-sheet closure after route selection.
- The established income/expense domain distinction, while strengthening redundant non-color cues.
- The current capability to create account or category values from transaction-related select workflows.
- Honest retention of legacy, one-off, and inconsistent values until a deliberate migration decision is made.

## 4. Phased sequence

The phases indicate dependency and priority, not duration.

### Phase 1 — Correct trust and action risks

- A1: Distinguish failed reads from valid empty and zero data.
- B1: Make every visible action operable and explicitly named.
- C1: Clarify create, edit, pending, and destructive modes.
- C2: Make destructive confirmation domain-correct.

**Exit condition:** The product communicates whether data is trustworthy, exposes core actions accessibly, identifies form mode correctly, and describes destructive scope accurately.

### Phase 2 — Stabilize responsive and interaction contracts

- D1: Define narrow-screen information priority for data management.
- E2: Define one row interaction hierarchy.
- D2: Unify responsive boundaries and adaptation rules.
- E1: Establish a shared selection-control contract.

**Exit condition:** Core management workflows remain understandable across narrow and wide layouts, with predictable row and select behavior.

### Phase 3 — Unify cross-product state and language

- A2: Create coherent loading, empty, and outcome language.
- F3: Establish a product-language standard.
- B2: Reinforce domain states without relying on color alone.

**Exit condition:** System state, domain meaning, and action outcomes are communicated consistently across visual and assistive channels.

### Phase 4 — Strengthen scalable foundations

- F1: Clarify semantic color ownership and domain exceptions.
- F2: Define role-based typography, control density, focus, and surface hierarchy.
- G1: Govern repeated CRUD, auth, loading, and list structures.

**Exit condition:** New screens and changes can use documented role and workflow rules without recreating the current drift or erasing justified differences.

### Phase 5 — Refine analytical continuity

- H1: Define which analytical view choices should persist.

**Exit condition:** Persistent and ephemeral analytical state follow an explicit, user-centered rule.

## 5. Program-level guardrails

- Resolve High-severity behavior before broad aesthetic normalization.
- Preserve verified strengths rather than rebuilding them without evidence.
- Keep error, empty, zero, loading, and success states semantically distinct.
- Do not hide current inconsistencies by renaming conflicting values as if they were already unified.
- Validate rendered contrast, hierarchy, overflow, and focus visibility before making claims that the evidence documents cannot support.
- Retain domain-specific meaning when establishing shared patterns.
- Judge reuse by consistent user outcomes, not by abstraction count.
- Treat responsive work as information and action prioritization, not only breakpoint adjustment.
- Keep destructive language specific to the actual entity and consequence.
- Ensure accessibility behavior is equivalent across pointer, keyboard, and assistive-technology use.
