import {
  clearAccessToken,
  getAccessToken,
  setAccessToken,
} from '@/lib/api/token-store';

/** Path of the refresh endpoint, relative to the same-origin API root. */
const REFRESH_PATH = '/api/auth/refresh';

/**
 * The contract's carve-out: an otherwise valid but expired access token is
 * reported with this code so a client knows to refresh rather than re-prompt
 * for credentials. Any other 401 means the credentials themselves are no good.
 */
const EXPIRED_CODE = 'ACCESS_TOKEN_EXPIRED';

type Dependencies = {
  fetch?: typeof globalThis.fetch;
  getToken?: () => string | null;
  setToken?: (token: string | null) => void;
  clearToken?: () => void;
  baseUrl?: string;
};

/**
 * Reads the error code out of a response without consuming the caller's copy.
 *
 * Returns null for a body that is missing, not JSON, or not shaped like the
 * contract's error envelope — all of which mean "not the expired case", so the
 * caller should not attempt a refresh.
 */
async function readErrorCode(response: Response): Promise<string | null> {
  try {
    const body = await response.clone().json();
    const code = body?.error?.code;
    return typeof code === 'string' ? code : null;
  } catch {
    return null;
  }
}

/**
 * Builds a fetch that attaches the access token and transparently refreshes it
 * once when the API reports the token expired.
 *
 * Two behaviors are worth stating because they are easy to get wrong:
 *
 * **One refresh at a time.** A page typically fires several queries at once, so
 * an expired token produces several simultaneous 401s. Without sharing, each
 * would start its own refresh; the first would rotate the refresh token and the
 * rest would present the now-consumed one and fail, logging the user out
 * mid-session. A single in-flight promise is shared by every caller instead.
 *
 * **Refresh is attempted once per request.** The retry is not a loop. If the
 * retried request still returns 401 the response is returned as-is, so a
 * genuinely rejected session surfaces rather than spinning.
 */
export function createAuthenticatedFetch({
  fetch: fetchImpl = globalThis.fetch,
  getToken = getAccessToken,
  setToken = setAccessToken,
  clearToken = clearAccessToken,
  baseUrl = '',
}: Dependencies = {}): typeof globalThis.fetch {
  let inFlightRefresh: Promise<boolean> | null = null;

  /**
   * Exchanges the refresh cookie for a new access token. Resolves true when a
   * new token was stored.
   *
   * `credentials: 'include'` is required: the refresh token is an httpOnly
   * cookie, and this is the only request that needs it.
   */
  async function refresh(): Promise<boolean> {
    // A transient network failure must not escape as a thrown refresh. The
    // caller's contract is "return the original 401 if the session cannot be
    // renewed"; letting this reject would instead surface a fetch error to
    // whichever query happened to trigger the refresh, which is both confusing
    // and different from what a plain 401 would have done.
    let response: Response;
    try {
      response = await fetchImpl(`${baseUrl}${REFRESH_PATH}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
      });
    } catch {
      return false;
    }

    if (!response.ok) {
      // The session is over. Clearing here rather than leaving a stale token
      // means the next request fails fast instead of trying to refresh again.
      clearToken();
      return false;
    }

    const body = await response.json().catch(() => null);
    const token = body?.data?.accessToken;
    if (typeof token !== 'string' || token === '') {
      clearToken();
      return false;
    }

    setToken(token);
    return true;
  }

  /** Shares one refresh across every caller that races into it. */
  function refreshOnce(): Promise<boolean> {
    inFlightRefresh ??= refresh().finally(() => {
      inFlightRefresh = null;
    });
    return inFlightRefresh;
  }

  function withAuthorization(init: RequestInit | undefined): RequestInit {
    const token = getToken();
    if (!token) {
      return { ...init, credentials: 'include' };
    }
    const headers = new Headers(init?.headers);
    headers.set('Authorization', `Bearer ${token}`);
    return { ...init, headers, credentials: 'include' };
  }

  return async function authenticatedFetch(input, init) {
    const response = await fetchImpl(input, withAuthorization(init));

    if (response.status !== 401) {
      return response;
    }

    // Only an expired token is refreshable. A bad signature or a missing header
    // is not going to be fixed by rotating, and retrying would just double the
    // failed requests.
    if ((await readErrorCode(response)) !== EXPIRED_CODE) {
      return response;
    }

    // Refreshing the refresh call itself would recurse forever.
    if (typeof input === 'string' && input.endsWith(REFRESH_PATH)) {
      return response;
    }

    if (!(await refreshOnce())) {
      return response;
    }

    return fetchImpl(input, withAuthorization(init));
  };
}
