import type { NextConfig } from 'next';

// Relative rather than the `@/` alias: next.config.ts is loaded before the
// tsconfig path aliases are available, so the alias fails to resolve here.
import { goApiRewrites } from './lib/go-api-rewrite';

const nextConfig: NextConfig = {
  /**
   * Same-origin `/api/*` proxying to the Go backend.
   *
   * Unconditional: the Go API is the only backend, so every `/api/*` request is
   * proxied to it and only the origin is configurable. There is no flag — the
   * `USE_GO_API` escape hatch was removed with the Hono surface in #84, because
   * with nothing to fall back to it selected nothing.
   *
   * The rewrite stays in `beforeFiles`, which is now a deliberate choice rather
   * than a forced one: with `app/api/[[...route]]` gone, `afterFiles` would fire
   * too, but a future Next route under `app/api/` would silently shadow the
   * backend from that phase. See lib/go-api-rewrite.ts for the full rationale.
   */
  async rewrites() {
    return goApiRewrites();
  },
};

export default nextConfig;
