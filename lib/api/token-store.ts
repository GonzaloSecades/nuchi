import 'client-only';

/**
 * In-memory holder for the current access token.
 *
 * Deliberately memory-only. The refresh token lives in an httpOnly cookie the
 * browser cannot read, and putting the access token in localStorage or a
 * readable cookie would hand any XSS a long-lived credential it could exfiltrate.
 * Keeping it in a closure means a successful XSS still has to act inside
 * the page rather than walking away with the token.
 *
 * The client-only marker is load-bearing: a module-level token in a Next server
 * bundle would be shared by unrelated requests and users.
 *
 * The cost is that a full page load starts with no access token. The fetch
 * wrapper recognizes an unauthenticated protected request, exchanges the
 * refresh cookie, and retries without the user noticing.
 */

export type AccessTokenStore = {
  get: () => string | null;
  set: (token: string | null) => void;
  clear: () => void;
  subscribe: (listener: (token: string | null) => void) => () => void;
};

/**
 * Builds an independent token store.
 *
 * A factory rather than module-level state so tests can take an isolated store
 * per case. The alternative was a `resetAccessTokenStore()` export, which put a
 * test-only function on the production surface where any code could import it
 * and wipe the session — clearing subscribers too, so nothing would even be
 * notified. Isolation by construction removes that hazard instead of
 * documenting around it.
 */
export function createAccessTokenStore(): AccessTokenStore {
  let accessToken: string | null = null;

  /** Listeners notified whenever the token changes, including on clear. */
  const subscribers = new Set<(token: string | null) => void>();

  function set(token: string | null): void {
    accessToken = token;
    for (const notify of subscribers) {
      // One broken listener must not stop the others from being told, and must
      // not fail the caller. This runs inside the refresh path: a subscriber
      // throwing here would turn a successful token renewal into a failed
      // request.
      try {
        notify(token);
      } catch (error) {
        // Gated to development, matching logError in lib/api-error.ts. A buggy
        // UI listener should not write to every user's console in production,
        // and this is where real error reporting would be wired instead.
        if (process.env.NODE_ENV === 'development') {
          console.error('[api] access token subscriber threw', error);
        }
      }
    }
  }

  return {
    get: () => accessToken,
    set,
    clear: () => set(null),
    /**
     * Subscribes to token changes and returns an unsubscribe function.
     *
     * Exists so UI can react to the session ending — a refresh failing
     * mid-session clears the token, and something has to send the user to
     * sign-in. The auth pages that consume this arrive in #51.
     */
    subscribe: (listener) => {
      subscribers.add(listener);
      return () => {
        subscribers.delete(listener);
      };
    },
  };
}

/** The store the application uses. */
export const accessTokenStore = createAccessTokenStore();

export const getAccessToken = accessTokenStore.get;
export const setAccessToken = accessTokenStore.set;
export const clearAccessToken = accessTokenStore.clear;
export const subscribeToAccessToken = accessTokenStore.subscribe;
