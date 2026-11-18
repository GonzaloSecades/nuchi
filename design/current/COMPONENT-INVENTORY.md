# Current component inventory

## Scope and notation

This inventory describes reusable frontend components in the current working
tree. Paths are repository-relative. “Tailwind” means utility classes in JSX;
“semantic variables” means the OKLCH variables and aliases in
`app/globals.css`. Radix components also expose `data-state`, focus management,
and keyboard behavior. A component marked unused exists in the source tree but
has no current application call site.

## Primitives

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `Button` / `buttonVariants` | `components/ui/button.tsx` | Variants: default, destructive, outline, secondary, ghost, link. Sizes: default, xs, sm, lg, icon, icon-xs, icon-sm, icon-lg. Hover, focus-visible, invalid, disabled, dark; optional `asChild`. | CVA + Tailwind + semantic variables. | Auth actions, list actions, table controls, overlays, settings, filters, calendars, sheets. |
| `Badge` | `components/ui/badge.tsx` | Default, secondary, destructive, outline, ghost, link, and app-added `primary`; focus/invalid/link hover; optional `asChild`. | CVA + Tailwind; `primary` directly uses blue-500. | Transaction amount pills use additional local rose/emerald classes. |
| `Input` | `components/ui/input.tsx` | Text/file input; placeholder, selection, focus-visible, invalid, disabled, dark. 16 px text below `md`, 14 px from `md`. | Tailwind + semantic variables. | Auth, account/category forms, payee field, data-table filter. |
| `Textarea` | `components/ui/textarea.tsx` | Auto content sizing, 4 rem minimum, focus-visible, invalid, disabled, dark; same responsive type shift as `Input`. | Tailwind + semantic variables. | Transaction notes. |
| `Label` | `components/ui/label.tsx` | Disabled peer/group state. | CVA + Tailwind. | React Hook Form labels and direct labels. |
| `Form` family | `components/ui/form.tsx` | `FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormDescription`, `FormMessage`; invalid state and generated ARIA relationships. | React Hook Form context + Radix label/slot + Tailwind. | All auth forms and all CRUD forms. |
| `Card` family | `components/ui/card.tsx` | Card, header, title, description, action, content, footer; optional border-aware header/footer spacing. | Tailwind + semantic variables; base `rounded-xl border shadow-sm`. | Every dashboard screen, summary metrics, charts, import, settings. |
| `Table` family | `components/ui/table.tsx` | Table, header, body, footer, row, head, cell, caption; selected-row and hover states. Wrapper has horizontal overflow. | Semantic HTML + Tailwind + semantic variables. | Generic `DataTable` and CSV mapping table. |
| `Checkbox` | `components/ui/checkbox.tsx` | Checked, indeterminate, focus-visible, invalid, disabled. | Radix Checkbox + Tailwind + semantic variables. | Page/row selection in all data tables. |
| `Select` family (Radix) | `components/ui/select.tsx` | Trigger, value, scroll buttons, content, viewport, group, label, item, separator; open/closed, checked, disabled, side-aware animation. | Radix Select + Tailwind + semantic variables. | Header account filter, chart type switches, CSV column mapping. |
| `Calendar` | `components/ui/calendar.tsx` | Single/range use through props; today, selected range start/middle/end, outside, disabled, hidden; month/year captions and navigation. | `react-day-picker` + `Button` variants + Tailwind + semantic variables. | Date picker and header date-range filter. |
| `Separator` | `components/ui/separator.tsx` | Horizontal or vertical; decorative by default. | Radix Separator + Tailwind. | Chart tooltip content and category tooltip. |
| `Skeleton` | `components/ui/skeleton.tsx` | Pulse only; dimensions supplied by caller. | Tailwind `animate-pulse`, semantic accent. | List-page titles and overview summary cards. |
| `VisuallyHidden` | `components/ui/visually-hidden.tsx` | Hides content visually while retaining it for assistive technology. | Radix VisuallyHidden. | Mobile navigation sheet title. |

