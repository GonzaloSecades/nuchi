# Nuchi Current Product Design System

## Status and evidence boundary

- **[CANONICAL]** This document formalizes the Nuchi product UI reconstructed for the working tree dated 2026-09-21 at base commit `f177ade4d2ab3c754ff5e79ba45bfee128b0ddb1`, including the frontend changes and `/settings` route that were uncommitted at that time. These changes are now committed alongside this documentation.
- **[CANONICAL]** The only evidence used is `design/current/UI-MAP.md`, `design/current/COMPONENT-INVENTORY.md`, `design/current/TOKENS.md`, and `design/current/INCONSISTENCIES.md` as supplied for this task.
- **[CANONICAL]** The evidence is a source reconstruction, not a browser screenshot audit. This document records implemented behavior and the closest formal token mapping; it does not infer visual intent, validate rendered appearance, or propose a redesign.
- **[CANONICAL]** `tokens.css` is a non-destructive reference layer. It is documentation, not an application patch or a claim that the application currently consumes the proposed semantic aliases.

### Classification key

| Classification | Meaning in this document |
| --- | --- |
| **CANONICAL** | Explicitly centralized or foundational in the current implementation. |
| **CONSISTENT** | Repeated coherently, even if not centralized. |
| **INCONSISTENT** | Conflicting implementations exist for the same role. |
| **LEGACY** | Present but inactive, unused, or disconnected from current rendering. |
| **ONE-OFF** | Isolated value or behavior with no repeated system role. |

## 1. Design principles currently evident

- **[CANONICAL]** The product is built from Tailwind CSS v4 defaults, semantic CSS variables, shadcn “new-york” primitives, Radix interaction primitives, Lucide icons, and utility styling in JSX.
- **[CONSISTENT]** Dashboard information is organized into elevated cards over a strong blue header, with dense operational tables for management workflows and sheets/dialogs for focused edits.
- **[CONSISTENT]** Forms expose labels, validation messages, disabled pending states, and replacement success outcomes for authentication flows.
- **[CONSISTENT]** URL-backed account and date filters preserve analytical context when either filter changes.
- **[CONSISTENT]** Core CRUD patterns repeat across accounts and categories: one-field forms, right-side sheets, sortable/selectable tables, row menus, confirmation, and mutation feedback.
- **[INCONSISTENT]** Semantic primitives coexist with direct Tailwind colors, hard-coded colors, inline library styles, and local SVG chart colors; a change to the semantic layer does not propagate across the whole product.
- **[INCONSISTENT]** Loading, empty, error, selection, focus, and responsive behaviors do not share one presentation or semantic contract across all surfaces.
- **[INCONSISTENT]** Similar structures are often copied rather than composed: account/category CRUD, list loading shells, auth forms, and spinners are parallel implementations that can drift.

## 2. Color system

### 2.1 Central semantic palette

- **[CANONICAL]** Light and dark semantic values are centralized in `app/globals.css` and exposed to Tailwind through inline theme aliases. The light family covers background, foreground, card, popover, primary, secondary, muted, accent, destructive, border, input, ring, and five chart slots.
- **[CANONICAL]** The light foundation is white `oklch(1 0 0)` for background/card/popover, deep slate `oklch(0.129 0.042 264.695)` for foreground, `oklch(0.208 0.042 265.755)` for primary, and `oklch(0.929 0.013 255.508)` for border/input.
- **[CANONICAL]** The dark foundation reverses background/foreground, uses `oklch(0.208 0.042 265.755)` for card/popover, white-alpha borders and inputs, and its own primary, destructive, ring, and chart values.
- **[LEGACY]** Light and dark sidebar token families are declared but no current sidebar primitive consumes them; mobile navigation uses a Sheet instead.
- **[LEGACY]** The ten declared light/dark `--chart-*` values are disconnected from the currently rendered Recharts variants.
- **[LEGACY]** Dark semantic variables and multiple `dark:` styles exist, but no ThemeProvider or theme switcher is mounted. Dark mode therefore has no complete product-owned activation path.

### 2.2 Direct and domain colors

