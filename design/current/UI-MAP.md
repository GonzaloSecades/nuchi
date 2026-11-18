# Current UI map

## Scope and evidence

This document reconstructs the UI from the working-tree snapshot dated
2026-09-21. The base commit is `f177ade4d2ab3c754ff5e79ba45bfee128b0ddb1`
on `master`. That snapshot included uncommitted frontend changes, including the
`/settings` route. This map includes those changes, which are now committed
alongside this documentation.

The audit covered the complete frontend source surface under `app/`,
`components/`, `features/`, `providers/`, and `hooks/`, plus
`app/globals.css`, `components.json`, `package.json`, and `public/logo.svg`.
It is a source reconstruction, not a browser screenshot audit. No visual
inconsistency has been normalized.

## Rendering and style chain

```text
app/layout.tsx
  -> next/font Geist + Geist Mono CSS variables
  -> app/globals.css
       -> Tailwind CSS v4 defaults
       -> tw-animate-css
       -> semantic light/dark CSS variables
       -> global border/ring and body color rules
       -> one mobile react-select font-size override
  -> SessionProvider
  -> QueryProvider
  -> global Sonner Toaster
  -> route-group layout
       -> auth split-screen shell, or
       -> session-guarded dashboard shell
```

Most styling is Tailwind utility strings in JSX. Reusable primitive variants use
`class-variance-authority`; conditional classes use `cn()`
(`clsx` + `tailwind-merge`). Radix supplies interaction/state attributes for
dialogs, sheets, dropdowns, popovers, selects, checkboxes, and tooltips. The
creatable account/category selector uses `react-select` and an inline style
object. Recharts uses inline SVG props and hard-coded colors.

## Application shells

### Root shell

- Source: `app/layout.tsx`.
- Sets `lang="en"`, metadata title “Nuchi,” and the Geist Sans/Mono variables.
- Body has only the font variables plus `antialiased`; Tailwind preflight
  resolves the default sans family through `--font-sans`.
- Provider order is `SessionProvider -> QueryProvider -> Toaster -> route`.
- No `ThemeProvider` or theme switcher is mounted even though dark variables
  and dark utilities exist.

### Authentication shell

- Source: `app/(auth)/layout.tsx`.
- One-column, full-height form area below `lg`; two equal columns at
  `lg` (1024 px).
- Form column: centered, `px-4 py-12`, content capped at `max-w-md` (28 rem).
- Brand column: hidden below `lg`; solid `bg-blue-500`, centered white
  `/logo.svg` rendered at 200 by 200.
- All five auth routes supply only their page content.

### Dashboard shell

- Source: `app/(dashboard)/layout.tsx`.
- `SessionGuard` blocks the entire shell until authenticated. Loading and
  unauthenticated-before-redirect both show a centered full-viewport spinner.
- `SheetProvider` mounts all six create/edit sheets globally.
- `Header` is a tall blue gradient (`blue-700 -> blue-500`) with 36 units of
  bottom padding so page cards can overlap it by `-mt-24`.
- Main content padding is 12 px by default and 56 px at `lg`.
- Header content is capped at 1536 px. Dashboard screens use either
  `max-w-screen-2xl` or `max-w-(--breakpoint-2xl)`; both resolve to 1536 px
  but are expressed differently.

## Route -> screen -> layout -> components -> primitives