## Navigation

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `Header` | `components/header.tsx` | Responsive logo/nav/filter composition; blue gradient background; stacked below `lg`, row layout at `lg`. | Tailwind with direct blue palette utilities. | Dashboard shell only. |
| `HeaderLogo` | `components/header-logo.tsx` | White logo plus “Nuchi”; hidden below `lg`. | Next Image + Tailwind. | `Header`. |
| `Navigation` | `components/navigation.tsx` | Desktop horizontal navigation or mobile left sheet selected with a JS media query; active/inactive route state. | `useMedia`, Radix Sheet, Tailwind. | `Header`. |
| `NavButton` | `components/nav-button.tsx` | Active, inactive, mobile, disabled-through-button behavior; renders a Next link through `asChild`. | `Button` with local Tailwind overrides. | Every dashboard route in `Navigation`. |
| `UserButton` | `components/auth/user-button.tsx` | Circular initial trigger; dropdown open/closed; logout pending; error toast on confirmed logout failure. | Radix Dropdown Menu + Tailwind + Sonner. | `Header`. |
| `WelcomeMsg` | `components/welcome-msg.tsx` | Static greeting plus formatted current date. | Tailwind; direct `#89b6fd` text color. | `Header`. |

## Forms

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `Select` (creatable) | `components/select.tsx` | Selected, empty, disabled, creatable/non-creatable, locally created option, async create result. | `react-select/creatable`; Tailwind outer class; inline style object hard-codes `#e2e8f0`; global mobile CSS targets `.nuchi-select__*`. | Transaction account/category fields and account-selection dialog. |
| `AmountInput` | `components/amount-input.tsx` | Neutral, positive/income, negative/expense, disabled currency field; sign-reversal icon and tooltip. | `react-currency-input-field` + Tailwind; direct slate/emerald/rose utilities. | Transaction form. |
| `DatePicker` | `components/date-picker.tsx` | Empty/selected date, disabled, popover open/closed. | Radix Popover + `Calendar` + `Button` + date-fns. | Transaction form. |
| `AccountFilter` | `components/account-filter.tsx` | All-accounts or selected account; disabled while account data is pending. | Radix Select + URL search parameters + Tailwind. | Dashboard header filters. |
| `DateFilter` | `components/date-filter.tsx` | Draft/applied range, presets, custom range, apply/reset disabled, mobile calendar/preset subpanels, popover state. | Radix Popover + Calendar + JS media query + Tailwind. | Dashboard header filters. |
| `Filters` | `components/filters.tsx` | Responsive container; no independent state. | Tailwind. | Composes account and date filters in `Header`. |
| `AccountForm` | `features/accounts/components/account-form.tsx` | Create/edit label, pending controls, validation message, optional in-form delete when `id` exists. | React Hook Form + Zod + UI primitives + Tailwind. | New/edit account sheets. |
| `CategoryForm` | `features/categories/components/category-form.tsx` | Same state model as `AccountForm`. | React Hook Form + Zod + UI primitives + Tailwind. | New/edit category sheets. |
| `TransactionForm` | `features/transactions/components/transaction-form.tsx` | Create/edit label driven by `id`, pending/disabled, validation, optional delete; supports creating account/category from a select. | React Hook Form + Zod + reusable form controls + Tailwind. | New/edit transaction sheets. The current edit sheet omits `id`, so the form presents create wording and omits its delete button. |