- **[CONSISTENT]** Direct Tailwind blues form the dashboard/auth brand surfaces: `blue-700` to `blue-500` in the header, `blue-500` in the auth panel and primary badge, and `blue-600` for blue hover/action states.
- **[CONSISTENT]** Emerald represents income/positive states; rose represents expense/negative or uncategorized states; red represents explicit error surfaces; slate supports neutral controls and spinners.
- **[INCONSISTENT]** Domain colors bypass the semantic primary/destructive aliases, so equivalent roles are expressed through both semantic variables and direct palette utilities.
- **[INCONSISTENT]** Auth text uses `#2E2A47` and `#7E8CA0` rather than the semantic foreground and muted-foreground values.
- **[ONE-OFF]** Dashboard welcome/date text uses `#89b6fd` as a direct supporting color.
- **[INCONSISTENT]** React Select fixes its border and hover border to `#e2e8f0`, bypassing semantic border/input and dark values.
- **[INCONSISTENT]** Rendered charts use hard-coded `#3d82f6`, `#3b82f6`, `#f43f5e`, `#0062FF`, `#12C6FF`, `#FF647F`, `#FF9354`, and `#8884d8` rather than declared chart tokens.
- **[INCONSISTENT]** Several near-primary blues remain separate because they are distinct current values: Tailwind `blue-500`, chart series `#3d82f6`, radar `#3b82f6`, category blue `#0062FF`, and dark `--chart-1`.
- **[INCONSISTENT]** Hard-coded white chart tooltips and direct/library colors do not automatically follow the dark semantic set.

## 3. Typography system

- **[CANONICAL]** Geist Sans is loaded through `next/font/google`, assigned to `--font-geist-sans`, and mapped to the Tailwind sans family used by preflight.
- **[CANONICAL]** Geist Mono is loaded, assigned to `--font-geist-mono`, and mapped to the Tailwind mono family.
- **[LEGACY]** No current frontend component explicitly requests the mono family.
- **[CANONICAL]** The implemented size/line-height scale is Tailwind’s `text-xs` 12/16, `text-sm` 14/20, `text-base` 16/24, `text-lg` 18/28, `text-xl` 20/28, `text-2xl` 24/32, `text-3xl` 30/36, and `text-4xl` 36/40 px.
- **[CANONICAL]** `leading-none` supplies a unitless line height of `1` where compact headings request it.
- **[CONSISTENT]** Implemented weights are 400, 500, 600, and 700; body and control text most often use 14 px, while headings and summary figures use the upper scale.
- **[CONSISTENT]** `tracking-widest` is `0.1em` where explicitly used.
- **[INCONSISTENT]** Screen, card, auth, status, summary, overlay, chart, and tooltip roles assemble recipes locally rather than consuming one named typographic role system.
- **[INCONSISTENT]** Standard inputs, textarea, AmountInput, and creatable selects transition from 16 px to 14 px at the `md` boundary, while React Select also depends on a separate `max-width: 767px` global override.

## 4. Spacing system

- **[CANONICAL]** Tailwind’s 4 px base unit is the foundational spacing scale.
- **[CONSISTENT]** Repeated spacing steps include 2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 36, 40, 48, 56, 64, 96, 120, 144, and 160 px.
- **[CONSISTENT]** Cards use 24 px horizontal padding; the dashboard shell uses 12 px page padding below `lg` and 56 px at `lg`; the auth shell uses 16 px horizontal and 48 px vertical padding.
- **[CONSISTENT]** Dashboard cards overlap the header by 96 px while the header reserves 144 px of bottom padding.
- **[INCONSISTENT]** Page, card, form, and auth spacing use different local compositions rather than named layout rhythms.
- **[ONE-OFF]** Fractional/arbitrary fixed dimensions documented in current use include 170, 300, 350, 420, and 500 px; they are retained as reference values rather than normalized into the 4 px rhythm.

## 5. Layout and grid system

