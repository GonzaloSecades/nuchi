'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

import { apiClient, unwrap } from '@/lib/api/client';
import {
  setAccessToken,
  clearAccessToken,
  subscribeToAccessToken,
} from '@/lib/api/token-store';
import {
  bootstrapSession,
  logoutSession,
  type AuthUser,
  type LogoutOutcome,
} from '@/lib/auth/session-requests';

export type { AuthUser };

/**
 * Where the session is in its lifecycle.
 *
 * `loading` is the state every page load starts in, and it is not cosmetic:
 * the access token lives in memory only, so immediately after a reload the app
 * cannot yet tell a signed-in user from a signed-out one. It has to ask the
 * server first. Rendering a guard's redirect during that window would bounce
 * signed-in users to sign-in on every refresh.
 */
export type SessionStatus = 'loading' | 'authenticated' | 'unauthenticated';

type SessionContextValue = {
  user: AuthUser | null;
  status: SessionStatus;
  login: (email: string, password: string) => Promise<AuthUser>;
  register: (email: string, password: string) => Promise<string>;
  /** Resolves with whether the server confirmed the revocation. Never rejects. */
  logout: () => Promise<LogoutOutcome>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

/**
 * Owns the session: bootstraps it from the refresh cookie on load, exposes the
 * auth commands, and tears it down when the token goes away.
 *
 * Replaces `ClerkProvider`. The state it holds is deliberately small — the
 * access token itself stays in `lib/api/token-store`, which is what
 * `apiClient` reads, so there is exactly one copy of the credential and this
 * provider never becomes a second place it can go stale.
 */
export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [status, setStatus] = useState<SessionStatus>('loading');

  const applySession = useCallback(
    (session: { accessToken: string; user: AuthUser }) => {
      setAccessToken(session.accessToken);
      setUser(session.user);
      setStatus('authenticated');
    },
    []
  );

  /**
   * Restores a session from the httpOnly refresh cookie on first paint.
   *
   * A page load has no access token, only that cookie, so this is the one
   * request that decides whether the user is still signed in. A 401 here is
   * the ordinary signed-out case, not an error worth surfacing.
   *
   * `bootstrapSession` deduplicates: Strict Mode runs this setup → cleanup →
   * setup in one commit, and two real requests would present the same
   * single-use refresh cookie — the second would be rejected and the server
   * would clear it, signing out a valid session on reload. It also never
   * rejects, so `status` cannot be stranded on `loading` by a network failure.
   */
  useEffect(() => {
    let cancelled = false;

    bootstrapSession().then((result) => {
      if (cancelled) {
        return;
      }
      if (result.status === 'unauthenticated') {
        setStatus('unauthenticated');
        return;
      }
      applySession(result);
    });

    return () => {
      cancelled = true;
    };
  }, [applySession]);

  /**
   * Ends the session when the token store is cleared from anywhere else.
   *
   * `authenticated-fetch` clears the token when a mid-session refresh fails,
   * and that happens deep inside a query with no way to reach this state. Only
   * a clear matters here: a `set` is a login or a successful renewal, both of
   * which are already reflected above.
   *
   * The updates are functional so the listener needs no view of the current
   * user. That keeps the subscription mounted once for the provider's lifetime
   * — the alternative, mirroring `user` into a ref, meant writing that ref
   * during render, which React forbids — and makes a clear while already
   * signed out a genuine no-op rather than a redundant re-render.
   */
  useEffect(() => {
    return subscribeToAccessToken((token) => {
      if (token !== null) {
        return;
      }
      setUser((current) => (current === null ? current : null));
      setStatus((current) =>
        current === 'authenticated' ? 'unauthenticated' : current
      );
    });
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const result = await apiClient.POST('/auth/login', {
        body: { email, password },
      });
      const session = unwrap(result, 'session');
      applySession(session);
      return session.user;
    },
    [applySession]
  );

  /**
   * Creates the account. Deliberately does not sign the user in: registration
   * returns no session, and login is refused with `EMAIL_NOT_VERIFIED` until
   * the emailed link is followed. Returns the API's own message so the page
   * can show what the server actually said.
   */
  const register = useCallback(async (email: string, password: string) => {
    const result = await apiClient.POST('/auth/register', {
      body: { email, password },
    });
    return unwrap(result, 'registration').message;
  }, []);

  /**
   * Ends the session, reporting whether the server actually revoked it.
   *
   * The local state is cleared either way: someone who asked to leave should
   * not stay signed in on this device because the server had a bad minute. But
   * the two cases are not the same — an unconfirmed logout leaves a valid
   * refresh cookie, so a reload would sign the user straight back in, and the
   * UI needs to be able to say so rather than claim a clean sign-out.
   */
  const logout = useCallback(async () => {
    const outcome = await logoutSession();
    clearAccessToken();
    setUser(null);
    setStatus('unauthenticated');
    return outcome;
  }, []);

  const value = useMemo(
    () => ({ user, status, login, register, logout }),
    [user, status, login, register, logout]
  );

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  );
}

/** Reads the session. Throws outside `SessionProvider` rather than returning a
 * silently signed-out value, which would render a guard as an infinite redirect. */
export function useSession(): SessionContextValue {
  const context = useContext(SessionContext);
  if (context === null) {
    throw new Error('useSession must be used within a SessionProvider');
  }
  return context;
}
