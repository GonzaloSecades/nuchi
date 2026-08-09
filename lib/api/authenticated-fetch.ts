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
 * for credentials. The only other refreshable 401 is a missing-token
 * UNAUTHORIZED response after a full page reload.
 */
const EXPIRED_CODE = 'ACCESS_TOKEN_EXPIRED';
const UNAUTHORIZED_CODE = 'UNAUTHORIZED';

/**
 * The callable surface of `fetch`, without the runtime statics that a type
 * package may hang off `typeof globalThis.fetch` — Bun's types, pulled in for
 * `bun:test`, declare a `fetch.preconnect`. Only the call signature is ever
 * used here, and a plain function this module builds cannot supply statics, so
 * annotating the result as `typeof globalThis.fetch` is both wrong and
 * unsatisfiable.
 */
type FetchLike = (
  input: RequestInfo | URL,
  init?: RequestInit
) => Promise<Response>;

type Dependencies = {
  fetch?: FetchLike;
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
}: Dependencies = {}): FetchLike {
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

    // Auth responses are deliberately un-enveloped: the contract's
    // AuthSessionResponse is `{ accessToken, expiresIn, tokenType, user }` at
    // the top level, and says so — only the app *resource* operations use
    // `{ data: ... }`. Reading `body.data.accessToken` here found undefined on
    // every real refresh, so the token was cleared and the session dropped at
    // the first expiry, with the page-reload bootstrap never resuming at all.
    const body = await response.json().catch(() => null);
    const token = body?.accessToken;
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

  function withAuthorization(
    input: RequestInfo | URL,
    init: RequestInit | undefined
  ): RequestInit {
    // openapi-fetch supplies a Request whose generated Content-Type and
    // operation headers live on input, normally with no second init argument.
    // Preserve those headers, then apply explicit init overrides before adding
    // the current token.
    const headers = new Headers(
      input instanceof Request ? input.headers : undefined
    );
    new Headers(init?.headers).forEach((value, name) =>
      headers.set(name, value)
    );

    const token = getToken();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    return { ...init, headers, credentials: 'include' };
  }

  function requestPath(input: RequestInfo | URL): string {
    const value =
      input instanceof Request
        ? input.url
        : input instanceof URL
          ? input.href
          : input;
    try {
      return new URL(value, 'http://same-origin.invalid').pathname;
    } catch {
      return '';
    }
  }

  function isAuthRequest(input: RequestInfo | URL): boolean {
    return requestPath(input).startsWith('/api/auth/');
  }

  return async function authenticatedFetch(input, init) {
    // Clone before native fetch consumes the body. The clone is reserved for
    // the one permitted retry and is especially important for generated POST,
    // PATCH, and bulk mutation Requests.
    const retryInput = input instanceof Request ? input.clone() : input;
    const startedWithoutToken = getToken() === null;
    const firstInit = withAuthorization(input, init);
    const startedWithoutAuthorization = !new Headers(firstInit.headers).has(
      'Authorization'
    );
    const response = await fetchImpl(input, firstInit);

    if (response.status !== 401) {
      return response;
    }

    // Auth operations use either no authentication or the refresh cookie; they
    // must never trigger this resource-request refresh path.
    if (isAuthRequest(input)) {
      return response;
    }

    const errorCode = await readErrorCode(response);
    const canBootstrapSession =
      startedWithoutToken &&
      startedWithoutAuthorization &&
      errorCode === UNAUTHORIZED_CODE;

    // An expired token is refreshable. A missing token is also refreshable for
    // the first protected request after a page load, when only the httpOnly
    // refresh cookie survives. Other 401s indicate rejected credentials.
    if (errorCode !== EXPIRED_CODE && !canBootstrapSession) {
      return response;
    }

    if (!(await refreshOnce())) {
      return response;
    }

    return fetchImpl(retryInput, withAuthorization(retryInput, init));
  };
}
