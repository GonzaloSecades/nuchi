# Nuchi Current Application UX/UI Design Audit

**Audit status:** Formal evidence-based audit  
**Evidence date:** 2026-09-21 working-tree reconstruction  
**Base commit identified by the evidence:** `f177ade4d2ab3c754ff5e79ba45bfee128b0ddb1`

## 1. Evidence boundary and limitations

This audit evaluates the current Nuchi application exclusively through the following supplied documents:

- `design/current/DESIGN.md`
- `design/current/UI-MAP.md`
- `design/current/COMPONENT-INVENTORY.md`
- `design/current/INCONSISTENCIES.md`

No application source, rendered interface, screenshots, runtime behavior, token files, external references, or additional project files were inspected. The evidence is a source reconstruction of implemented structures and states, not a browser-based usability or visual-quality evaluation.

Accordingly:

- “Verified current behavior” means behavior explicitly documented in one or more of the four evidence documents.
- “User impact” is an audit judgment derived from that behavior. It is not a measured usability outcome or a report of user testing.
- Visual qualities that require rendering—such as exact contrast ratios, alignment quality, clipping, animation smoothness, and perceived polish—cannot be confirmed.
- Responsive findings describe documented branching and layout behavior; they do not establish how the interface renders on physical devices.
- The audit does not normalize contradictory values into a cleaner system. Documented inconsistencies remain explicit findings.
- No Critical issue is assigned because the supplied evidence does not establish a system-wide blocker, unrecoverable loss, or confirmed severe safety failure.

## 2. Executive audit summary

Nuchi has a credible structural foundation: a clear auth/dashboard split, semantic form relationships, Radix-backed overlay behavior, reusable primitives, URL-backed global filters, and repeatable CRUD/table workflows. The current product is not an unstructured interface. Its strongest patterns are concentrated in form validation, overlay behavior, dashboard card organization, and the shared accounts/categories/transactions management model.

The principal weakness is that this foundation is not applied uniformly at the points where users need the clearest state and interaction semantics. Dashboard read failures can look like legitimate zero or empty data. Some visually interactive table cells and icon-only controls are not represented as keyboard-accessible or explicitly named controls. The edit-transaction flow presents create-state language, and generic destructive confirmation text can name the wrong domain. These are the highest-risk findings because they can change the user’s interpretation of system state or action consequences.

The secondary weakness is systemic drift. Two select implementations, multiple loading and empty-state treatments, locally assembled typography, hard-coded color families, copied CRUD structures, inconsistent responsive boundaries, and separate overlay/layout conventions weaken interaction consistency and make the product harder to scale safely. The evidence does not support describing these differences as a coherent intentional design system.

Mobile behavior is adaptive at the shell and control level, but core data-management tables retain desktop column structures and depend on horizontal scrolling. This creates a material information-density and discoverability concern on narrow screens. Perceived performance is also affected by uneven loading semantics and visualization heights, although rendered performance itself was not measured.

## 3. Severity overview

| Severity | Count | Audit interpretation |
| --- | ---: | --- |
| Critical | 0 | No evidence-supported system-wide blocker or severe irreversible outcome. |
| High | 5 | Material risk to task comprehension, accessibility, destructive-action confidence, or core mobile workflows. |
| Medium | 11 | Significant inconsistency or scalability weakness that can impair clarity, feedback, accessibility, or cross-screen coherence. |
| Low | 2 | Localized quality or persistence issues with limited direct task risk. |

The five High findings should be treated as the primary risk set. Medium findings are not cosmetic by default; several have broad design-system leverage and should be addressed through shared rules rather than isolated screen corrections.

## 4. Positive foundations to preserve

### PF-01 — Clear product-level shell and route structure

The auth and dashboard areas have distinct, documented shells. Dashboard route order is stable—Overview, Transactions, Accounts, Categories, Settings—and the dashboard uses a consistent header, overlapping content-card structure, and session guard. This gives the product a recognizable information architecture. Evidence: `UI-MAP.md`, “Application shells,” “Navigation,” and the route map.

### PF-02 — Strong form semantics and validation foundation

React Hook Form and Zod provide explicit validation across authentication and CRUD flows. The Form primitive connects labels, descriptions, errors, `aria-invalid`, and `aria-describedby` through generated IDs. Evidence: `DESIGN.md`, sections “Form behavior” and “Accessibility behavior”; `UI-MAP.md`, “Forms and validation.”

