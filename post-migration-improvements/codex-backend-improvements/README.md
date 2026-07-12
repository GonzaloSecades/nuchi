# Codex Backend Improvements

This directory is the working design set for the backend optimization project
that starts after the Go migration reaches parity and the legacy backend is
retired. It is deliberately separate from the numbered issue seeds in the
parent directory:

- `post-migration-improvements/NNNN-*.md` records specific behavior or schema
  decisions deferred by the port-not-redesign rule.
- This directory defines the system-wide architecture, quality bars, module
  work, and acceptance gates for improving the resulting Go backend.

The requirements are non-negotiable: every endpoint must be secure,
performance-engineered, robust, well documented, and ready for a future
observability implementation.

## How to use this set

Read in this order:

1. [`00-current-state.md`](00-current-state.md) records the evidence and the
   migration boundary.
2. [`01-target-architecture.md`](01-target-architecture.md) defines the target
   request and dependency flow.
3. [`02-security.md`](02-security.md),
   [`03-performance-and-queries.md`](03-performance-and-queries.md),
   [`04-robustness-and-transactions.md`](04-robustness-and-transactions.md),
   [`05-api-documentation.md`](05-api-documentation.md), and
   [`06-observability-readiness.md`](06-observability-readiness.md) define the
   five quality tracks.
4. [`07-module-improvement-map.md`](07-module-improvement-map.md) applies those
   tracks to auth, accounts, categories, transactions, summary, and platform
   infrastructure.
5. [`08-delivery-roadmap.md`](08-delivery-roadmap.md) turns the design into
   phases, evidence, and release gates.
6. [`templates/`](templates/) contains the required proposal and endpoint
   review templates.

## Decision hierarchy

When sources disagree, use this order:

1. Security invariants and data isolation.
2. The hand-edited OpenAPI contract for observable API behavior.
3. Parity fixtures while migration is active.
4. Go implementation and migrations for current internal behavior.
5. Legacy Hono behavior as an oracle, not as the future architecture.
6. This improvement set for post-migration target decisions.

No optimization may weaken ownership isolation or monetary correctness. A
performance change that alters observable behavior requires an explicit
OpenAPI change, compatibility plan, and acceptance tests.

## Definition of done for an endpoint

An endpoint is not optimized merely because its handler is complete. It is
done only when the endpoint review in
[`templates/endpoint-review.md`](templates/endpoint-review.md) has evidence for
all five quality tracks, generated artifacts are current, tests pass, and any
accepted risk has an owner and expiry date.

## Source set

This plan was derived from:

- the Hono routes under `app/api/[[...route]]/` and shared transaction route
  utilities;
- `db/schema.ts` and current Drizzle access patterns;
- the Go scaffold, migrations, RLS policies, pool, router, and generated strict
  OpenAPI server types under `backend/`;
- `openapi/nuchi.openapi.json` and `openapi/README.md`;
- the Go replacement spec and API parity fixtures in
  `docs/specs/18-go-backend-replacement/`;
- all existing numbered post-migration improvement entries; and
- scoped queries against `graphify-out/graph.json`.

This is a planning artifact, not authorization to redesign behavior during
the parity migration.
