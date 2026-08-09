import type { NextConfig } from 'next';

// Relative rather than the `@/` alias: next.config.ts is loaded before the
// tsconfig path aliases are available, so the alias fails to resolve here.
import { goApiRewrites } from './lib/go-api-rewrite';

const nextConfig: NextConfig = {
  /**
   * Same-origin `/api/*` proxying to the Go backend.
   *
   * On by default after the generated-client/auth cutover. See
   * lib/go-api-rewrite.ts for the explicit diagnostic escape hatch, why the
   * rewrite is all-or-nothing, and why it must live in beforeFiles.
   */
  async rewrites() {
    return goApiRewrites();
  },
};

export default nextConfig;