### PF-03 — Mature overlay interaction primitives

Dialogs, sheets, dropdowns, popovers, selects, checkboxes, and tooltips are built on Radix primitives with documented keyboard, focus, and state behavior. The mobile navigation Sheet includes a visually hidden title. Evidence: `DESIGN.md`, “Component states” and “Accessibility behavior”; `COMPONENT-INVENTORY.md`, “Overlays.”

### PF-04 — Repeatable management workflows

Accounts, categories, and transactions share recognizable table, selection, sorting, pagination, row-action, confirmation, and sheet-form patterns. Although some structures are copied rather than composed, the resulting task model is broadly repeatable. Evidence: `DESIGN.md`, “Design principles currently evident” and “Component taxonomy”; `UI-MAP.md`, route and table descriptions.

### PF-05 — Persistent analytical context for global filters

Account and date filters write to URL search parameters and preserve each other when changed. This is a sound foundation for navigable analytical state. Evidence: `DESIGN.md`, “Navigation patterns”; `UI-MAP.md`, “Filters.”

### PF-06 — Explicit handling of important authentication outcomes

Authentication flows distinguish missing tokens, verification, failure, success, pending submission, and replacement success panels. Forgot-password behavior intentionally avoids revealing whether an address exists. Evidence: `UI-MAP.md`, authentication route descriptions and “Forms and validation.”

### PF-07 — Semantic table structure and selection states

Data tables retain semantic table elements, checked/indeterminate selection states, sorting, pagination, and a horizontal overflow wrapper. This is a sound base even though the narrow-screen presentation remains unresolved. Evidence: `DESIGN.md`, “Accessibility behavior”; `COMPONENT-INVENTORY.md`, `Table`, `Checkbox`, and `DataTable` entries.

## 5. Audit findings

### UX-001 — Dashboard read failures are not distinguishable from valid empty or zero states

**Severity:** High

**Problem:** The dashboard does not provide a reliable visual or semantic distinction between a failed data request and a successful request returning no data.

**Evidence:** `DESIGN.md`, “Feedback patterns,” documents that dashboard query errors have no dedicated visual branch and may appear as zeros, empty charts, “No results,” or blank edit defaults. `UI-MAP.md` documents the same behavior on `/`, `/transactions`, `/accounts`, and `/categories`. `INCONSISTENCIES.md`, “Dashboard query errors are visually indistinguishable from valid emptiness,” confirms the difference is absent from the current implementation.

**Affected screens/components:** Overview `/`; Transactions `/transactions`; Accounts `/accounts`; Categories `/categories`; summary cards; chart cards; `DataTable`; edit-sheet data loading.

**User impact:** Audit judgment: users may interpret unavailable or failed data as a truthful account balance, a valid empty history, or an empty entity record. This can reduce confidence in financial information and obscure recovery actions.

**Design-system impact:** Error, empty, and success-with-zero states do not have a shared semantic contract. Any new data-driven screen can reproduce the same ambiguity.

**Suggested direction:** Establish a distinct read-error state family with clear status, scope, and recovery guidance, while retaining separate empty and zero-data meanings.

### UX-002 — Several interactive controls are inaccessible or undiscoverable to keyboard and assistive-technology users

**Severity:** High

**Problem:** Some visible interactions are not exposed with equivalent keyboard semantics or explicit accessible names.

**Evidence:** `DESIGN.md`, “Accessibility behavior,” identifies transaction account/category cells implemented as clickable `div` elements without button semantics, tab stops, or keyboard handlers. It also identifies unnamed icon-only controls for mobile navigation, row action menus, date reset, and amount sign reversal. `INCONSISTENCIES.md`, “Accessibility inconsistencies,” documents the same call sites. `COMPONENT-INVENTORY.md` identifies the affected `Navigation`, row actions, `DateFilter`, and `AmountInput` components.

**Affected screens/components:** Mobile dashboard navigation; account/category/transaction row menus; transaction account/category table cells; date-filter reset control; `AmountInput` sign-reversal control.

**User impact:** Audit judgment: keyboard and screen-reader users may be unable to discover, identify, or operate actions that are available through pointer interaction.

**Design-system impact:** The accessible semantics provided by Form and Radix primitives are not consistently extended to locally composed controls. The system lacks a dependable rule for interactive-cell and icon-only-control semantics.