- **[CANONICAL]** The root rendering chain is root layout → Geist variables → global Tailwind/semantic styles → SessionProvider → QueryProvider → global Toaster → route-group layout.
- **[CANONICAL]** Authentication uses one full-height form column below `lg` and an equal 50/50 form/brand split from `lg`; form content is centered and capped at 28 rem.
- **[CANONICAL]** The dashboard shell is session guarded, mounts global create/edit sheets, places content over a tall blue header, and caps header/page content at 96 rem (1536 px).
- **[INCONSISTENT]** The same 1536 px cap is expressed as both `max-w-screen-2xl` and `max-w-(--breakpoint-2xl)`.
- **[CONSISTENT]** Overview uses a three-card summary grid followed by two chart cards; chart proportions change from equal columns to 4/2 at `xl`.
- **[CONSISTENT]** List screens use one overlapping card containing title/actions, filter controls, a horizontally scrollable table, pagination, and bulk actions.
- **[CONSISTENT]** Right form sheets are full viewport width below `lg` and cap at 28 rem from `lg`; left navigation sheets use 75% width and cap at 24 rem from `sm`; dialogs use viewport width minus 2 rem and cap at 32 rem from `sm`.
- **[INCONSISTENT]** Overlay types do not share one width convention.
- **[INCONSISTENT]** Tables preserve desktop columns on narrow screens and use horizontal scrolling rather than a mobile row/card representation.
- **[INCONSISTENT]** Area charts use a 300 px plot height while bar/line, empty, and loading regions use 350 px, so chart type changes can alter card occupation.

## 6. Shape and radius system

- **[CANONICAL]** The centralized base radius is `0.625rem` (10 px).
- **[CANONICAL]** Derived radius aliases are 6 px (`sm`), 8 px (`md`), 10 px (`lg`), 14 px (`xl`), 18 px (`2xl`), 22 px (`3xl`), and 26 px (`4xl`).
- **[CONSISTENT]** Most controls use 8 px, cards use 14 px, and badges/avatars use full rounding.
- **[INCONSISTENT]** A 16 px checkbox uses the 10 px radius, while tiny controls use 8 px and isolated elements use either Tailwind’s independent 2 px `rounded-xs` or an arbitrary 2 px radius.

## 7. Elevation and shadow system

- **[CANONICAL]** The available shadow values are `shadow-xs`, `shadow-sm`, `shadow-md`, `shadow-lg`, and `drop-shadow-sm`, preserved exactly in `tokens.css`.
- **[CONSISTENT]** Base cards use `shadow-sm`; inputs and outline controls may use `shadow-xs`; overlays request `shadow-sm`, `shadow-md`, or `shadow-lg` by component.
- **[INCONSISTENT]** Dashboard route cards can combine the Card primitive’s `shadow-sm` box shadow with a screen-level `drop-shadow-sm` filter, while other cards use only the primitive shadow.
- **[CONSISTENT]** Modal dialogs and sheets use black at 50% opacity for overlays and share `z-50` with overlay content families.

## 8. Iconography

- **[CANONICAL]** Product icons come from `lucide-react`.
- **[CANONICAL]** Primitive defaults normalize unspecified SVGs to 16 px; call sites also use 12, 14, 16, 20, 24, and 32 px.
- **[CANONICAL]** `public/logo.svg` has a 56 × 40 view box and a white mark; current render sizes are 200 × 200 in the auth shell and 28 × 28 in the dashboard header.
- **[CONSISTENT]** `Loader2` is the repeated loading glyph, combined with spin motion at full-screen, bounded-panel, sheet-overlay, and button-inline scales.
- **[INCONSISTENT]** Icon-only controls do not all expose accessible names; documented omissions include mobile navigation, row action menus, date-filter reset, and amount sign reversal.

## 9. Component taxonomy

| Classification | Family | Current members and role |
| --- | --- | --- |
| **CANONICAL** | Primitives | Button, Badge, Input, Textarea, Label, Form, Card, Table, Checkbox, Radix Select, Calendar, Separator, Skeleton, VisuallyHidden. |
| **CANONICAL** | Navigation | Header, HeaderLogo, Navigation, NavButton, UserButton, WelcomeMsg. |
| **CONSISTENT** | Forms | AccountForm, CategoryForm, TransactionForm, DatePicker, AccountFilter, DateFilter, Filters, AmountInput. |
| **INCONSISTENT** | Select controls | Radix Select and creatable React Select implement parallel control/menu/focus/theme systems. |
| **CONSISTENT** | Data display | DataTable, DataGrid, DataCard, Countup, DataCharts, Chart, SpendingPie, chart variants, and custom tooltips. |
| **CONSISTENT** | Feedback | AuthError, global Sonner Toaster, inline spinners, Skeleton, empty rows/panels, and outcome panels. |
| **CANONICAL** | Overlays | Dialog, Sheet, DropdownMenu, Popover, Tooltip, and promise-returning confirmation hook, based on Radix interaction behavior. |
| **CONSISTENT** | Domain modules | Account/category/transaction columns and actions, CSV import, global CRUD sheets, and settings panel. |
| **LEGACY** | Exported but uncomposed parts | Additional dropdown subparts, CardDescription/CardAction/CardFooter, and unused subsets of dialog/sheet primitive families exist without current standalone composition. |

