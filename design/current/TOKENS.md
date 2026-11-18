# Current implemented tokens

## Sources and precedence

The application has no single token file. Values reach the UI through four
layers, all preserved here:

1. Tailwind CSS v4 defaults imported by `@import "tailwindcss"` in
   `app/globals.css`.
2. Light and dark semantic CSS variables declared in `app/globals.css`, then
   exposed to Tailwind through `@theme inline` aliases such as
   `--color-background: var(--background)`.
3. Tailwind palette utilities and arbitrary values embedded in component JSX.
4. Inline JavaScript/SVG styles in `components/select.tsx` and the Recharts
   components.

`components.json` identifies the component baseline as shadcn “new-york,” with
CSS variables enabled, slate as the base palette, and Lucide icons. There is no
Tailwind configuration file. A dark token set exists, but the current root
layout mounts neither a theme provider nor a theme switcher.

## Color tokens

### Semantic light theme (`:root`)

| Token | Actual value | Observed role |
| --- | --- | --- |
| `--background` | `oklch(1 0 0)` | Page/background surfaces. |
| `--foreground` | `oklch(0.129 0.042 264.695)` | Default text. |
| `--card` | `oklch(1 0 0)` | Card surface. |
| `--card-foreground` | `oklch(0.129 0.042 264.695)` | Card text. |
| `--popover` | `oklch(1 0 0)` | Popover/menu/dialog surface. |
| `--popover-foreground` | `oklch(0.129 0.042 264.695)` | Overlay text. |
| `--primary` | `oklch(0.208 0.042 265.755)` | Default buttons and strong actions. |
| `--primary-foreground` | `oklch(0.984 0.003 247.858)` | Text on primary. |
| `--secondary` | `oklch(0.968 0.007 247.896)` | Secondary controls. |
| `--secondary-foreground` | `oklch(0.208 0.042 265.755)` | Secondary text. |
| `--muted` | `oklch(0.968 0.007 247.896)` | Muted surfaces. |
| `--muted-foreground` | `oklch(0.554 0.046 257.417)` | Muted/supporting text. |
| `--accent` | `oklch(0.968 0.007 247.896)` | Hover/selected neutral surface. |
| `--accent-foreground` | `oklch(0.208 0.042 265.755)` | Text on accent. |
| `--destructive` | `oklch(0.577 0.245 27.325)` | Destructive action/error. |
| `--border` | `oklch(0.929 0.013 255.508)` | Default borders. |
| `--input` | `oklch(0.929 0.013 255.508)` | Input borders/background alias. |
| `--ring` | `oklch(0.704 0.04 256.788)` | Focus ring. |
| `--chart-1` | `oklch(0.646 0.222 41.116)` | Declared chart token; not consumed by current Recharts code. |
| `--chart-2` | `oklch(0.6 0.118 184.704)` | Declared, not consumed. |
| `--chart-3` | `oklch(0.398 0.07 227.392)` | Declared, not consumed. |
| `--chart-4` | `oklch(0.828 0.189 84.429)` | Declared, not consumed. |
| `--chart-5` | `oklch(0.769 0.188 70.08)` | Declared, not consumed. |

The light sidebar tokens are also declared even though no sidebar primitive
uses them: `--sidebar` and `--sidebar-primary-foreground` are
`oklch(0.984 0.003 247.858)`; `--sidebar-foreground` is
`oklch(0.129 0.042 264.695)`; `--sidebar-primary` and
`--sidebar-accent-foreground` are `oklch(0.208 0.042 265.755)`;
`--sidebar-accent` is `oklch(0.968 0.007 247.896)`;
`--sidebar-border` is `oklch(0.929 0.013 255.508)`; and `--sidebar-ring` is
`oklch(0.704 0.04 256.788)`.

### Semantic dark theme (`.dark`)