**Suggested direction:** Require every interactive surface to expose an explicit role, name, keyboard path, focus state, and equivalent action semantics, including interactions embedded in data cells.

### UX-003 — The edit-transaction flow presents create-state semantics

**Severity:** High

**Problem:** The transaction edit sheet does not supply the identifier that the shared form uses to select edit behavior.

**Evidence:** `DESIGN.md`, “Component states,” documents that the edit transaction sheet omits the transaction `id`, resulting in create wording and a hidden form-level delete action. `COMPONENT-INVENTORY.md`, `TransactionForm` and transaction-sheet entries, confirms the same behavior. `INCONSISTENCIES.md`, “Edit transaction form receives create-state semantics,” states that the form shows “Create Transaction” rather than “Save changes,” while row deletion remains available.

**Affected screens/components:** Transaction edit sheet; `TransactionForm`; transaction row actions.

**User impact:** Audit judgment: users may be uncertain whether they are updating an existing transaction or creating another one, increasing the risk of hesitation, abandonment, or an unintended duplicate mental model.

**Design-system impact:** The shared form’s mode contract is not reliably enforced by its container. Similar create/edit abstractions can drift if state is inferred from optional data without an explicit mode guarantee.

**Suggested direction:** Make create and edit modes unambiguous at the experience-contract level, with consistent language and action availability across entry points.

### UX-004 — Generic bulk deletion can present the wrong domain and consequence

**Severity:** High

**Problem:** A reusable bulk-delete confirmation uses account-specific copy when invoked for categories or transactions.

**Evidence:** `DESIGN.md`, “Component states,” documents account-specific confirmation text in the generic bulk-delete flow. `INCONSISTENCIES.md`, “Bulk-delete confirmation is domain-specific in a generic component,” states that categories and transactions can display copy about deleting “accounts and their account data.” `COMPONENT-INVENTORY.md` confirms `DataTable` is shared by all three domains.

**Affected screens/components:** `DataTable` bulk deletion on `/accounts`, `/categories`, and `/transactions`; generic confirmation dialog.

**User impact:** Audit judgment: incorrect confirmation language can prevent users from accurately judging the scope of a destructive action. It may cause unnecessary cancellation or misplaced confidence.

**Design-system impact:** The reusable destructive-action pattern does not require domain-correct object, count, consequence, and recovery language.

**Suggested direction:** Define a domain-aware destructive confirmation content contract that communicates the actual selected objects and consequences without assuming one entity type.

### UX-005 — Core data tables retain desktop density on narrow screens

**Severity:** High

**Problem:** The responsive strategy preserves all desktop table columns and relies on horizontal scrolling rather than prioritizing information for narrow screens.

**Evidence:** `DESIGN.md`, “Layout and grid system” and “Responsive behavior,” documents desktop columns with horizontal overflow and no mobile row/card alternative. `UI-MAP.md`, “Tables and data display,” confirms this behavior for accounts, categories, and transactions. `INCONSISTENCIES.md`, “Tables have overflow but no alternate mobile presentation,” notes that headers adapt while rows do not.

**Affected screens/components:** `/transactions`, `/accounts`, `/categories`; `DataTable`; table actions, selection, sorting, and pagination.

**User impact:** Audit judgment: narrow-screen users may have difficulty comparing rows, discovering off-screen fields or row actions, and maintaining context while scrolling horizontally. The risk is greatest in the transaction table because it carries the most columns and edit entry points.

**Design-system impact:** The product lacks a shared responsive information-priority model for operational data, limiting scalability as additional fields or row actions are introduced.

**Suggested direction:** Define responsive content priority and action discoverability rules for tabular workflows, preserving essential context and task completion without treating overflow as the only adaptation.

### UX-006 — Pending transaction forms do not fully prevent amount changes

**Severity:** Medium

**Problem:** The amount field becomes disabled during pending submission, but the adjacent sign-reversal control remains operable.

**Evidence:** `DESIGN.md`, “Component states,” documents that disabled state reaches the currency field but not the sign-reversal button. `INCONSISTENCIES.md`, “Disabled amount input is only partially disabled,” confirms the control remains operable during a pending form. `COMPONENT-INVENTORY.md` identifies the coupled field/button structure in `AmountInput`.

**Affected screens/components:** New and edit transaction sheets; `TransactionForm`; `AmountInput`.

**User impact:** Audit judgment: the form communicates that submission is locked while leaving a value-changing action active, creating uncertainty about which value is being submitted and whether the late interaction took effect.

