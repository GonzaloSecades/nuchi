import { describe, expect, it } from 'bun:test';

import {
  createEmailVerifier,
  createLogout,
  createSessionBootstrap,
  type MessageResponse,
  type SessionResponse,
} from '@/lib/auth/session-requests';

const user = { id: 'user-1', email: 'user@example.com', emailVerified: true };

/** A resolver a test can settle by hand, to hold a request genuinely in flight. */
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('createSessionBootstrap', () => {
  it('restores a session from a successful refresh', async () => {
    const bootstrap = createSessionBootstrap({
      refresh: async () => ({
        ok: true,
        body: { accessToken: 'fresh-token', user },
      }),
    });

    expect(await bootstrap()).toEqual({
      status: 'authenticated',
      accessToken: 'fresh-token',
      user,
    });
  });

  /**
   * The Strict Mode case. Setup runs, cleanup runs, setup runs again — all in
   * one commit, before the request can settle. Two real requests would present
   * the same single-use refresh cookie and the second would be rejected,
   * clearing it server-side and signing out a valid session.
   */
  it('issues one request when the mount effect runs twice before it settles', async () => {
    const gate = deferred<SessionResponse>();
    let calls = 0;
    const bootstrap = createSessionBootstrap({
      refresh: () => {
        calls += 1;
        return gate.promise;
      },
    });

    const first = bootstrap(); // effect setup
    const second = bootstrap(); // effect setup again after cleanup

    expect(calls).toBe(1);

    gate.resolve({ ok: true, body: { accessToken: 'shared', user } });

    // The surviving effect must receive the result, not a discarded one.
    expect(await second).toEqual({
      status: 'authenticated',
      accessToken: 'shared',
      user,
    });
    expect(await first).toEqual(await second);
  });

  it('shares one request across many concurrent callers', async () => {
    const gate = deferred<SessionResponse>();
    let calls = 0;
    const bootstrap = createSessionBootstrap({
      refresh: () => {
        calls += 1;
        return gate.promise;
      },
    });

    const all = Promise.all([bootstrap(), bootstrap(), bootstrap()]);
    gate.resolve({ ok: true, body: { accessToken: 't', user } });
    await all;

    expect(calls).toBe(1);
  });

  // The promise must be released once it settles, or a later sign-in after a
  // sign-out would replay the stale signed-out result forever.
  it('bootstraps again after the first attempt settles', async () => {
    let calls = 0;
    const bootstrap = createSessionBootstrap({
      refresh: async () => {
        calls += 1;
        return { ok: true, body: { accessToken: `token-${calls}`, user } };
      },
    });

    await bootstrap();
    const second = await bootstrap();

    expect(calls).toBe(2);
    expect(second).toEqual({
      status: 'authenticated',
      accessToken: 'token-2',
      user,
    });
  });

  it('reports an unauthenticated session for a rejected refresh', async () => {
    const bootstrap = createSessionBootstrap({
      refresh: async () => ({ ok: false, body: null }),
    });

    expect(await bootstrap()).toEqual({ status: 'unauthenticated' });
  });

  const unusableBodies: Array<[string, SessionResponse['body']]> = [
    ['no body', null],
    ['no token', { user }],
    ['an empty token', { accessToken: '', user }],
    ['a non-string token', { accessToken: 42, user }],
    ['no user', { accessToken: 'fresh' }],
  ];

  for (const [label, body] of unusableBodies) {
    it(`reports unauthenticated for a 200 with ${label}`, async () => {
      const bootstrap = createSessionBootstrap({
        refresh: async () => ({ ok: true, body }),
      });

      expect(await bootstrap()).toEqual({ status: 'unauthenticated' });
    });
  }

  /**
   * Without this the async effect rejects, `status` never leaves `loading`,
   * and the app sits on a spinner that nothing will ever clear.
   */
  it('resolves to a usable state when the request throws', async () => {
    const bootstrap = createSessionBootstrap({
      refresh: async () => {
        throw new Error('network down');
      },
    });

    expect(await bootstrap()).toEqual({ status: 'unauthenticated' });
  });

  it('can retry after a network failure', async () => {
    let calls = 0;
    const bootstrap = createSessionBootstrap({
      refresh: async () => {
        calls += 1;
        if (calls === 1) {
          throw new Error('network down');
        }
        return { ok: true, body: { accessToken: 'recovered', user } };
      },
    });

    expect(await bootstrap()).toEqual({ status: 'unauthenticated' });
    expect(await bootstrap()).toEqual({
      status: 'authenticated',
      accessToken: 'recovered',
      user,
    });
  });
});