## Data display

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `DataTable` | `components/data-table.tsx` | Filtered/unfiltered, sorted, selected, empty, paginated, bulk-delete confirmation/pending. | TanStack Table + UI table/input/button primitives + Tailwind. | Accounts, categories, transactions. |
| `DataGrid` | `components/data-grid.tsx` | Loaded three-card grid or three skeleton cards. | CSS grid via Tailwind. | Overview. |
| `DataCard` | `components/data-card.tsx` | Remaining/income/expense coloring supplied by caller; formatted animated value. | Card primitive + Tailwind + `Countup`. | Overview summary. |
| `Countup` | `components/count-up.tsx` | Animated numeric transition; formatting supplied by component code. | `react-countup`; inherits surrounding type. | `DataCard`. |
| `DataCharts` | `components/data-charts.tsx` | Loading pair or loaded chart pair; responsive grid. | Tailwind layout. | Overview. |
| `Chart` / `ChartLoading` | `components/chart.tsx` | Area, bar, line; loading and empty states; local visualization selection. | Card + Radix Select + Recharts + Tailwind. | Overview transaction history. |
| `SpendingPie` / loading counterpart | `components/spending-pie.tsx` | Pie, radar, radial; loading and empty states; local visualization selection. | Card + Radix Select + Recharts + Tailwind. | Overview category spending. |
| Chart variants | `components/area-variant.tsx`, `bar-variant.tsx`, `line-variant.tsx`, `pie-variant.tsx`, `radar-variant.tsx`, `radial-variant.tsx` | Recharts hover/active states; each file fixes its own geometry and colors. | Recharts inline props + hard-coded color constants. | Selected by `Chart` or `SpendingPie`. |
| `CustomTooltip` | `components/custom-tooltip.tsx` | Visible only while active with payload; per-series rows. | Tailwind + Separator; white card chrome. | Area, bar, and line charts. |
| `CategoryTooltip` | `components/category-tooltip.tsx` | Visible only while active; category/value rows. | Tailwind + Separator; white card chrome. | Pie/radar/radial category charts. |

## Feedback

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `AuthError` | `components/auth/auth-error.tsx` | Hidden when no message; visible destructive message with icon and `role=alert`. | Tailwind with direct red palette utilities. | Sign-in, sign-up, forgot/reset password. |
| `AuthHeader` | `components/auth/auth-header.tsx` | Title and optional description. | Tailwind; hard-coded auth foreground colors. | All auth screens. |
| `Toaster` | `components/ui/sonner.tsx` | Sonner success/error/info/loading plus close/action styles; theme requested from `next-themes`. | Sonner CSS-variable mapping + icons. | Mounted globally; mutations, logout, and CSV workflow emit toasts. |
| Spinner usages | Inline across auth pages, `SessionGuard`, dashboard pages, charts, and edit sheets | Full-screen, bounded panel, button-inline, and overlay loading presentations. | Repeated `Loader2 animate-spin` Tailwind strings rather than one component. | All major async flows. |

## Overlays

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| `Dialog` family | `components/ui/dialog.tsx` | Open/closed overlay and content animations; header/footer/title/description; built-in icon close; responsive footer direction. | Radix Dialog + Tailwind + semantic variables. | Generic confirmations and account selection during CSV import. |
| `Sheet` family | `components/ui/sheet.tsx` | Open/closed; top/right/bottom/left variants; modal overlay; icon close. | Radix Dialog + CVA + Tailwind. | Mobile navigation and all create/edit forms. |
| `DropdownMenu` family | `components/ui/dropdown-menu.tsx` | Open/closed, highlighted, checked, disabled, inset, destructive; submenus, radio/checkbox items, shortcuts. | Radix Dropdown Menu + Tailwind. | User menu and row actions; many exported subparts are not otherwise used. |
| `Popover` family | `components/ui/popover.tsx` | Open/closed, side-aware animation, optional anchor. | Radix Popover + Tailwind. | Date picker and date filter. |
| `Tooltip` family | `components/ui/tooltip.tsx` | Delayed open/closed, side-aware animation; provider/trigger/content. | Radix Tooltip + Tailwind. | Amount sign guidance. |
| Confirmation hook | `hooks/use-confirm.tsx` | Open/closed, confirm/cancel, pending, destructive or default action; returns a promise. | Dialog primitives + local state + Tailwind. | Single and bulk delete flows. |

