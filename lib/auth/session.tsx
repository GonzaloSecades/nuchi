'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';
import {
  setAccessToken,
  clearAccessToken,
  subscribeToAccessToken,
} from '@/lib/api/token-store';

export type AuthUser = components['schemas']['AuthUser'];

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
  logout: () => Promise<void>;
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

  /**
   * Tracks the user without re-subscribing the token listener on every change.
   *
   * The listener below only needs to know whether a session was live at the
   * moment the token was cleared; reading that from a ref keeps the
   * subscription itself mounted once for the provider's lifetime instead of
   * being torn down and rebuilt each time the user changes.
   */
  const userRef = useRef<AuthUser | null>(null);
  userRef.current = user;

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
   */
  useEffect(() => {
    let cancelled = false;

    (async () => {
      const { data, error, response } = await apiClient.POST('/auth/refresh');

      if (cancelled) {
        return;
      }
      if (error !== undefined || !response.ok || !data) {
        setStatus('unauthenticated');
        return;
      }
      applySession(data);
    })();

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
   */
  useEffect(() => {
    return subscribeToAccessToken((token) => {
      if (token === null && userRef.current !== null) {
        setUser(null);
        setStatus('unauthenticated');
      }
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
   * Ends the session. The local state is cleared whatever the server says: a
   * logout that 401s means the refresh token was already invalid, so the
   * session is over either way and keeping the user "signed in" against a dead
   * token would be worse than the alternative.
   */
  const logout = useCallback(async () => {
    try {
      await apiClient.POST('/auth/logout');
    } finally {
      clearAccessToken();
      setUser(null);
      setStatus('unauthenticated');
    }
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
