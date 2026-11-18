# Current UI inconsistencies

## Reading this document

These are observed differences in the current implementation, not proposed
changes. Similar patterns are listed even when each version may have been
intentional. “Impact” describes what the code produces today; it is not a
redesign priority.

## Duplicated and parallel patterns

### Account and category CRUD are near copies

- `features/accounts/components/account-form.tsx` and
  `features/categories/components/category-form.tsx` have the same one-field
  form structure, create/edit switching, pending state, and optional delete
  action.
- Their new/edit sheets under `features/accounts/components/` and
  `features/categories/components/` repeat the same shell, dimensions, loading
  overlay, and mutation wiring.
- `app/(dashboard)/accounts/{columns,actions}.tsx` and
  `app/(dashboard)/categories/{columns,actions}.tsx` repeat select, name-sort,
  edit, and delete patterns.
- The duplication has already drifted in domain wording and makes later visual
  divergence possible even though the screens currently appear very similar.

### List-page loading shells are repeated

`/accounts`, `/categories`, and `/transactions` each implement their own card,
title skeleton, and centered spinner with a 500 px content height. There is no
shared list-loading component even though the visual structure is the same.

### Auth form composition is repeated

Sign-in, sign-up, forgot-password, and reset-password independently compose the
same `AuthHeader -> Form -> FormField -> Input -> AuthError -> Button` pattern.
The route-specific validation is appropriate, but vertical rhythm, error
placement, pending labels, and success-panel construction are maintained in
separate files.

### Loading indicators are not a component

`Loader2 animate-spin` is repeated as:

- a full-viewport session guard,
- auth bootstrap and Suspense fallbacks,
- 500 px list-loading regions,
- 350 px chart-loading regions,
- absolute edit-sheet overlays, and
- inline pending buttons.

Sizes range from 1 rem to 2 rem, colors vary, and only the sign-in restoration
state has an explicit status role and accessible label.

## Duplicated component roles

### Two unrelated select systems

- `components/ui/select.tsx` is a Radix select using semantic variables,
  Tailwind classes, Radix state attributes, and portal content.
- `components/select.tsx` is a creatable React Select with an inline border
  color, a library-specific DOM/class structure, and a mobile CSS override in
  `app/globals.css`.

The Radix version is used for filters, chart switchers, and CSV mapping. The
React Select version is used for transaction account/category fields and the
import account dialog. Their heights, menus, focus appearance, empty behavior,
and dark-mode behavior come from different systems.

### Several state surfaces represent errors differently

- Auth validation uses field messages and `AuthError`.
- CSV parse and mutation errors use Sonner toasts.
- CSV row validation uses an inline red `role=alert` panel.
- Missing/expired auth tokens use dedicated full outcome panels.
- Dashboard query failures have no dedicated visual branch.

The same broad “error” concept therefore appears as inline field text, an alert
box, a toast, a full screen state, or no visible error.

### Empty-state patterns drift

- Generic tables render one row containing “No results.”
- Charts render an icon with “No data for this period.”
- CSV empty/header-only input renders a bordered alert.
- Missing auth tokens render a full status panel.

Iconography, alignment, height, explanatory copy, and available recovery
actions differ across these cases.

## Color-system drift

### Semantic tokens and direct palette utilities coexist

Primitive buttons and inputs mainly use `primary`, `destructive`, `border`, and
other semantic aliases. Dashboard chrome and domain states use direct
`blue-*`, `emerald-*`, `rose-*`, `red-*`, and `slate-*` utilities instead.
Changing a semantic token would not update those direct palette uses.

### Auth colors form a separate palette

`AuthHeader` and auth pages use hard-coded `#2E2A47` and `#7E8CA0`, while the
rest of the application uses `foreground` and `muted-foreground`. These values
are close in purpose but are not linked.

### Header supporting text is another direct value

`components/welcome-msg.tsx` uses `#89b6fd` instead of an opacity of white, a
blue palette step, or a semantic token. Header nav/control states otherwise use
white with different opacity values.

### Chart tokens are disconnected from chart rendering

`app/globals.css` defines five light and five dark `--chart-*` variables, but
the Recharts variants use hard-coded `#3d82f6`, `#3b82f6`, `#f43f5e`,
`#0062FF`, `#12C6FF`, `#FF647F`, `#FF9354`, and `#8884d8`. The declared chart
theme cannot change the rendered charts.

### Near-identical blues are preserved separately