## Layout

| Component | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| Root layout | `app/layout.tsx` | One provider composition. | Next metadata/font plus Tailwind body class. | Every route. |
| Auth layout | `app/(auth)/layout.tsx` | Single-column below `lg`; 50/50 form and brand panel from `lg`. | Tailwind + Next Image. | Five auth routes. |
| Dashboard layout | `app/(dashboard)/layout.tsx` | Session pending, redirecting, authenticated; responsive padding. | `SessionGuard`, `SheetProvider`, Tailwind. | Five dashboard routes. |
| `SessionGuard` | `components/auth/session-guard.tsx` | Loading, unauthenticated redirect, authenticated children. | Session context + router + inline full-viewport spinner. | Dashboard layout. |
| `SheetProvider` | `providers/sheet-provider.tsx` | Always mounts all create/edit sheet controllers. | React composition only. | Dashboard layout. |
| `QueryProvider` | `providers/query-provider.tsx` | One browser QueryClient instance. | TanStack Query; no visible styling. | Root layout. |

## Domain-specific components

| Component/group | Source | Variants and states | Styling mechanism | Current use |
| --- | --- | --- | --- | --- |
| Account list columns/actions | `app/(dashboard)/accounts/columns.tsx`, `actions.tsx` | Select all/row, name sort, row edit/delete, delete pending. | TanStack column definitions + Checkbox/Button/Dropdown. | `/accounts`. |
| Category list columns/actions | `app/(dashboard)/categories/columns.tsx`, `actions.tsx` | Same structure as account list. | Same primitives and Tailwind pattern. | `/categories`. |
| Transaction list columns/actions | `app/(dashboard)/transactions/columns.tsx`, `actions.tsx`, `account-column.tsx`, `category-column.tsx` | Select/sort, formatted date/amount, categorized/uncategorized color, clickable account/category, row edit/delete. | TanStack columns + Badge/Button/Dropdown + local Tailwind. | `/transactions`. |
| Transaction import | `app/(dashboard)/transactions/import-card.tsx`, `import-table.tsx`, `table-head-select.tsx`, `upload-button.tsx` | Upload, mapping, mapped/unmapped, duplicate-mapping prevention, row errors, account dialog, import pending. | Card/Table/Radix Select/Dialog/Button + Tailwind + Sonner. | Alternate `/transactions` screen. |
| Account sheets | `features/accounts/components/new-account-sheet.tsx`, `edit-account-sheet.tsx` | Open/closed, create/edit, loading overlay, pending form. | Sheet + `AccountForm` + Tailwind. | Global sheet provider, triggered by accounts and transaction flows. |
| Category sheets | `features/categories/components/new-category-sheet.tsx`, `edit-category-sheet.tsx` | Same structure as account sheets. | Sheet + `CategoryForm` + Tailwind. | Global sheet provider and category flows. |
| Transaction sheets | `features/transactions/components/new-transaction-sheet.tsx`, `edit-transaction-sheet.tsx` | Open/closed, create/edit, data loading, pending; edit form currently lacks its `id` prop. | Sheet + `TransactionForm` + Tailwind. | Global sheet provider and transaction flows. |
| Settings panel | `features/settings/components/settings.tsx` | Static email display and password-reset link. | Card/Button/Next Link + Tailwind. | `/settings`. |

## Present but not composed as standalone UI

- `components/ui/dropdown-menu.tsx` exports checkbox, radio, group, portal,
  sub-menu, label, separator, and shortcut primitives beyond the simple menus
  currently rendered.
- `components/ui/card.tsx` exports `CardDescription`, `CardAction`, and
  `CardFooter`; current screens primarily use card/header/title/content.
- `components/ui/dialog.tsx` and `components/ui/sheet.tsx` export the full
  primitive families even where only a subset is used.
- `components/ui/label.tsx` is normally reached through the Form abstraction.