**Design-system impact:** Composite controls do not share an atomic disabled-state contract across all interactive subparts.

**Suggested direction:** Treat disabled and pending behavior as a state of the complete composite control, with all value-changing actions following the same interaction rule.

### UX-007 — Responsive boundaries can apply mobile and larger-layout rules simultaneously

**Severity:** Medium

**Problem:** JavaScript component replacement and CSS breakpoints use overlapping inclusive boundaries at 640 px and 1024 px.

**Evidence:** `DESIGN.md`, “Responsive behavior,” documents `max-width: 640px` against `sm: 640px`, and `max-width: 1024px` against `lg: 1024px`. `UI-MAP.md`, “Responsive summary,” describes the affected DateFilter and Navigation branches. `INCONSISTENCIES.md`, “JavaScript and CSS breakpoint boundaries overlap,” confirms mobile component branches can coexist with larger-breakpoint CSS at the exact boundary.

**Affected screens/components:** `Navigation`, dashboard header, `DateFilter`, header filters, logo/desktop navigation visibility.

**User impact:** Audit judgment: users at exact boundary widths may encounter mixed layout assumptions, unstable navigation/header composition, or inconsistent date-filter presentation. Actual visual breakage was not verified.

**Design-system impact:** Responsive behavior does not use one shared boundary contract, and visually related decisions are split between render-time and CSS-time adaptation.

**Suggested direction:** Establish one documented responsive boundary model and a consistent rule for when structural replacement is warranted versus CSS adaptation.

### UX-008 — Two select systems create inconsistent form interaction and appearance

**Severity:** Medium

**Problem:** Radix Select and creatable React Select implement parallel control, menu, focus, height, theming, and responsive-text systems.

**Evidence:** `DESIGN.md`, “Component taxonomy,” classifies select controls as inconsistent. `COMPONENT-INVENTORY.md` documents Radix Select for filters/chart/CSV mapping and React Select for transaction account/category fields and the import account dialog. `INCONSISTENCIES.md`, “Two unrelated select systems,” records separate borders, DOM/state models, heights, focus behavior, empty behavior, and dark-mode behavior.

**Affected screens/components:** Dashboard account filter; chart type selectors; CSV mapping; transaction account/category fields; import account dialog.

**User impact:** Audit judgment: users may encounter different keyboard, focus, menu, creation, and error expectations for controls that appear to perform closely related selection tasks.

**Design-system impact:** A core input role has no unified behavioral or visual contract, increasing accessibility, theming, and maintenance risk.

**Suggested direction:** Define a shared select experience contract while preserving the distinct capability required for creatable values.

### UX-009 — Loading, empty, and error feedback lack a coherent state model

**Severity:** Medium

**Problem:** Async and no-data states use unrelated visual treatments, semantic announcements, copy styles, and recovery patterns.

**Evidence:** `DESIGN.md`, “Feedback patterns,” documents full-viewport spinners, bounded spinners, sheet overlays, button spinners, skeletons, variable empty states, and per-feature toast wording. It also notes that only sign-in restoration has `role="status"` and an accessible label among most spinner uses. `INCONSISTENCIES.md` documents repeated loaders, varied error surfaces, and drifting empty-state patterns. `UI-MAP.md`, “State map,” lists the screen-level variants.

**Affected screens/components:** Authentication bootstrap; `SessionGuard`; list-page loaders; summary cards; charts; edit sheets; submit buttons; table empty rows; chart empty panels; CSV alerts; auth outcome panels; Sonner toasts.

**User impact:** Audit judgment: users may have difficulty understanding whether the system is loading, empty, failed, or complete, and assistive-technology users may not receive equivalent status feedback across workflows.

**Design-system impact:** Status meaning is tied to individual screens instead of shared state semantics. New features are likely to add another local treatment.

**Suggested direction:** Define a small state taxonomy with consistent semantic announcements, message hierarchy, scope, recovery expectations, and appropriate presentation by context.

### UX-010 — Editing and row-action entry points are not consistently discoverable

**Severity:** Medium

**Problem:** Related editing actions are distributed across clickable names/cells, uncategorized-category behavior, row menus, and optional form-level deletion.