| Route | Screen and implemented states | Layout and major components | Reusable primitives |
| --- | --- | --- | --- |
| `/sign-in` | “Welcome back” email/password form. While session bootstrap is unresolved it shows a `role=status` spinner instead of the form. Submit disables both fields and shows a spinner in the button. Client validation covers email and required password; server/form errors use `AuthError`. Links to password reset and sign-up. | Auth shell -> `SignInForm` -> `AuthHeader` + form. A `Suspense` fallback is a small centered spinner. | `Form` family, `Input`, `Button`, `AuthError`, `Loader2`. |
| `/sign-up` | “Create an account” email/password form. Password minimum is 8 characters. Server field errors are attached to email/password. Success replaces the form with a blue mail icon, “Check your email,” expiry guidance, and an outline back button. | Auth shell -> form or success panel. | `AuthHeader`, `Form` family, `Input`, `Button`, `AuthError`. |
| `/forgot-password` | Email form with submitting spinner and form-level error. Success replaces it with a mail icon, server-supplied message, and “Back to sign in.” The response is intentionally identical for known/unknown addresses. | Auth shell -> form or success panel. | `AuthHeader`, `Form` family, `Input`, `Button`, `AuthError`. |
| `/reset-password?token=…` | Missing token: rose error icon and request-new-link action. Valid token: new password + confirmation form. Validation covers 8-character minimum and matching confirmation. Success: emerald icon and continue-to-sign-in action. API invalid/expired token becomes a form error. | Auth shell -> missing-token, form, or success panel; small `Suspense` spinner while search params resolve. | `AuthHeader`, `Form` family, `Input`, `Button`, `AuthError`. |
| `/verify-email?token=…` | Missing token, verifying, verified, and failed states. Verification starts automatically. Retryable failures show “Try again”; terminal failures link back to sign-up; success links to sign-in. | Auth shell -> centered status panel; small `Suspense` spinner around search-param resolution. | `AuthHeader`, `Button`, `Loader2`, `CircleCheck`, `CircleX`. |
| `/` | Overview dashboard. Three summary cards (“Remaining,” “Income,” “Expenses”), then transaction and category charts. Summary loading shows three card skeletons plus two chart loaders. Empty chart data shows “No data for this period.” No explicit query-error screen exists. | Dashboard shell -> overlapping max-width container -> `DataGrid` -> `DataCard` x3; `DataCharts` -> `Chart` + `SpendingPie`. | `Card` family, `Skeleton`, Radix `Select`, Recharts, `Countup`. |
| `/transactions` | Default list card titled “Transactions History,” add and CSV import actions, filterable/sortable/selectable/paginated table. Loading is a title skeleton plus a 500 px spinner area. Import switches the screen to `ImportCard`; invalid CSV uses toasts, row validation uses an inline red alert, empty/header-only data uses a bordered alert, and account selection uses a dialog. No explicit query-error screen exists. | Dashboard shell -> overlapping card -> header actions -> `DataTable`; alternate import card -> mapping table -> account dialog. Global new/edit transaction sheets remain mounted. | `Card`, `Button`, `DataTable`, `Table`, `Checkbox`, `DropdownMenu`, `Dialog`, `Sheet`, `Select`, `Skeleton`, Sonner. |
| `/accounts` | “Accounts” card with Add new. Filter, row selection, sort by name, edit/delete menu, bulk delete, pagination, and “No results.” Loading uses the same 500 px spinner shell as other list pages. No explicit query-error screen exists. | Dashboard shell -> overlapping card -> `DataTable`; global account sheets. | `Card`, `Button`, `Input`, `Table`, `Checkbox`, `DropdownMenu`, `Dialog`, `Sheet`, `Skeleton`. |
| `/categories` | “Categories” card with the same table behavior as accounts. Loading and empty behavior match accounts. No explicit query-error screen exists. | Dashboard shell -> overlapping card -> `DataTable`; global category sheets. | Same primitive set as accounts. |
| `/settings` | “Settings” card showing the authenticated email and an outline “Reset password” link. There are no local loading, error, or empty branches; the parent session guard supplies authentication loading. | Dashboard shell -> overlapping single card -> email and password sections. | `Card`, `Button`, `Link`. |

## Navigation

### Dashboard navigation

- Routes, in order: Overview, Transactions, Accounts, Categories, Settings.
- Source: `components/navigation.tsx`.
- Desktop path (`useMedia('(max-width: 1024px)') === false`): horizontal
  `NavButton` row, outline buttons with transparent/10%-white active state.
- Mobile path (including exactly 1024 px in the JS query): icon-only menu button
  opens a left sheet. Routes become full-width ghost/secondary buttons and the
  sheet closes after navigation.
- Header logo and “Nuchi” wordmark are also hidden below Tailwind `lg`.
- The account menu is a circular email-initial avatar. It shows the full email
  and a sign-out item; a failed server-confirmed logout produces an error toast.

### Filters

- Header filters are stacked and full width below `lg`, inline at `lg`.
- Account filter is a Radix select and writes `accountId` to the current URL.
- Date filter is a Radix popover. Desktop shows preset column + two months;
  mobile (`max-width: 640px`) alternates between one-month calendar and a
  two-column preset panel. Apply and reset buttons are disabled until both
  range ends exist.
- Date/account URL state is retained when the other filter changes.

## Forms and validation

| Form | Fields | Validation and feedback | Container |
| --- | --- | --- | --- |
| Sign in | Email, password | Valid email; non-empty password; field messages + `AuthError`; disabled during submit. | Auth page. |
| Sign up | Email, password | Valid email; minimum 8 chars; server field mapping; success replaces form. | Auth page. |
| Forgot password | Email | Valid email; generic server error; success replaces form. | Auth page. |
| Reset password | Password, confirmation | Minimum 8 chars; values must match; invalid-token form error; success replaces form. | Auth page. |
| Account | Name | `z.string().min(1)`; create/save button; delete outline button only when `id` is supplied. | Right sheet. |
| Category | Name | `z.string().min(1)`; same structure as account. | Right sheet. |
| Transaction | Date, account, category, payee, amount, notes | Date required; account/payee non-empty; amount must parse and fit safe milliunits; category/notes optional. Creatable account/category selects. | Right sheet. |
| CSV mapping | One Radix select per column | Amount/date/payee must each be mapped once; Continue disabled until all three exist; parsed row errors listed inline. | Full content card. |