- Tailwind `blue-500` is used for header/auth brand surfaces.
- `#3d82f6` is used by area/bar/line charts.
- `#3b82f6` is used by radar charts.
- `#0062FF` begins the category palette.
- The dark semantic `--chart-1` is another blue.

They serve related “primary/chart blue” roles without a shared value.

### React Select border bypasses semantic borders

`components/select.tsx` fixes the control and hover border at `#e2e8f0`.
Standard controls use `--input`/`--border`, so changes to the semantic border or
dark class do not reach this control.

### Dark styling exists without a complete activation path

Dark variables and many `dark:` primitive styles exist, and the Sonner wrapper
calls `useTheme`, but `app/layout.tsx` mounts no `ThemeProvider` and exposes no
theme control. If a `.dark` class were introduced externally, hard-coded white
chart tooltips, direct colors, and React Select inline styles would still not
follow the semantic theme.

### Unused sidebar tokens remain in the global theme

Light and dark sidebar variable families are declared, but the current mobile
navigation uses `Sheet` and no sidebar component consumes these tokens.

## Spacing, size, radius, and elevation drift

### Two chart-height conventions

Area chart content uses 300 px, while bar/line charts use 350 px; chart empty
and loading regions also use 350 px. Switching visualization type can therefore
change the occupied plot height inside the same card.

### Equivalent maximum width expressed two ways

Dashboard screens use both `max-w-screen-2xl` and
`max-w-(--breakpoint-2xl)`. Both currently resolve to 1536 px but communicate
the constraint differently and depend on different-looking tokens.

### Cards may receive two shadow systems

The base `Card` always applies `shadow-sm`. Dashboard route containers add
`drop-shadow-sm`, so those cards can carry both a box shadow and a CSS filter
drop shadow. Other cards use only the primitive shadow.

### Control height is split between 36 and 40 px

The base `Input` and default `Button` are 36 px high. React Select, AmountInput,
and several large actions are 40 px. These controls can appear within the same
form while retaining separate height rules.

### Radius roles are not consistent by component size

Most controls use the calculated 8 px `rounded-md`, cards use 14 px
`rounded-xl`, and avatars/badges use full rounding. The 16 px checkbox uses
`rounded-lg` (10 px), while several tiny controls use 8 px and a few elements
use the independent 2 px `rounded-xs` or an arbitrary 2 px radius.

### Focus rings differ

Most primitives use a 3 px ring at 50% opacity plus a focus border. AmountInput
uses a 2 px ring plus a 2 px offset. Header controls locally override or remove
the standard ring for contrast. React Select delegates focus behavior to its
library styling and does not share the primitive rule.

### Page and component padding mix abstractions

Cards establish 24 px horizontal padding, list pages add their own responsive
header/action gaps, and the dashboard shell uses 12 px page padding below
`lg`/56 px above it. Auth uses a separate 16 px/48 px shell rhythm. The code
contains both scale values and arbitrary fractional sizes such as 170, 350,
420, and 500 px without named layout tokens.

## Typography drift

### Semantic roles do not map to one type recipe

Screen titles, card titles, auth headings, status headings, summary numbers,
and overlay titles assemble size/weight/line-height directly at each call site.
There are reusable `CardTitle` and `AuthHeader`, but no shared screen-heading or
status-heading role.

### Input text changes at `md`, while React Select changes one pixel earlier

Standard inputs, textarea, AmountInput, and the creatable select use 16 px text
below `md` and 14 px from 768 px. A global React Select override applies 16 px
only through 767 px. These produce the same intended transition but through
separate mechanisms and adjacent thresholds.

### Chart and tooltip typography is local

Recharts axes/labels rely partly on library defaults and inline props, while
custom tooltips use Tailwind text utilities. Neither derives from a named chart
typography role.

## State and behavior inconsistencies

### Dashboard query errors are visually indistinguishable from valid emptiness

Overview and list screens do not render explicit query-error branches. Depending
on the hook/component, a failed read appears as zero totals, an empty chart,
“No results,” or blank edit-form defaults. Mutation failures, by contrast, are
surfaced with toasts.

### Edit transaction form receives create-state semantics

`features/transactions/components/edit-transaction-sheet.tsx` renders
`TransactionForm` without its transaction `id`. The form uses the presence of
`id` to choose “Save changes” versus “Create Transaction” and to show its
in-form delete button. As implemented, the edit sheet uses create wording and
does not show that delete action, although row actions still allow deletion.

### Disabled amount input is only partially disabled