**Evidence:** `DESIGN.md`, “Navigation patterns,” documents varying table edit entry points. `UI-MAP.md`, “Tables and data display,” states that account/category cells are clickable `div` elements and uncategorized transaction cells open the transaction edit sheet. `INCONSISTENCIES.md`, “Interaction-pattern inconsistencies,” records split entry points and multiple delete locations.

**Affected screens/components:** Account, category, and transaction tables; row action menus; clickable transaction cells; edit sheets; in-form delete actions.

**User impact:** Audit judgment: users must learn screen- and cell-specific behavior instead of transferring one predictable row-interaction model. Hidden click targets can be missed, while duplicated delete entry points can complicate consequence recognition.

**Design-system impact:** Data-row interaction lacks a shared hierarchy for primary row action, contextual actions, and destructive actions.

**Suggested direction:** Establish a predictable interaction hierarchy for row inspection/editing and secondary actions, with equivalent semantics across domains.

### UX-011 — Semantic color infrastructure does not control the complete product

**Severity:** Medium

**Problem:** Semantic variables coexist with direct Tailwind palettes, hard-coded auth and header colors, inline React Select borders, hard-coded chart palettes, and white chart tooltips.

**Evidence:** `DESIGN.md`, “Color system,” documents direct blues, emerald/rose/red/slate utilities, auth values `#2E2A47` and `#7E8CA0`, header text `#89b6fd`, React Select border `#e2e8f0`, and multiple chart hex values disconnected from declared chart tokens. `INCONSISTENCIES.md`, “Color-system drift,” confirms these systems do not update together. `COMPONENT-INVENTORY.md` locates the affected components.

**Affected screens/components:** Dashboard header and navigation; auth headings and copy; React Select; badges and amount states; charts and tooltips; primitive controls; inactive dark token set.

**User impact:** Audit judgment: users may experience uneven visual emphasis and state communication across screens. A theme or brand change could produce partial, conflicting results. Rendered contrast was not measured.

**Design-system impact:** The documented semantic layer is not a reliable single source of truth. Declared chart and dark tokens are partly disconnected, and inactive/legacy tokens can imply support that the current product does not provide.

**Suggested direction:** Clarify which semantic roles are authoritative, explicitly retain justified domain colors, and treat conflicting current values as migration decisions rather than silently merging them.

### UX-012 — Typography, control sizing, and focus treatment do not form one hierarchy

**Severity:** Medium

**Problem:** Screen, card, auth, status, summary, overlay, chart, and tooltip typography is assembled locally; form controls also split between 36 px and 40 px heights and multiple focus-ring rules.

**Evidence:** `DESIGN.md`, “Typography system” and “Form behavior,” documents local type recipes, responsive 16-to-14 px input changes, 36/40 px height differences, and divergent focus treatments. `INCONSISTENCIES.md`, “Typography drift” and “Spacing, size, radius, and elevation drift,” records the same variations. `COMPONENT-INVENTORY.md` identifies the primitives and custom controls involved.

**Affected screens/components:** Page/card/auth/status headings; summary values; overlay titles; chart labels/tooltips; Input; Textarea; Button; AmountInput; Radix Select; React Select; header controls.

**User impact:** Audit judgment: hierarchy and control affordance may feel less predictable across routes, and differing focus treatments may make keyboard location easier to identify in some controls than others. Exact rendered hierarchy and contrast were not verified.

**Design-system impact:** There is no named typographic-role or control-state system capable of governing new screens consistently.

**Suggested direction:** Define role-based hierarchy, control-density, and focus principles based on current product needs while preserving legitimate differences in content and interaction complexity.

### UX-013 — Chart switching can change layout occupation and weaken perceived stability

**Severity:** Medium

**Problem:** Chart variants use different plot-height conventions within the same visualization card.

**Evidence:** `DESIGN.md`, “Layout and grid system,” documents a 300 px area plot versus 350 px bar/line, empty, and loading regions. `INCONSISTENCIES.md`, “Two chart-height conventions,” states that changing visualization type can change occupied height. `COMPONENT-INVENTORY.md` identifies the Area, Bar, Line, loading, and empty variants.

**Affected screens/components:** Overview `/`; `Chart`; `ChartLoading`; Area, Bar, and Line variants; surrounding chart-card grid.

**User impact:** Audit judgment: chart changes may cause avoidable content movement or make the dashboard feel less stable during exploration and loading. This is a perceived-performance judgment; animation smoothness and actual layout shift were not measured.

**Design-system impact:** Visualization containers do not share a stable geometry contract across loaded, loading, empty, and alternate-chart states.