| Token | Actual value |
| --- | --- |
| `--background` | `oklch(0.129 0.042 264.695)` |
| `--foreground` | `oklch(0.984 0.003 247.858)` |
| `--card`, `--popover` | `oklch(0.208 0.042 265.755)` |
| `--card-foreground`, `--popover-foreground` | `oklch(0.984 0.003 247.858)` |
| `--primary` | `oklch(0.929 0.013 255.508)` |
| `--primary-foreground` | `oklch(0.208 0.042 265.755)` |
| `--secondary`, `--muted`, `--accent` | `oklch(0.279 0.041 260.031)` |
| `--secondary-foreground`, `--accent-foreground` | `oklch(0.984 0.003 247.858)` |
| `--muted-foreground` | `oklch(0.704 0.04 256.788)` |
| `--destructive` | `oklch(0.704 0.191 22.216)` |
| `--border` | `oklch(1 0 0 / 10%)` |
| `--input` | `oklch(1 0 0 / 15%)` |
| `--ring` | `oklch(0.551 0.027 264.364)` |
| `--chart-1` | `oklch(0.488 0.243 264.376)` |
| `--chart-2` | `oklch(0.696 0.17 162.48)` |
| `--chart-3` | `oklch(0.769 0.188 70.08)` |
| `--chart-4` | `oklch(0.627 0.265 303.9)` |
| `--chart-5` | `oklch(0.645 0.246 16.439)` |

Dark sidebar values are `--sidebar: oklch(0.208 0.042 265.755)`,
`--sidebar-foreground: oklch(0.984 0.003 247.858)`,
`--sidebar-primary: oklch(0.488 0.243 264.376)`,
`--sidebar-primary-foreground: oklch(0.984 0.003 247.858)`,
`--sidebar-accent: oklch(0.279 0.041 260.031)`,
`--sidebar-accent-foreground: oklch(0.984 0.003 247.858)`,
`--sidebar-border: oklch(1 0 0 / 10%)`, and
`--sidebar-ring: oklch(0.551 0.027 264.364)`.

### Direct Tailwind palette values actually used

| Utility family | Tailwind v4 value | Current use |
| --- | --- | --- |
| `blue-500` | `oklch(62.3% 0.214 259.815)` | Auth brand panel, header gradient end, primary badge and chart-adjacent UI. |
| `blue-600` | `oklch(54.6% 0.245 262.881)` | Hover/action blue. |
| `blue-700` | `oklch(48.8% 0.243 264.376)` | Header gradient start. |
| `emerald-500` | `oklch(69.6% 0.17 162.48)` | Income/positive state. |
| `emerald-600` | `oklch(59.6% 0.145 163.225)` | Income hover. |
| `rose-50` | `oklch(96.9% 0.015 12.422)` | Destructive/expense tint. |
| `rose-500` | `oklch(64.5% 0.246 16.439)` | Expense/negative state. |
| `rose-600` | `oklch(58.6% 0.253 17.585)` | Expense hover. |
| `rose-700` | `oklch(51.4% 0.222 16.935)` | Strong expense text. |
| `red-50` | `oklch(97.1% 0.013 17.38)` | Auth and CSV error surface. |
| `red-200` | `oklch(88.5% 0.062 18.334)` | Error border. |
| `red-700` | `oklch(50.5% 0.213 27.518)` | Error text/icon. |
| `slate-300` | `oklch(86.9% 0.022 252.894)` | Light neutral chrome. |
| `slate-400` | `oklch(70.4% 0.04 256.788)` | Spinner and neutral amount button. |
| `slate-500` | `oklch(55.4% 0.046 257.417)` | Neutral hover/text. |
| `white` | `#fff` | Header/auth brand text, icons, chart tooltips. |
| `black` | `#000` | Modal overlays at 50%; inherited Tailwind black where specified. |

### Hard-coded color values outside the token aliases

| Value | Source/use |
| --- | --- |
| `#2E2A47` | Auth heading/strong text. |
| `#7E8CA0` | Auth supporting text. |
| `#89b6fd` | Dashboard welcome/date text. |
| `#e2e8f0` | React Select border and hover border. |
| `#3d82f6` | Area/bar/line chart stroke/fill. |
| `#3b82f6` | Radar chart stroke/fill. |
| `#f43f5e` | Negative/expense chart value. |
| `#0062FF`, `#12C6FF`, `#FF647F`, `#FF9354` | Category chart palette. |
| `#8884d8` | Additional chart color. |

The last three groups coexist with semantic `--primary`, `--destructive`, and
`--chart-*`; they are not aliases of those tokens.

## Typography

### Families

- `Geist Sans` is loaded with `next/font/google` and assigned to
  `--font-geist-sans`; `@theme inline` maps `--font-sans` to it.
- `Geist Mono` is loaded and assigned to `--font-geist-mono`; `--font-mono`
  maps to it.