`AmountInput` forwards `disabled` to the currency field but not to the adjacent
sign-reversal `<button>`. During a pending form, the value field is visibly
disabled while the icon button remains operable.

### Bulk-delete confirmation is domain-specific in a generic component

`components/data-table.tsx` owns a reusable bulk-delete flow, but its
confirmation copy refers to deleting “accounts and their account data.” The
same component is used for categories and transactions, so those screens show
account-specific wording.

### Loading-state semantics are uneven

Sign-in session restoration uses `role="status"` and an accessible label. Most
other standalone spinners are visually animated icons without a status region
or equivalent loading text.

### Toast wording and capitalization drift

Mutation strings are written per hook and include constructions such as “Error
to delete account,” while other messages use different tense and title casing.
Button labels also mix sentence case and title case, including “Create
Transaction” beside sentence-case actions.

### Visible copy contains isolated errors

- Category creation uses the placeholder “Suscriptions.”
- The new transaction sheet says “Add a new transactions.”
- Domain nouns and action capitalization vary between sheet titles, submit
  buttons, and toasts.

These are recorded as current UI text, not normalized here.

## Accessibility inconsistencies

### Icon-only controls do not all have accessible names

The following visual icon buttons have no explicit `aria-label` or visible
screen-reader text at their call sites:

- mobile navigation menu trigger in `components/navigation.tsx`;
- row action menu triggers in account/category/transaction `actions.tsx`;
- date-filter reset icon control;
- amount sign-reversal button in `components/amount-input.tsx`.

The amount button has a tooltip, but the tooltip text is not an explicit button
name in the component markup.

### Clickable table cells are not keyboard controls

Transaction account/category cells use `div` elements with `onClick` to open an
edit sheet. They do not have button semantics, tab stops, or keyboard handlers,
so the same interaction is not available through ordinary keyboard navigation.

### Good form ARIA is not matched by all custom controls

The Form primitive consistently links labels, descriptions, error messages,
and `aria-invalid`. Radix overlays supply titles/focus behavior, including a
visually hidden mobile sheet title. The hand-built loading regions and clickable
cells do not reach the same semantic coverage.

### Color is part of several state distinctions

Income/expense and categorized/uncategorized treatments use green/rose color
changes plus icons or signs in some places, but the exact reinforcement differs
between amount input, badge, chart, and table presentations.

## Responsive inconsistencies

### JavaScript and CSS breakpoint boundaries overlap

- Navigation chooses its mobile component with `max-width: 1024px`, while
  Tailwind `lg:` styles begin at `min-width: 1024px`.
- DateFilter chooses its mobile component with `max-width: 640px`, while
  Tailwind `sm:` styles begin at `min-width: 640px`.

At exactly 1024 px or 640 px, the JavaScript branch is mobile while CSS rules
for the larger breakpoint may also apply.

### React Select owns a one-off breakpoint

`app/globals.css` uses `max-width: 767px` only for `.nuchi-select__*` font
sizes. Other control responsiveness is expressed through Tailwind `md` at
768 px. The visual result aligns, but the rule lives outside the component and
outside the normal breakpoint syntax.

### Tables have overflow but no alternate mobile presentation

All data tables retain their desktop column structure and use horizontal
scrolling on narrow viewports. Header actions wrap/stack responsively, but rows
do not become cards and columns are not selectively hidden.

### Overlay width rules differ by overlay type

Right sheets are full viewport width until `lg`, then cap at 28 rem. The left
navigation sheet uses 75% width and caps at 24 rem from `sm`. Dialogs use the
viewport minus 2 rem and cap at 32 rem from `sm`. These three overlay systems do
not share one width convention.

### Responsive decisions mix render-time and CSS-time approaches

Navigation and DateFilter replace component structure using `useMedia`, while
auth layout, grids, form controls, sheets, and table overflow adapt only through
CSS. This creates different hydration/boundary considerations for visually
related responsive behavior.

## Interaction-pattern inconsistencies

- Table-row edit entry points are split: account/category names are clickable,
  uncategorized transaction category cells open the transaction sheet, and
  every row also has an action menu.
- Delete confirmation is available through row actions and generic bulk delete;
  forms sometimes add an additional delete button when they receive an `id`.
- Chart type selection is local to each chart and is not encoded in the URL;
  account/date filters are URL state and persist across navigation/query changes.
- Auth success usually replaces the form inline, while dashboard mutation
  success closes a sheet and emits a toast.
- The mobile navigation sheet closes explicitly after route selection; other
  overlays depend on form completion, outside click, Escape, or their close
  control.