**Suggested direction:** Define stable visualization-region behavior across equivalent chart modes and async states, allowing intentional exceptions only when the data form requires them.

### UX-014 — Repeated feature structures are copied rather than governed by shared patterns

**Severity:** Medium

**Problem:** Account/category CRUD, list loading shells, auth form composition, and spinner usage are maintained as parallel implementations.

**Evidence:** `DESIGN.md`, “Design principles currently evident,” states that similar structures are often copied rather than composed. `INCONSISTENCIES.md`, “Duplicated and parallel patterns,” documents near-copy account/category forms, sheets, columns, list-loading shells, auth composition, and repeated loaders. `COMPONENT-INVENTORY.md` identifies the repeated component groups.

**Affected screens/components:** Account and category forms/sheets/tables; `/accounts`, `/categories`, `/transactions` loading shells; sign-in, sign-up, forgot-password, reset-password; all inline spinner usages.

**User impact:** Audit judgment: immediate user harm may be limited, but drift is already visible in wording and state semantics, making future inconsistencies more likely and corrections less reliable across screens.

**Design-system impact:** Component reuse exists at the primitive level but is weaker at the workflow-pattern level, limiting design scalability and consistent evolution.

**Suggested direction:** Govern repeated workflow structures through shared experience contracts and composition patterns, without erasing domain-specific language or validation.

### UX-015 — Layout, overlay, elevation, and shape conventions are locally assembled

**Severity:** Medium

**Problem:** Equivalent layout roles use differing maximum-width expressions, overlay width rules, shadow combinations, spacing compositions, chart dimensions, and small-control radii.

**Evidence:** `DESIGN.md`, sections “Spacing system,” “Layout and grid system,” “Shape and radius system,” and “Elevation and shadow system,” documents the parallel 1536 px expressions, overlay widths, page/card/auth rhythms, 16 px checkbox radius treatment, and combined box/drop shadows. `INCONSISTENCIES.md`, “Spacing, size, radius, and elevation drift,” details the affected conventions.

**Affected screens/components:** Dashboard route containers; auth shell; right form sheets; left navigation sheet; dialogs; Cards; Checkbox; list and chart regions.

**User impact:** Audit judgment: the product may present subtle but cumulative differences in density, containment, and elevation. Exact rendered severity cannot be determined from source reconstruction alone.

**Design-system impact:** Layout and surface roles are not consistently named or constrained, so new screens must reconstruct composition rules locally.

**Suggested direction:** Document role-based layout and surface conventions while preserving distinct overlay purposes and explicitly retaining justified exceptions.

### UX-016 — Local chart choices do not persist like global analytical filters

**Severity:** Low

**Problem:** Account and date filters are URL-backed, while chart visualization choices remain local component state.

**Evidence:** `DESIGN.md`, “Navigation patterns,” documents URL persistence for account/date filters and local state for chart type selections. `INCONSISTENCIES.md`, “Interaction-pattern inconsistencies,” records the same distinction. `COMPONENT-INVENTORY.md` locates the select-driven chart variants in `Chart` and `SpendingPie`.

**Affected screens/components:** Overview `/`; `Chart`; `SpendingPie`; browser navigation, reload, and shareable analytical context.

**User impact:** Audit judgment: users may lose their chosen chart representation when reloading or revisiting and cannot reliably share the same visualization state, although the underlying filtered data context remains persistent.

**Design-system impact:** The product has no explicit rule distinguishing shareable/persistent analytical preferences from ephemeral view controls.

**Suggested direction:** Define which analytical controls form navigable state and apply that rule consistently rather than automatically persisting every local choice.

### UX-017 — Visible copy and action-label conventions contain localized defects

**Severity:** Low

**Problem:** Isolated spelling, grammar, tense, capitalization, and domain-noun inconsistencies reduce textual polish.

**Evidence:** `DESIGN.md`, “Feedback patterns,” documents “Suscriptions” and “Add a new transactions.” `INCONSISTENCIES.md`, “Toast wording and capitalization drift” and “Visible copy contains isolated errors,” documents “Error to delete account,” mixed sentence/title case, and inconsistent action wording.

**Affected screens/components:** Category creation; new transaction sheet; mutation toasts; form action labels; sheet titles.

**User impact:** Audit judgment: inconsistent or incorrect language can create minor hesitation and reduce perceived product reliability, particularly around actions and outcomes.

