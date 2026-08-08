import type { NextConfig } from 'next';

// Relative rather than the `@/` alias: next.config.ts is loaded before the
// tsconfig path aliases are available, so the alias fails to resolve here.
import { goApiRewrites } from './lib/go-api-rewrite';

const nextConfig: NextConfig = {
  /**
   * Same-origin `/api/*` proxying to the Go backend.
   *
   * Off unless USE_GO_API is "true", so the legacy Hono routes keep serving
   * `/api/*` today. See lib/go-api-rewrite.ts for why enabling it is
   * all-or-nothing and why the rewrite must live in beforeFiles.
   */
  async rewrites() {
    return goApiRewrites();
  },
};

export default nextConfig;
