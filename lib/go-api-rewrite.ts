import type { NextConfig } from 'next';

/** Default Go API origin, matching backend/README.md and the Go config default. */
export const DEFAULT_GO_API_URL = 'http://localhost:8080';

/**
 * The environment slice this module reads.
 *
 * The index signature is required, not decorative: without it TypeScript's
 * weak-type check rejects `process.env` as having "no properties in common"
 * with a type whose members are all optional.
 */
type RewriteEnv = {
  GO_API_URL?: string;
  USE_GO_API?: string;
  [key: string]: string | undefined;
};

/**
 * Normalizes the configured backend URL, rejecting anything that would produce
 * a broken rewrite destination.
 *
 * Next appends the matched path to the destination, so a URL carrying a path
 * silently yields `/base/api/...` and surfaces as a puzzling 404 from the Go
 * router rather than as a configuration problem. That fails loudly at build
 * time with the offending value instead.
 *
 * A trailing slash is *normalized*, not rejected: `URL.origin` drops it, and it
 * is the common copy-paste shape rather than a mistake worth failing a build
 * over.
 */
export function normalizeGoApiUrl(url: string): string {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    throw new Error(
      `GO_API_URL must be an absolute URL such as ${DEFAULT_GO_API_URL}, got: ${url}`
    );
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error(`GO_API_URL must use http or https, got: ${url}`);
  }
  if (parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error(
      `GO_API_URL must be origin-only (no path, query, or fragment), got: ${url}`
    );
  }

  return parsed.origin;
}

/**
 * Builds the `/api/*` rewrite set for the given environment.
 *
 * Returns no rewrites unless `USE_GO_API` is exactly `"true"`, so the legacy
 * Hono routes keep serving `/api/*` by default.
 *
 * When enabled the rewrite goes in `beforeFiles`, because that is the only
 * phase where it has any effect: `afterFiles` runs solely when no filesystem
 * route matched, and `app/api/[[...route]]` matches everything under `/api`, so
 * an `afterFiles` entry would never fire. The consequence is that enabling this
 * takes over `/api/*` completely — there is no partial or per-route mixing, and
 * mixing would be incoherent anyway because the two stacks authenticate
 * differently (Clerk vs. the owned JWT).
 *
 * The browser keeps calling same-origin `/api/*` either way, so cookies, CSRF
 * assumptions and CORS are unchanged and the backend is reachable only through
 * this hop.
 */
export function goApiRewrites(
  env: RewriteEnv = process.env
): NonNullable<Awaited<ReturnType<NonNullable<NextConfig['rewrites']>>>> {
  if (env.USE_GO_API !== 'true') {
    return [];
  }

  const origin = normalizeGoApiUrl(env.GO_API_URL ?? DEFAULT_GO_API_URL);

  return {
    beforeFiles: [
      {
        source: '/api/:path*',
        destination: `${origin}/api/:path*`,
      },
    ],
    afterFiles: [],
    fallback: [],
  };
}
