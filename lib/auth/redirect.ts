/** Where a signed-in user lands when no specific destination was requested. */
export const DEFAULT_SIGNED_IN_PATH = '/';

/** Builds the exact in-app location the session guard should restore. */
export function redirectTargetFromLocation(location: {
  pathname: string;
  search: string;
  hash: string;
}): string {
  return `${location.pathname}${location.search}${location.hash}`;
}

/**
 * Validates a post-sign-in destination taken from the URL.
 *
 * The guard sends users to `/sign-in?redirect=<where they were headed>`, and
 * that parameter is attacker-controllable: a link to
 * `/sign-in?redirect=https://evil.example/login` would, unchecked, bounce a
 * user who just authenticated onto a convincing fake — the classic open
 * redirect, and a good phishing primitive precisely because the first hop is
 * the real site.
 *
 * Only a path on this origin is accepted, which means:
 *
 * - it must start with a single `/`. `//evil.example` is protocol-relative and
 *   navigates off-site despite looking like a path, so a second leading slash
 *   is rejected too;
 * - it must not be a full URL. `https:/evil.example` and backslash variants
 *   are excluded by the same rule, since neither begins with `/` followed by a
 *   non-slash character;
 * - `/sign-in` and the other auth routes are rejected, so a stale `redirect`
 *   cannot loop the user back to the page they just left.
 */
export function safeRedirectTarget(raw: string | null | undefined): string {
  if (
    !raw ||
    !raw.startsWith('/') ||
    raw.startsWith('//') ||
    raw.startsWith('/\\')
  ) {
    return DEFAULT_SIGNED_IN_PATH;
  }

  const path = raw.split(/[?#]/, 1)[0];
  if (AUTH_PATHS.some((authPath) => path === authPath)) {
    return DEFAULT_SIGNED_IN_PATH;
  }

  return raw;
}

/**
 * Gives signed-in visitors to the sign-in page a safe route back into the app.
 *
 * The session guard normally waits for bootstrap before it can redirect. This
 * second check is intentional recovery: if navigation has already reached
 * sign-in while the refresh request is still resolving, a successful refresh
 * must not leave the user stranded on a form for a session they already have.
 */
export function restoredSessionRedirectTarget(
  status: 'loading' | 'authenticated' | 'unauthenticated',
  requestedTarget: string | null | undefined
): string | null {
  return status === 'authenticated'
    ? safeRedirectTarget(requestedTarget)
    : null;
}

/**
 * The pages that exist only for signed-out users.
 *
 * Used both to reject them as redirect targets and, by the guard, to decide
 * which routes need no session.
 */
export const AUTH_PATHS = [
  '/sign-in',
  '/sign-up',
  '/verify-email',
  '/forgot-password',
  '/reset-password',
] as const;

/** Whether a pathname is one of the signed-out-only auth pages. */
export function isAuthPath(pathname: string): boolean {
  return AUTH_PATHS.some((authPath) => pathname === authPath);
}