- The body receives both font variables and `antialiased`. No explicit
  `font-sans` class is needed because Tailwind preflight uses the theme family.
- No frontend component currently requests `font-mono` directly.

### Sizes and line heights used

| Class | Size | Default line height | Examples |
| --- | --- | --- | --- |
| `text-xs` | 0.75 rem / 12 px | 1 rem / 16 px | Help text, badges, table metadata. |
| `text-sm` | 0.875 rem / 14 px | 1.25 rem / 20 px | Controls, labels, body copy. |
| `text-base` | 1 rem / 16 px | 1.5 rem / 24 px | Mobile inputs and select content. |
| `text-lg` | 1.125 rem / 18 px | 1.75 rem / 28 px | Card/display accents. |
| `text-xl` | 1.25 rem / 20 px | 1.75 rem / 28 px | Screen/card titles. |
| `text-2xl` | 1.5 rem / 24 px | 2 rem / 32 px | Auth/status headings and summary values. |
| `text-3xl` | 1.875 rem / 30 px | 2.25 rem / 36 px | Large dashboard figures/heading use. |
| `text-4xl` | 2.25 rem / 36 px | 2.5 rem / 40 px | Largest auth/status display use. |

`leading-none` (`1`) is also used for compact titles. The implemented weights
are 400 (`font-normal` or inherited), 500 (`font-medium`), 600
(`font-semibold`), and 700 (`font-bold`). `tracking-widest` is `0.1em` and is
used where explicitly declared. There is no separate typographic scale file.

## Spacing and sizing

Tailwind’s base spacing unit is `0.25rem` (4 px). These scale values appear in
the current JSX; the table preserves both ordinary and nonstandard fractional
uses:

| Scale | rem | px | Representative roles |
| --- | ---: | ---: | --- |
| `0.5` | 0.125 | 2 | Fine gaps/padding. |
| `1` | 0.25 | 4 | Icon gaps, compact offsets. |
| `1.5` | 0.375 | 6 | Amount icon inset, small gaps. |
| `2` | 0.5 | 8 | Common gaps, compact padding. |
| `2.5` | 0.625 | 10 | Compact spacing. |
| `3` | 0.75 | 12 | Default page padding below `lg`, control padding. |
| `3.5` | 0.875 | 14 | Icon/control dimension. |
| `4` | 1 | 16 | Common gaps/padding. |
| `5` | 1.25 | 20 | Gaps and icon sizes. |
| `6` | 1.5 | 24 | Card gaps/padding. |
| `7` | 1.75 | 28 | Header/action spacing. |
| `8` | 2 | 32 | Section spacing/spinners. |
| `9` | 2.25 | 36 | Default control height. |
| `10` | 2.5 | 40 | Large control height. |
| `12` | 3 | 48 | Vertical sections and actions. |
| `14` | 3.5 | 56 | Dashboard padding from `lg`. |
| `16` | 4 | 64 | Textarea minimum height and large spacing. |
| `24` | 6 | 96 | Negative dashboard overlap (`-mt-24`). |
| `30` | 7.5 | 120 | Bounded status/empty areas. |
| `36` | 9 | 144 | Header bottom padding. |
| `40` | 10 | 160 | Auth bootstrap minimum height. |
| `42.5` | 10.625 | 170 | Arbitrary fixed dimension. |
| `48` | 12 | 192 | Large layout dimension. |
| `56` | 14 | 224 | Large layout dimension. |
| `72` | 18 | 288 | Large layout dimension. |
| `75` | 18.75 | 300 | Chart height. |
| `87.5` | 21.875 | 350 | Chart/loading height. |
| `105` | 26.25 | 420 | Large bounded content. |
| `125` | 31.25 | 500 | List loading panel height. |

Additional sizing forms include `px` (1 px), fractions (`1/2`, `3/4`),
viewport values (`min-h-screen`, `w-screen`), `full`, `fit`, `min`, and
arbitrary viewport calculations such as dialog width minus 2 rem. Layout caps
come from Tailwind containers: `max-w-md` 28 rem, `max-w-lg` 32 rem,
`max-w-sm` 24 rem, and screen/2xl caps of 96 rem.

## Radii

The application overrides the shadcn radius base with
`--radius: 0.625rem` (10 px). Inline theme aliases calculate:

