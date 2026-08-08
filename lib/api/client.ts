import createClient from 'openapi-fetch';

import type { paths } from '@/lib/api/generated/schema';
import { createAuthenticatedFetch } from '@/lib/api/authenticated-fetch';
import { resolveApiBaseUrl } from '@/lib/api-base-url';
import { ApiError, type ApiErrorDetails } from '@/lib/api-error';

/**
 * The app-owned wrapper around the generated OpenAPI client.
 *
 * Types come from `lib/api/generated/schema.d.ts`, which is generated from
 * `openapi/nuchi.openapi.json`, so a contract change that breaks a call site is
 * a compile error rather than a runtime surprise. Nothing outside this module
 * should import `openapi-fetch` directly — the auth and refresh behavior lives
 * here, and a client built elsewhere would silently skip it.
 *
 * Requests stay **same-origin**: `resolveApiBaseUrl()` returns `''` in the
 * browser, so calls go to the Next origin and are proxied to Go (#30). Cookies
 * therefore stay first-party and no CORS setup is involved.
 */

/**
 * The origin the API is reached through. `''` in the browser, so requests are
 * same-origin and relative.
 */
const apiOrigin = resolveApiBaseUrl();

/**
 * The base every generated operation is resolved against.
 *
 * The `/api` suffix is required. Contract paths are rooted at `/`
 * (`/accounts`, `/summary`, …) and the `/api` prefix comes from the contract's
 * `servers[].url` — but openapi-fetch does not read `servers`, it only prepends
 * `baseUrl`. Passing the bare origin would send `GET('/accounts')` to
 * `/accounts` and 404, silently missing the proxy entirely.
 */
export const API_BASE_URL = `${apiOrigin}/api`;

export const apiClient = createClient<paths>({
  baseUrl: API_BASE_URL,
  // The origin, not API_BASE_URL: the refresh path this uses already includes
  // `/api`, so handing it the suffixed base would produce `/api/api/auth/refresh`.
  fetch: createAuthenticatedFetch({ baseUrl: apiOrigin }),
});

/**
 * The shape every error response in the contract uses.
 */
type ApiErrorEnvelope = {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
  };
};

/**
 * Converts a failed generated-client call into the app's `ApiError`.
 *
 * `createApiError` takes a `Response` and reads its body; openapi-fetch has
 * already parsed the body by the time a caller sees it, and the response stream
 * cannot be read twice. So this is the deliberate mapping the ticket calls for
 * rather than a reuse: same `ApiError` class, same `details` shape, same
 * `isUnauthorized()` / `isNotFound()` helpers, so nothing downstream changes.
 *
 * The gain over the old path is that `message` now comes from the API's own
 * structured error when it has one — "You already have an account with this
 * name." instead of "Failed to fetch accounts: 409 Conflict". The generic form
 * remains the fallback for responses with no usable envelope.
 */
export function toApiError(
  response: Response,
  resource: string,
  errorBody: unknown
): ApiError {
  const envelope = (errorBody ?? null) as ApiErrorEnvelope | null;
  const apiMessage = envelope?.error?.message;

  const details: ApiErrorDetails = {
    status: response.status,
    statusText: response.statusText,
    url: response.url,
    resource,
    timestamp: new Date().toISOString(),
    errorData: errorBody ?? null,
  };

  const message =
    typeof apiMessage === 'string' && apiMessage !== ''
      ? apiMessage
      : `Failed to fetch ${resource}: ${response.status} ${response.statusText}`;

  return new ApiError(message, details);
}

/**
 * Returns the machine-readable error code from a failed call, when the response
 * carried one.
 *
 * Call sites should branch on this rather than on the human-readable message,
 * which is subject to change and to translation. `DUPLICATE_ACCOUNT_NAME` and
 * `ACCESS_TOKEN_EXPIRED` are the ones the UI currently cares about.
 */
export function apiErrorCode(errorBody: unknown): string | null {
  const code = (errorBody as ApiErrorEnvelope | null)?.error?.code;
  return typeof code === 'string' ? code : null;
}

/**
 * Unwraps a generated-client result, throwing `ApiError` on failure.
 *
 * Every hook does the same three things — check `error`, build an `ApiError`,
 * return `data` — so this keeps that in one place and out of #50's hook
 * migrations.
 */
export function unwrap<T>(
  result: { data?: T; error?: unknown; response: Response },
  resource: string
): T {
  if (result.error !== undefined || !result.response.ok) {
    throw toApiError(result.response, resource, result.error);
  }
  // A 2xx with no body cannot happen on the app resource operations: every one
  // of them returns the `{ data: ... }` envelope.
  return result.data as T;
}
