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
  if (parsed.username || parsed.password) {
    throw new Error(
      `GO_API_URL must not include username or password credentials, got: ${url}`
    );
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
 * The Go API is the only backend, so the rewrite is unconditional: every
 * `/api/*` request is proxied to it. Only its origin is configurable.
 *
 * The rewrite stays in `beforeFiles` after the removal of
 * `app/api/[[...route]]` in #84. `afterFiles` would also fire now that no
 * filesystem route matches `/api/*`, but `beforeFiles` keeps the proxy
 * authoritative: any future Next route added under `app/api/` would silently
 * shadow the backend from `afterFiles`, whereas from `beforeFiles` the proxy
 * still wins. Verified by running the app with the legacy route deleted.
 *
 * The browser keeps calling same-origin `/api/*`, so cookies, CSRF assumptions
 * and CORS are unchanged and the backend is reachable only through this hop.
 */
export function goApiRewrites(
  env: RewriteEnv = process.env
): NonNullable<Awaited<ReturnType<NonNullable<NextConfig['rewrites']>>>> {
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
