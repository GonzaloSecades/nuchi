import 'client-only';

/**
 * In-memory holder for the current access token.
 *
 * Deliberately memory-only. The refresh token lives in an httpOnly cookie the
 * browser cannot read, and putting the access token in localStorage or a
 * readable cookie would hand any XSS a long-lived credential it could exfiltrate.
 * Keeping it in a module variable means a successful XSS still has to act inside
 * the page rather than walking away with the token.
 *
 * The client-only marker is load-bearing: a module-level token in a Next server
 * bundle would be shared by unrelated requests and users.
 *
 * The cost is that a full page load starts with no access token. The fetch
 * wrapper recognizes an unauthenticated protected request, exchanges the
 * refresh cookie, and retries without the user noticing.
 */

let accessToken: string | null = null;

/** Listeners notified whenever the token changes, including on clear. */
const subscribers = new Set<(token: string | null) => void>();

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
  for (const notify of subscribers) {
    // One broken listener must not stop the others from being told, and must
    // not fail the caller. This runs inside the refresh path: a subscriber
    // throwing here would turn a successful token renewal into a failed
    // request.
    try {
      notify(token);
    } catch (error) {
      console.error('[api] access token subscriber threw', error);
    }
  }
}

export function clearAccessToken(): void {
  setAccessToken(null);
}

/**
 * Subscribes to token changes and returns an unsubscribe function.
 *
 * Exists so UI can react to the session ending — a refresh failing mid-session
 * clears the token, and something has to send the user to sign-in. The auth
 * pages that consume this arrive in #51.
 */
export function subscribeToAccessToken(
  listener: (token: string | null) => void
): () => void {
  subscribers.add(listener);
  return () => {
    subscribers.delete(listener);
  };
}

/** Test-only reset so one test's token cannot leak into the next. */
export function resetAccessTokenStore(): void {
  accessToken = null;
  subscribers.clear();
}