| Token/class | Actual value |
| --- | --- |
| `--radius-sm` / `rounded-sm` | 0.375 rem / 6 px |
| `--radius-md` / `rounded-md` | 0.5 rem / 8 px |
| `--radius-lg` / `rounded-lg` | 0.625 rem / 10 px |
| `--radius-xl` / `rounded-xl` | 0.875 rem / 14 px |
| `--radius-2xl` / `rounded-2xl` | 1.125 rem / 18 px |
| `--radius-3xl` / `rounded-3xl` | 1.375 rem / 22 px |
| `--radius-4xl` / `rounded-4xl` | 1.625 rem / 26 px |

Tailwind’s independent `rounded-xs` (0.125 rem / 2 px), `rounded-full`, and an
arbitrary 2 px radius also appear. Controls mostly use `rounded-md`; cards use
`rounded-xl`; badges/avatars use `rounded-full`; the 16 px checkbox uses
`rounded-lg`.

## Shadows

| Token/class | Actual value | Current role |
| --- | --- | --- |
| `shadow-xs` | `0 1px 2px 0 rgb(0 0 0 / 0.05)` | Inputs and outline controls. |
| `shadow-sm` | `0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)` | Base card shadow and small overlays. |
| `shadow-md` | `0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)` | Overlay/elevated content where requested. |
| `shadow-lg` | `0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)` | Higher overlay elevation where requested. |
| `drop-shadow-sm` | `0 1px 2px rgb(0 0 0 / 0.15)` | Dashboard screen cards layered over the header. |

Some dashboard cards combine the base `shadow-sm` box shadow with a screen-level
`drop-shadow-sm` filter.

## Borders and focus treatment

- Default border width is 1 px (`border`); dividers use `border-t`/`border-b`.
- The default color is `--border`; form controls use `--input`; invalid
  controls use `--destructive`.
- Most primitives use a 3 px `focus-visible` ring at 50% of `--ring` plus a
  focus border.
- `AmountInput` instead uses a 2 px ring plus a 2 px ring offset.
- Header/navigation overrides use semi-transparent white borders/backgrounds
  and locally replace or suppress default rings.
- Dashed borders are used for certain import/empty drop-zone presentations.

## Breakpoints and responsive values

| Name/source | Actual value | Current behavior |
| --- | ---: | --- |
| `sm` Tailwind | 40 rem / 640 px | Dialog/footer and sheet width changes. |
| Date-filter JS query | `max-width: 640px` | Mobile one-month/presets-panel branch includes exactly 640 px. |
| React Select CSS query | `max-width: 767px` | Forces select control/value/input/option text to 16 px. |
| `md` Tailwind | 48 rem / 768 px | Inputs/textarea/custom amount/select move from 16 px to 14 px; calendar layout changes. |
| `lg` Tailwind | 64 rem / 1024 px | Auth split, header/nav visibility, page padding and grid expansion. |
| Navigation JS query | `max-width: 1024px` | Mobile sheet branch includes exactly 1024 px. |
| `xl` Tailwind | 80 rem / 1280 px | Overview chart proportions change. |
| `2xl` Tailwind | 96 rem / 1536 px | Header and dashboard content cap. |

The inclusive JS `max-width` checks and inclusive Tailwind `min-width`
breakpoints overlap at exactly 640 px and 1024 px.

## Z-index and layering

- `z-50`: dialogs, sheets, dropdown menus, popovers, select content,
  tooltips, and their modal overlays.
- `z-10`: calendar focus/selected-day stacking.
- No broader z-index scale or named layering variables exist.
- Dashboard content overlap is created geometrically with `-mt-24`, not with a
  dedicated z-index token.

## Icons and motion

- Icons come from `lucide-react`; primitive defaults normalize unspecified SVGs
  to 1 rem, while call sites also use 12, 14, 16, 20, 24, and 32 px sizes.
- `public/logo.svg` has a 56 by 40 view box and a white mark. It is rendered at
  200 by 200 in the auth shell and 28 by 28 in the dashboard header.
- Loading indicators use `Loader2` with `animate-spin`; skeletons use
  `animate-pulse`.
- Radix overlays use fade, zoom, and directional slide animations supplied by
  `tw-animate-css`. Buttons use `transition-all`; inputs transition color and
  shadow. No centralized duration/easing tokens are declared.