## 10. Component states

- **[CANONICAL]** Buttons expose default, destructive, outline, secondary, ghost, and link variants plus default/xs/sm/lg/icon sizes; hover, focus-visible, invalid, disabled, and dark styles exist.
- **[CANONICAL]** Radix-backed components expose open/closed, checked, highlighted, disabled, selected, and side-aware states through attributes and built-in keyboard/focus behavior.
- **[CONSISTENT]** Pending forms disable value controls and present an inline spinner in the main submit action.
- **[CONSISTENT]** Tables expose filtered, sorted, selected, indeterminate, empty, paginated, bulk-delete confirmation, and pending deletion states.
- **[INCONSISTENT]** The edit transaction sheet omits the transaction `id`, causing create wording and hiding the form-level delete action even though row deletion still exists.
- **[INCONSISTENT]** Disabled AmountInput state reaches the currency field but not the adjacent sign-reversal button.
- **[INCONSISTENT]** Generic bulk-delete confirmation copy refers to accounts when the component is also used for categories and transactions.
- **[INCONSISTENT]** Loading semantics vary: sign-in restoration has `role="status"` and an accessible label, while most other spinners are visual icons without equivalent status text.

## 11. Form behavior

- **[CANONICAL]** React Hook Form and Zod drive auth and CRUD validation; `FormControl` provides `aria-invalid` and `aria-describedby`, with generated relationships among labels, descriptions, and messages.
- **[CONSISTENT]** Sign-in requires valid email and non-empty password; sign-up requires valid email and an eight-character password; reset requires eight characters and matching confirmation.
- **[CONSISTENT]** Account/category forms require a non-empty name and switch create/edit/pending behavior based on supplied data.
- **[CONSISTENT]** Transaction validation requires date, account, payee, and a safely parseable milliunit amount; category and notes are optional.
- **[CONSISTENT]** CSV mapping requires unique assignments for amount, date, and payee before Continue becomes enabled.
- **[INCONSISTENT]** Standard inputs are 36 px tall, while React Select, AmountInput, and several large actions are 40 px and can appear in the same form.
- **[INCONSISTENT]** Focus treatment is 3 px at 50% ring color for most primitives, 2 px plus a 2 px offset for AmountInput, locally overridden/suppressed in the header, and library-controlled for React Select.

## 12. Feedback patterns

- **[CONSISTENT]** Loading appears as full-viewport session spinners, auth bootstrap regions, 500 px list panels, 350 px chart panels, full-sheet overlays, button spinners, and skeletons that preserve summary-card chrome.
- **[INCONSISTENT]** These loading presentations repeat `Loader2 animate-spin` instead of consuming a single loading component; sizes and colors vary.
- **[CONSISTENT]** Auth field/server errors use field messages and `AuthError`; CSV row validation uses an inline `role="alert"`; mutation and parse failures use Sonner toasts; missing/expired tokens use dedicated outcome panels.
- **[INCONSISTENT]** Dashboard query errors have no dedicated visual branch and may appear as zeros, empty charts, “No results,” or blank edit defaults.
- **[INCONSISTENT]** Empty patterns vary between a table row, icon-and-copy chart panel, bordered CSV alert, and full auth status panel.
- **[CONSISTENT]** Authentication success generally replaces the form inline; dashboard mutation success generally closes its sheet and emits a toast.
- **[INCONSISTENT]** Toast tense/capitalization and action label casing are authored per feature and drift.
- **[ONE-OFF]** Current visible-copy errors include “Suscriptions” and “Add a new transactions”; they are documented as implementation facts, not normalized here.

## 13. Navigation patterns

