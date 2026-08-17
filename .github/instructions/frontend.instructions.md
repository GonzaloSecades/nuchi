---
applyTo: 'app/**/*.ts,app/**/*.tsx,features/**/*.ts,features/**/*.tsx,components/**/*.ts,components/**/*.tsx,lib/**/*.ts,lib/**/*.tsx'
---

Review frontend changes as a Next.js App Router client over the generated
OpenAPI client. There is no Next API layer and no frontend database access.
Browser requests should stay same-origin through `/api/*`; do not introduce
absolute browser API origins unless the existing API base URL helper already
supports the case.

Feature code belongs in `features/<domain>/`. Server state belongs in TanStack
Query hooks, and API calls should use `lib/api/` rather than ad hoc `fetch`.
Prefer existing `components/ui/*` primitives.

Money amounts are signed integer milliunits. Dates sent to transaction APIs are
plain `yyyy-MM-dd` values; avoid timezone shifts from Date serialization.

Do not flag formatting/import order or generated files under `lib/api/generated/`.

Relevant validation is `bun run lint`, `bunx tsc --noEmit`, `bun test`, and
`bun run build`.
