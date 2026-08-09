# API Client Layer

The app-owned wrapper around the generated OpenAPI client. Added in
[#49](https://github.com/GonzaloSecades/nuchi/issues/49); the hooks move onto it
in [#50](https://github.com/GonzaloSecades/nuchi/issues/50).

## Files

| File | Role |
| --- | --- |
| `generated/schema.d.ts` | Generated from the contract. **Never edited by hand.** |
| `client.ts` | The configured client, error mapping, `unwrap` |
| `authenticated-fetch.ts` | Bearer attachment, refresh-and-retry |
| `token-store.ts` | In-memory access token |

## Using it

```ts
import { apiClient, unwrap } from '@/lib/api/client';

const { data, error, response } = await apiClient.GET('/accounts');
const accounts = unwrap({ data, error, response }, 'accounts');
```

Paths and shapes are checked against the contract at compile time, so an
operation that changes shape breaks the build rather than failing at runtime.

**Do not import `openapi-fetch` directly.** The auth and refresh behavior lives
in `apiClient`; a client constructed elsewhere would silently skip it and send
unauthenticated requests.

## Same-origin, always

The client's base URL is `` `${resolveApiBaseUrl()}/api` `` — exported as
`API_BASE_URL`. `resolveApiBaseUrl()` returns `''` in the browser, so that
resolves to a relative `/api`, calls go to the Next origin, and the rewrite from
#30 proxies them to Go. Cookies stay first-party, no CORS configuration is
involved, and the browser never learns the backend's address.

**The `/api` suffix is required, not decorative.** Contract paths are rooted at
`/` (`/accounts`, `/summary`, …) and the `/api` prefix lives in the contract's
`servers[].url` — which openapi-fetch does not read. It only prepends `baseUrl`.
A bare origin would send `GET('/accounts')` to `/accounts`, which 404s and never
reaches the proxy at all.

Note `createAuthenticatedFetch` is given the **origin**, not `API_BASE_URL`: the
refresh path it uses already includes `/api`, so the suffixed base would produce
`/api/api/auth/refresh`.

## Tokens

The access token is held **in memory only**. The refresh token is an httpOnly
cookie the browser cannot read.

That combination is deliberate: `localStorage` or a readable cookie would hand
any XSS a credential it could exfiltrate and reuse elsewhere. In memory, a
successful XSS still has to act inside the page.

The cost is that a fresh page load starts with no access token. When the first
protected request receives `UNAUTHORIZED` without having sent an Authorization
header, the client exchanges the refresh cookie and retries without the user
noticing. Auth endpoints never enter this bootstrap path.

## Refresh behavior

On `401` with code `ACCESS_TOKEN_EXPIRED`, the client refreshes once and retries
the original request. It also does this for the missing-token `UNAUTHORIZED`
case described above so a full page reload can resume an existing session.

Three rules worth knowing before changing this:

- **Only expiry or missing-token bootstrap triggers a refresh.** Any other 401
  is a credential problem that rotating will not fix; retrying would just
  double the failed requests.
- **One refresh at a time.** A dashboard fires several queries at once, so an
  expired token produces several simultaneous 401s. Without a shared in-flight
  promise, the first refresh rotates the token and the rest present the consumed
  one and fail — logging the user out mid-session. That promise is released once
  it settles, so a later expiry can refresh again.
- **The retry is not a loop.** If the retried request still 401s, that response
  is returned. A genuinely rejected session surfaces instead of spinning.

When a refresh fails, the token is cleared. `subscribeToAccessToken` exists so
the UI can react to that and send the user to sign-in; the auth pages that
consume it arrive in #51.

## Errors

`toApiError` produces the same `ApiError` class the app already uses, with the
same `details` shape and the same `isUnauthorized()` / `isNotFound()` helpers,
so nothing downstream changes.

It is a **mapping rather than a reuse** of `createApiError`: that function reads
the body off a `Response`, and openapi-fetch has already parsed it by the time a
caller sees the result — a response stream cannot be read twice.

The gain is that `message` now comes from the API's own structured error when
there is one: *"You already have an account with this name."* instead of
*"Failed to fetch accounts: 409 Conflict"*. The generic form remains the
fallback.

Branch on `apiErrorCode(error)` rather than on the message — codes are stable,
messages are not and will eventually be translated.

## Not in this layer

Nothing here knows about TanStack Query, hooks, or components. `#50` builds that
on top. Keeping the boundary means the client can be tested without React and
the hooks can be migrated one at a time.