- **[CANONICAL]** Dashboard route order is Overview, Transactions, Accounts, Categories, Settings.
- **[CANONICAL]** Desktop navigation is a horizontal row of outline NavButtons with transparent/10%-white active treatment; mobile navigation is an icon trigger opening a left Sheet.
- **[CONSISTENT]** The mobile sheet closes explicitly after route selection.
- **[CANONICAL]** The user menu uses a circular email-initial avatar, exposes the full email and sign-out action, and reports confirmed logout failure through an error toast.
- **[CONSISTENT]** Account and date filters live in the header, write URL search parameters, and preserve each other’s state.
- **[INCONSISTENT]** Chart type selections are local component state and not encoded in the URL, unlike account/date filters.
- **[INCONSISTENT]** Table edit entry points vary among clickable names/cells, uncategorized-category behavior, row action menus, and optional form delete actions.

## 14. Responsive behavior

| Classification | Boundary | Implemented behavior |
| --- | --- | --- |
| **INCONSISTENT** | JS `max-width: 640px` and CSS `sm: 640px` | DateFilter uses its mobile branch at exactly 640 px while `sm` rules also begin. |
| **ONE-OFF** | CSS `max-width: 767px` | Global `.nuchi-select__*` rules force React Select text to 16 px. |
| **CANONICAL** | `md: 768px` | Inputs, textarea, AmountInput, and creatable select text change from 16 px to 14 px; calendar months may become a row. |
| **INCONSISTENT** | JS `max-width: 1024px` and CSS `lg: 1024px` | Mobile navigation is selected at exactly 1024 px while desktop-oriented `lg` styles also begin. |
| **CANONICAL** | `lg: 1024px` | Auth split screen, header brand/desktop navigation, row filters/actions, expanded grids, and 56 px horizontal dashboard padding. |
| **CANONICAL** | `xl: 1280px` | Overview chart columns change from equal proportions to 4/2. |
| **CANONICAL** | `2xl: 1536px` | Header and dashboard content reach their maximum width. |
- **[INCONSISTENT]** Responsive adaptation mixes JS render-time component replacement with CSS-only changes, creating distinct boundary and hydration characteristics for related UI.
- **[INCONSISTENT]** Narrow tables remain desktop tables with horizontal overflow; no columns are selectively hidden and no card/list alternative is implemented.

## 15. Accessibility behavior

- **[CANONICAL]** Form primitives connect labels, descriptions, and messages through generated IDs and expose invalid state with ARIA attributes.
- **[CANONICAL]** Radix overlays supply keyboard interaction and focus management; the mobile navigation Sheet includes a visually hidden title.
- **[CONSISTENT]** Auth-level and CSV import errors use `role="alert"`; sign-in restoration uses `role="status"` with an accessible label.
- **[INCONSISTENT]** Most other standalone loading indicators lack an equivalent status region or loading label.
- **[INCONSISTENT]** Transaction account/category table cells are clickable `div` elements without button semantics, tab stops, or keyboard handlers.
- **[INCONSISTENT]** Several icon-only buttons lack an explicit accessible name, including the documented mobile menu, row menus, date reset, and amount sign reversal controls.
- **[INCONSISTENT]** Color contributes to income/expense and categorized/uncategorized distinctions; icons or signs reinforce some occurrences but not all.
- **[CONSISTENT]** Tables retain semantic table structure and an overflow wrapper, while Checkbox exposes checked and indeterminate states.

## 16. Motion and transition patterns

- **[CANONICAL]** `tw-animate-css` supplies fade, zoom, and directional slide animations for Radix overlays based on open/closed and side state.
- **[CONSISTENT]** `Loader2` uses continuous spin; Skeleton uses pulse; Countup animates dashboard numeric values.
- **[CONSISTENT]** Buttons use `transition-all`; inputs transition color and shadow.
- **[INCONSISTENT]** No centralized duration or easing tokens are declared, so motion names exist without a documented timing scale.
- **[ONE-OFF]** Chart hover/active behavior is defined within individual Recharts variants rather than through a shared product motion role.

## Formal token interpretation

- **[CANONICAL]** Tokens prefixed `--nuchi-current-*` in `tokens.css` transcribe centralized current implementation values.
- **[CONSISTENT]** Tokens prefixed `--nuchi-ref-*` name repeated current conventions without claiming that application code consumes those names.
- **[INCONSISTENT]** Parallel aliases are intentionally retained for conflicting current values; they must not be collapsed without a separate migration decision.
- **[LEGACY]** Disconnected chart/sidebar and inactive dark values remain documented because they are present in the current implementation.
- **[ONE-OFF]** Isolated values remain separately named so the reference layer does not manufacture a repeated semantic role.
