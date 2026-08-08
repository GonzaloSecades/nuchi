import { beforeEach, describe, expect, it } from 'bun:test';

import { createAuthenticatedFetch } from '@/lib/api/authenticated-fetch';

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

const expired = () =>
  jsonResponse(401, {
    error: { code: 'ACCESS_TOKEN_EXPIRED', message: 'Access token expired.' },
  });

const unauthorized = () =>
  jsonResponse(401, {
    error: { code: 'UNAUTHORIZED', message: 'Authentication required.' },
  });

/** Records every call so tests can assert what was sent, and in what order. */
function recordingFetch(responses: Array<() => Response>) {
  const calls: Array<{ url: string; init?: RequestInit }> = [];
  let index = 0;

  const fetchImpl = (async (input: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(input), init });
    const next = responses[Math.min(index, responses.length - 1)];
    index += 1;
    return next();
  }) as unknown as typeof globalThis.fetch;

  return { fetchImpl, calls, callCount: () => index };
}

describe('createAuthenticatedFetch', () => {
  let token: string | null;

  beforeEach(() => {
    token = 'initial-token';
  });

  const store = () => ({
    getToken: () => token,
    setToken: (next: string | null) => {
      token = next;
    },
    clearToken: () => {
      token = null;
    },
  });

  it('attaches the access token as a Bearer header', async () => {
    const { fetchImpl, calls } = recordingFetch([() => jsonResponse(200, {})]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    await authed('/api/accounts');

    const headers = new Headers(calls[0].init?.headers);
    expect(headers.get('Authorization')).toBe('Bearer initial-token');
  });

  it('sends no Authorization header when there is no token', async () => {
    token = null;
    const { fetchImpl, calls } = recordingFetch([() => jsonResponse(200, {})]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    await authed('/api/accounts');

    const headers = new Headers(calls[0].init?.headers);
    expect(headers.get('Authorization')).toBeNull();
  });

  // The refresh token is an httpOnly cookie, so it only travels when the
  // request opts into sending credentials.
  it('always includes credentials so the refresh cookie travels', async () => {
    const { fetchImpl, calls } = recordingFetch([() => jsonResponse(200, {})]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    await authed('/api/accounts');

    expect(calls[0].init?.credentials).toBe('include');
  });

  it('passes a successful response straight through', async () => {
    const { fetchImpl, callCount } = recordingFetch([
      () => jsonResponse(200, { data: [] }),
    ]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/accounts');

    expect(response.status).toBe(200);
    expect(callCount()).toBe(1);
  });

  it('refreshes and retries once when the token has expired', async () => {
    const { fetchImpl, calls } = recordingFetch([
      expired,
      () => jsonResponse(200, { data: { accessToken: 'fresh-token' } }),
      () => jsonResponse(200, { data: [] }),
    ]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/accounts');

    expect(response.status).toBe(200);
    expect(calls.map((c) => c.url)).toEqual([
      '/api/accounts',
      '/api/auth/refresh',
      '/api/accounts',
    ]);
    // The retry carries the new token, not the stale one.
    expect(new Headers(calls[2].init?.headers).get('Authorization')).toBe(
      'Bearer fresh-token'
    );
    expect(token).toBe('fresh-token');
  });

  // Any 401 that is not the documented expiry carve-out is a credential
  // problem; rotating would not fix it and retrying only doubles the failures.
  it('does not refresh on a non-expiry 401', async () => {
    const { fetchImpl, callCount } = recordingFetch([unauthorized]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/accounts');

    expect(response.status).toBe(401);
    expect(callCount()).toBe(1);
    expect(token).toBe('initial-token');
  });

  it('does not refresh on a 401 with no parseable body', async () => {
    const { fetchImpl, callCount } = recordingFetch([
      () => new Response('not json', { status: 401 }),
    ]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    expect((await authed('/api/accounts')).status).toBe(401);
    expect(callCount()).toBe(1);
  });

  it('clears the token and gives up when the refresh itself fails', async () => {
    const { fetchImpl, calls } = recordingFetch([
      expired,
      () => jsonResponse(401, { error: { code: 'INVALID_REFRESH_TOKEN' } }),
    ]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/accounts');

    expect(response.status).toBe(401);
    expect(calls).toHaveLength(2);
    expect(token).toBeNull();
  });

  it('clears the token when the refresh returns no usable token', async () => {
    const { fetchImpl } = recordingFetch([
      expired,
      () => jsonResponse(200, { data: {} }),
    ]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    await authed('/api/accounts');

    expect(token).toBeNull();
  });

  // The important one. A dashboard fires several queries at once, so an expired
  // token yields several simultaneous 401s. Refreshing per request would rotate
  // the refresh token once and then present the consumed one, logging the user
  // out mid-session.
  it('shares a single refresh across concurrent requests', async () => {
    let refreshCalls = 0;
    const fetchImpl = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith('/api/auth/refresh')) {
        refreshCalls += 1;
        // Resolve on a later tick so the other callers are genuinely in flight.
        await new Promise((resolve) => setTimeout(resolve, 5));
        return jsonResponse(200, { data: { accessToken: 'fresh-token' } });
      }
      return token === 'fresh-token' ? jsonResponse(200, { data: [] }) : expired();
    }) as unknown as typeof globalThis.fetch;

    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const responses = await Promise.all([
      authed('/api/accounts'),
      authed('/api/categories'),
      authed('/api/transactions'),
      authed('/api/summary'),
    ]);

    expect(refreshCalls).toBe(1);
    for (const response of responses) {
      expect(response.status).toBe(200);
    }
  });

  // The shared promise must be released once it settles. If it were cached
  // forever, the second expiry would reuse the first (already resolved) refresh
  // and retry with a token that has since expired again — a session that dies
  // after exactly one renewal.
  it('allows a later refresh after an earlier one finished', async () => {
    let refreshCalls = 0;
    let resourceAttempts = 0;

    const fetchImpl = (async (input: RequestInfo | URL) => {
      if (String(input).endsWith('/api/auth/refresh')) {
        refreshCalls += 1;
        return jsonResponse(200, {
          data: { accessToken: `token-${refreshCalls}` },
        });
      }
      resourceAttempts += 1;
      // Attempts 1 and 3 are the first try of each authed() call and expire;
      // attempts 2 and 4 are the retries and succeed.
      return resourceAttempts % 2 === 1 ? expired() : jsonResponse(200, {});
    }) as unknown as typeof globalThis.fetch;

    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    expect((await authed('/api/accounts')).status).toBe(200);
    expect((await authed('/api/accounts')).status).toBe(200);

    expect(refreshCalls).toBe(2);
    expect(token).toBe('token-2');
  });

  // Refreshing the refresh call would recurse until the stack gave out.
  it('never tries to refresh the refresh endpoint itself', async () => {
    const { fetchImpl, callCount } = recordingFetch([expired]);
    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/auth/refresh');

    expect(response.status).toBe(401);
    expect(callCount()).toBe(1);
  });

  it('prefixes the refresh call with the configured base URL', async () => {
    const { fetchImpl, calls } = recordingFetch([
      expired,
      () => jsonResponse(200, { data: { accessToken: 'fresh' } }),
      () => jsonResponse(200, {}),
    ]);
    const authed = createAuthenticatedFetch({
      fetch: fetchImpl,
      baseUrl: 'http://localhost:3000',
      ...store(),
    });

    await authed('/api/accounts');

    expect(calls[1].url).toBe('http://localhost:3000/api/auth/refresh');
  });
});

describe('createAuthenticatedFetch resilience', () => {
  let token: string | null = 'initial-token';
  const store = () => ({
    getToken: () => token,
    setToken: (next: string | null) => {
      token = next;
    },
    clearToken: () => {
      token = null;
    },
  });

  // A dropped connection during refresh must surface as the original 401, not
  // as a thrown fetch error. Otherwise a transient network blip turns into a
  // crash in whichever query happened to trigger the renewal, which looks
  // nothing like the 401 the same situation produces when online.
  it('returns the original 401 when the refresh request throws', async () => {
    token = 'initial-token';
    let calls = 0;
    const fetchImpl = (async (input: RequestInfo | URL) => {
      calls += 1;
      if (String(input).endsWith('/api/auth/refresh')) {
        throw new TypeError('network error');
      }
      return jsonResponse(401, {
        error: { code: 'ACCESS_TOKEN_EXPIRED', message: 'Access token expired.' },
      });
    }) as unknown as typeof globalThis.fetch;

    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    const response = await authed('/api/accounts');

    expect(response.status).toBe(401);
    expect(calls).toBe(2);
  });

  // The shared promise must also be released after a throwing refresh, or the
  // session could never recover once the network came back.
  it('can refresh again after a failed refresh', async () => {
    token = 'initial-token';
    let refreshAttempts = 0;
    let resourceAttempts = 0;

    const fetchImpl = (async (input: RequestInfo | URL) => {
      if (String(input).endsWith('/api/auth/refresh')) {
        refreshAttempts += 1;
        if (refreshAttempts === 1) throw new TypeError('network error');
        return jsonResponse(200, { data: { accessToken: 'recovered' } });
      }
      resourceAttempts += 1;
      return resourceAttempts >= 3
        ? jsonResponse(200, {})
        : jsonResponse(401, { error: { code: 'ACCESS_TOKEN_EXPIRED' } });
    }) as unknown as typeof globalThis.fetch;

    const authed = createAuthenticatedFetch({ fetch: fetchImpl, ...store() });

    expect((await authed('/api/accounts')).status).toBe(401);
    expect((await authed('/api/accounts')).status).toBe(200);
    expect(refreshAttempts).toBe(2);
  });
});