`FormControl` supplies `aria-invalid` and `aria-describedby`; labels and
messages share generated IDs. Invalid primitive styling uses the semantic
destructive border/ring. Auth-level errors and CSV import errors use
`role="alert"`.

## Tables and data display

- `DataTable` uses TanStack Table for filtering, sorting, page-level row
  selection, pagination, and bulk deletion.
- The table wrapper scrolls horizontally. There is no alternate mobile row/card
  representation.
- Accounts and categories expose select, sortable Name, and actions columns.
- Transactions expose select, Date, Category, Payee, Amount, Account, and
  actions. Amount is a pill badge; uncategorized rows are rose and clickable.
- Table cells for account/category are clickable `div` elements that open the
  related edit sheet (or transaction edit when uncategorized).
- Empty filtered/current page state is one full-width “No results.” row.
- CSV import uses the same table primitives but a different header with mapping
  selects and no pagination.
- Summary cards animate values with `react-countup`. Chart switchers select
  Area/Bar/Line and Pie/Radar/Radial representations.

## Overlays

- Right sheets: create/edit account, category, transaction. Right sheets are
  full viewport width until `lg`, where they cap at 28 rem.
- Left sheet: mobile navigation, 75% width and capped at 24 rem from `sm`.
- Center dialogs: generic confirmation and import-account selection; width is
  viewport minus 2 rem, capped at 32 rem from `sm`.
- Popovers: date picker and date-range filter.
- Dropdowns: account menu and per-row action menus.
- Tooltips: amount-sign reversal guidance.
- All Radix overlays use `z-50`, black/50 overlays where modal, and
  fade/zoom/slide state animations from `tw-animate-css`.

## State map

### Loading

- Full viewport: `SessionGuard` (`size-8`, slate-400 spinner).
- Auth bootstrap: 40-unit minimum-height status region; route `Suspense`
  fallbacks use small centered spinners.
- List pages: title skeleton + centered `size-6` spinner in `h-125`.
- Summary: card skeletons; chart areas retain card chrome and use a centered
  spinner in `h-87.5`.
- Edit sheets: absolute full-sheet centered `size-4` spinner.
- Pending forms/actions: controls disabled; auth submit buttons add a spinner.

### Empty

- Data table: “No results.”
- Charts: icon + “No data for this period.”
- CSV without rows: bordered `role=alert` message.
- Missing auth link token: dedicated error outcome panel.

### Error

- Auth forms: field messages plus `AuthError`.
- CSV parse/preparation and mutations: Sonner error toast.
- CSV row validation: inline red alert, first five errors shown.
- CRUD mutation errors: Sonner toasts from API hooks.
- Dashboard query errors have no dedicated visual branch; most resolve visually
  to empty arrays, zero-valued cards, or blank edit-form defaults.

### Interaction

- Buttons, inputs, native primitives: hover, focus-visible ring, disabled
  opacity/cursor/pointer behavior.
- Table sort toggles through header buttons; selection uses checked,
  indeterminate, and selected-row states.
- Cards are static; charts expose select-driven visualization changes and
  hover tooltips.
- Sheets/dialogs/dropdowns/popovers use Radix keyboard and focus behavior.
- Delete operations require confirmation except the transaction edit sheet's
  missing form `id` currently prevents its in-form delete action from
  rendering; the row action still provides deletion.

## Responsive summary

| Threshold/source | Current behavior |
| --- | --- |
| Below 640 px (JS date filter) | One-month date calendar; presets are a separate panel. |
| `sm` = 640 px | Dialog footer becomes a row; dialog caps at 32 rem; left sheet caps at 24 rem. |
| CSS max-width 767 px | All react-select control/value/input/option text is forced to 16 px. |
| `md` = 768 px | Inputs, textarea, amount input, and react-select drop from 16 px to 14 px; calendar months may become a row. |
| JS max-width 1024 px | Mobile navigation sheet is rendered. |
| `lg` = 1024 px | Auth becomes split screen; header logo/desktop nav appear when that component path is rendered; filters/actions become rows; dashboard grids expand; main/header horizontal padding becomes 56 px. |
| `xl` = 1280 px | Overview charts change from equal 3/3 columns to 4/2. |
| `2xl` = 1536 px | Header/page maximum width. |

