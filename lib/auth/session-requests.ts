import { apiClient, toApiError } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';

export type AuthUser = components['schemas']['AuthUser'];

/**
 * Request coordination for the auth flows.
 *
 * This lives outside the React components on purpose. Every hazard here comes
 * from an effect running more than once — React Strict Mode runs each mount
 * effect setup → cleanup → setup in development — and each of these flows
 * spends a **single-use** credential. A second request is not a wasted
 * round-trip; it consumes or rotates a token the first request already claimed.
 *
 * Keeping the coordination in plain functions also makes it testable. The repo
 * has no DOM or React test setup, so logic left inside a component effect can
 * only be verified by hand.
 */

export type BootstrapResult =
  | { status: 'authenticated'; accessToken: string; user: AuthUser }
  | { status: 'unauthenticated' };

/** What the coordinators need from a session request, so tests need no HTTP. */
export type SessionResponse = {
  ok: boolean;
  body: { accessToken?: unknown; user?: unknown } | null;
};

/** What the coordinators need from a message-shaped request. */
export type MessageResponse = {
  ok: boolean;
  message: string | null;
  /** Populated only when `ok` is false. */
  error?: unknown;
};

/**
 * Builds the session bootstrap, deduplicated across concurrent callers.
 *
 * The refresh token is single-use and rotating. Under Strict Mode the mount
 * effect runs twice, and both runs happen in the same commit — before the first
 * request can settle. Two independent calls would therefore present the *same*
 * cookie: the first rotates it, the second arrives with a consumed token, gets
 * a 401, and the server clears the cookie. The user reloads a valid session and
 * lands signed out.
 *
 * One shared in-flight promise removes that: the second caller awaits the first
 * result instead of racing it. The promise is released once it settles, so a
 * genuine later remount (sign out, sign back in) bootstraps again. This mirrors
 * `refreshOnce` in `lib/api/authenticated-fetch.ts`, which solves the same
 * problem for concurrent 401s.
 */
export function createSessionBootstrap(deps: {
  refresh: () => Promise<SessionResponse>;
}) {
  let inFlight: Promise<BootstrapResult> | null = null;

  async function run(): Promise<BootstrapResult> {
    let response: SessionResponse;
    try {
      response = await deps.refresh();
    } catch {
      // A network failure is not a signed-in session, and it must not leave the
      // caller waiting forever. Reporting it as signed out lets the UI render
      // sign-in rather than a permanent spinner.
      return { status: 'unauthenticated' };
    }

    const accessToken = response.body?.accessToken;
    const user = response.body?.user;

    if (
      !response.ok ||
      typeof accessToken !== 'string' ||
      accessToken === '' ||
      typeof user !== 'object' ||
      user === null
    ) {
      return { status: 'unauthenticated' };
    }

    return { status: 'authenticated', accessToken, user: user as AuthUser };
  }

  return function bootstrap(): Promise<BootstrapResult> {
    inFlight ??= run().finally(() => {
      inFlight = null;
    });
    return inFlight;
  };
}

export type VerifyResult =
  | { status: 'verified'; message: string }
  | { status: 'failed'; error: unknown };

/**
 * Builds the email verifier, memoized per token.
 *
 * Deliberately different from the bootstrap above: the settled promise is
 * **kept**, not released. An email verification token is single-use, so if the
 * second Strict Mode effect arrives after the first request already succeeded,
 * resubmitting would spend a token that is now consumed and turn a successful
 * verification into `INVALID_TOKEN` on screen. Replaying the first result is
 * the only correct answer.
 *
 * The cache holds one entry, keyed by token, rather than a growing map. That is
 * what lets `?token=A` → `?token=B` issue a real second request: Next keeps
 * client state when only the search parameter changes, so a plain boolean
 * "already submitted" flag would strand the second link on "Verifying" forever.
 */
export function createEmailVerifier(deps: {
  verify: (token: string) => Promise<MessageResponse>;
}) {
  let cache: { token: string; promise: Promise<VerifyResult> } | null = null;

  async function run(token: string): Promise<VerifyResult> {
    let response: MessageResponse;
    try {
      response = await deps.verify(token);
    } catch (error) {
      return { status: 'failed', error };
    }

    if (!response.ok) {
      return { status: 'failed', error: response.error };
    }

    return {
      status: 'verified',
      message: response.message ?? 'Email verified.',
    };
  }

  return function verifyOnce(token: string): Promise<VerifyResult> {
    if (cache !== null && cache.token === token) {
      return cache.promise;
    }
    const promise = run(token);
    cache = { token, promise };
    return promise;
  };
}

export type LogoutOutcome = {
  /** True only when the server confirmed it revoked the session. */
  serverConfirmed: boolean;
};

/**
 * Builds the logout call.
 *
 * `apiClient.POST` resolves normally for a non-2xx response, so an unchecked
 * call treats a 500 as success. That matters more here than elsewhere: the
 * refresh cookie would still be valid, so "signed out" would be a claim a page
 * reload immediately disproves.
 *
 * The local session is cleared either way — someone who asked to leave should
 * not stay signed in on this device because the server had a bad minute — but
 * the caller is told which of the two happened so the UI can say so.
 */
export function createLogout(deps: { logout: () => Promise<{ ok: boolean }> }) {
  return async function logout(): Promise<LogoutOutcome> {
    try {
      const { ok } = await deps.logout();
      return { serverConfirmed: ok };
    } catch {
      return { serverConfirmed: false };
    }
  };
}

/* -------------------------------------------------------------------------
 * Default instances, wired to the real client.
 * ---------------------------------------------------------------------- */

export const bootstrapSession = createSessionBootstrap({
  refresh: async () => {
    const { data, error, response } = await apiClient.POST('/auth/refresh');
    return {
      ok: response.ok && error === undefined,
      body: (data ?? null) as SessionResponse['body'],
    };
  },
});

export const verifyEmailToken = createEmailVerifier({
  verify: async (token) => {
    const { data, error, response } = await apiClient.POST(
      '/auth/verify-email',
      { body: { token } }
    );
    const ok = response.ok && error === undefined;
    return {
      ok,
      message: data?.message ?? null,
      error: ok ? undefined : toApiError(response, 'email verification', error),
    };
  },
});

export const logoutSession = createLogout({
  logout: async () => {
    const { response } = await apiClient.POST('/auth/logout');
    return { ok: response.ok };
  },
});