**Design-system impact:** Content roles do not have a shared voice, tense, capitalization, or domain-term standard.

**Suggested direction:** Establish concise product-language conventions for actions, pending states, success/error outcomes, and domain nouns, then audit current copy without changing underlying behavior.

### UX-018 — Color is not consistently reinforced as a state signal

**Severity:** Medium

**Problem:** Income/expense and categorized/uncategorized states use green/rose distinctions, but non-color reinforcement varies by component.

**Evidence:** `DESIGN.md`, “Accessibility behavior,” states that color contributes to income/expense and categorized/uncategorized distinctions and that icons or signs reinforce some occurrences but not all. `INCONSISTENCIES.md`, “Color is part of several state distinctions,” identifies variation across amount input, badge, chart, and table presentations. `COMPONENT-INVENTORY.md` locates these treatments in `AmountInput`, transaction badges/cells, and chart variants.

**Affected screens/components:** `AmountInput`; transaction amount badges; categorized/uncategorized cells; charts and tooltips; summary states.

**User impact:** Audit judgment: users with reduced color discrimination may not receive an equivalent distinction in every presentation. Actual contrast and color-vision performance were not measured.

**Design-system impact:** Domain-state semantics do not require a consistent secondary cue such as sign, label, icon, or position.

**Suggested direction:** Define redundant state-cue principles so color supports meaning without carrying it alone, adapted appropriately to controls, tables, summaries, and charts.

## 6. Evaluation-dimension coverage

| Evaluation dimension | Principal findings | Audit conclusion |
| --- | --- | --- |
| Visual hierarchy | UX-011, UX-012, UX-015 | A strong header/card foundation exists, but role hierarchy is locally assembled and cannot be assumed visually uniform. |
| Information density | UX-005, UX-013, UX-015 | Desktop management density is coherent; narrow-screen prioritization and chart geometry are unresolved. |
| Discoverability | UX-002, UX-005, UX-010 | Several actions depend on hidden cell behavior, unlabeled icons, row menus, or off-screen columns. |
| Navigation | UX-007, UX-010, UX-016 | Route order and URL filters are sound; breakpoint behavior and interaction/state persistence are inconsistent. |
| Consistency | UX-008–UX-017 | Drift spans controls, feedback, interaction, language, color, typography, and layout. |
| Accessibility | UX-002, UX-006, UX-009, UX-012, UX-018 | Form and Radix foundations are strong, but local controls and state communication do not consistently match them. |
| Readability | UX-005, UX-012, UX-017 | Text scales are documented, but semantic roles, mobile density, and language quality are inconsistent. |
| Forms | UX-003, UX-006, UX-008, UX-012 | Validation and ARIA relationships are strong; mode, composite disabled state, select behavior, and sizing drift remain. |
| Error prevention | UX-003, UX-004, UX-006 | Edit mode, destructive confirmation, and pending-state behavior require clearer action contracts. |
| Feedback | UX-001, UX-009, UX-013, UX-017 | Authentication feedback is comparatively explicit; dashboard reads and cross-product status semantics are weaker. |
| Mobile/responsive behavior | UX-005, UX-007, UX-008, UX-015 | Shells and headers adapt, but core tables and exact breakpoint contracts do not form a complete responsive system. |
| Component reuse | UX-008, UX-014, UX-015 | Reusable primitives are established; workflow patterns and certain role-level components remain duplicated or parallel. |
| Interaction consistency | UX-003, UX-006, UX-008, UX-010, UX-016 | Closely related controls and actions do not always share mode, state, or persistence behavior. |
| Perceived performance | UX-001, UX-009, UX-013 | Loading coverage exists, but ambiguous failures, uneven announcements, and changing chart occupation can weaken perceived stability. |
| Design scalability | UX-009, UX-011, UX-012, UX-014, UX-015, UX-018 | The primitive foundation is scalable, but semantic roles and workflow-level governance are incomplete. |

## 7. Overall assessment

The current application has sufficient reusable structure to support systematic improvement without assuming a full redesign. The evidence indicates that the priority is not cosmetic unification. It is to make system status, action semantics, accessibility, and responsive task completion dependable across the product. The next level of work should preserve the established shells, form semantics, overlay primitives, route model, and URL-backed filters while resolving the High-risk behavioral inconsistencies and defining cross-screen contracts for states, controls, data interaction, and content.