describe('createEmailVerifier', () => {
  const verified: MessageResponse = { ok: true, message: 'Email verified.' };

  it('verifies a token', async () => {
    const verify = createEmailVerifier({ verify: async () => verified });

    expect(await verify('token-a')).toEqual({
      status: 'verified',
      message: 'Email verified.',
    });
  });

  /**
   * The core of the finding: the token is single-use. A second submission of
   * the same token would be rejected as already consumed and would report
   * failure for a verification that in fact succeeded.
   */
  it('submits a single-use token only once across repeated effects', async () => {
    let calls = 0;
    const verify = createEmailVerifier({
      verify: async () => {
        calls += 1;
        return verified;
      },
    });

    const first = verify('token-a');
    const second = verify('token-a');
    await Promise.all([first, second]);

    // Even after settling — the later Strict Mode setup may arrive then.
    const third = await verify('token-a');

    expect(calls).toBe(1);
    expect(third).toEqual({ status: 'verified', message: 'Email verified.' });
  });

  it('replays the first result rather than resubmitting after it settles', async () => {
    let calls = 0;
    const verify = createEmailVerifier({
      verify: async () => {
        calls += 1;
        return { ok: true, message: `attempt-${calls}` };
      },
    });

    expect((await verify('token-a')) as { message: string }).toMatchObject({
      message: 'attempt-1',
    });
    expect((await verify('token-a')) as { message: string }).toMatchObject({
      message: 'attempt-1',
    });
    expect(calls).toBe(1);
  });

  /**
   * Next keeps client state when only the search parameter changes, so a
   * boolean "already submitted" flag would strand the second link forever.
   */
  it('processes a different token without a reload', async () => {
    const seen: string[] = [];
    const verify = createEmailVerifier({
      verify: async (token) => {
        seen.push(token);
        return { ok: true, message: `verified ${token}` };
      },
    });

    await verify('token-a');
    const second = await verify('token-b');

    expect(seen).toEqual(['token-a', 'token-b']);
    expect(second).toEqual({
      status: 'verified',
      message: 'verified token-b',
    });
  });

  it('reports a rejected token as failed with the API error', async () => {
    const apiError = new Error('INVALID_TOKEN');
    const verify = createEmailVerifier({
      verify: async () => ({ ok: false, message: null, error: apiError }),
    });

    expect(await verify('bad')).toEqual({ status: 'failed', error: apiError });
  });

  it('reports a thrown request as failed', async () => {
    const boom = new Error('network down');
    const verify = createEmailVerifier({
      verify: async () => {
        throw boom;
      },
    });

    expect(await verify('token-a')).toEqual({ status: 'failed', error: boom });
  });

  it('falls back to a message when the API omits one', async () => {
    const verify = createEmailVerifier({
      verify: async () => ({ ok: true, message: null }),
    });

    expect(await verify('token-a')).toEqual({
      status: 'verified',
      message: 'Email verified.',
    });
  });
});

describe('createLogout', () => {
  it('reports a confirmed server logout', async () => {
    const logout = createLogout({ logout: async () => ({ ok: true }) });

    expect(await logout()).toEqual({ serverConfirmed: true });
  });

  /**
   * `apiClient.POST` resolves for a non-2xx, so an unchecked call would treat a
   * 500 as success while the refresh cookie stayed valid — "signed out" that a
   * reload undoes.
   */
  it('does not treat a non-2xx response as a server logout', async () => {
    const logout = createLogout({ logout: async () => ({ ok: false }) });

    expect(await logout()).toEqual({ serverConfirmed: false });
  });

  it('does not throw when the request fails outright', async () => {
    const logout = createLogout({
      logout: async () => {
        throw new Error('network down');
      },
    });

    expect(await logout()).toEqual({ serverConfirmed: false });
  });
});
