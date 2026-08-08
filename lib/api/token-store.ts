/**
 * In-memory holder for the current access token.
 *
 * Deliberately memory-only. The refresh token lives in an httpOnly cookie the
 * browser cannot read, and putting the access token in localStorage or a
 * readable cookie would hand any XSS a long-lived credential it could exfiltrate.
 * Keeping it in a module variable means a successful XSS still has to act inside
 * the page rather than walking away with the token.
 *
 * The cost is that a full page load starts with no access token. That is fine:
 * the refresh cookie is sent automatically, so the first 401 triggers a refresh
 * and the session resumes without the user noticing.
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
    notify(token);
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
