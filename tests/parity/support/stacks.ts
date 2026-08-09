import { Client } from 'pg';

/**
 * Connection and session plumbing shared by the parity tests.
 *
 * Everything here is gated on environment variables and reports cleanly when
 * they are absent, matching how the Go live tests gate on TEST_DATABASE_URL. A
 * parity run that cannot reach both stacks must say so rather than pass on an
 * empty comparison — a silently skipped differential test is worse than none,
 * because it looks like evidence.
 */

/**
 * Admin connection string. Admin rather than the app role on purpose: the
 * harness seeds and reads rows across users, and the app role is subject to
 * RLS, which would return zero rows and make every comparison trivially equal.
 */
export const ADMIN_DATABASE_URL = process.env.ADMIN_DATABASE_URL ?? '';

/** Base URL of a running Go API, including the `/api` prefix's origin. */
export const GO_API_URL = process.env.GO_API_URL ?? 'http://localhost:8080';

/** Whether the harness has everything it needs to run against both stacks. */
export async function parityPrerequisites(): Promise<{
  ok: boolean;
  reason: string;
}> {
  if (!ADMIN_DATABASE_URL) {
    return {
      ok: false,
      reason:
        'ADMIN_DATABASE_URL is unset. See tests/parity/README.md for the values.',
    };
  }

  try {
    const response = await fetch(`${GO_API_URL}/api/auth/refresh`, {
      method: 'POST',
    });
    // Any HTTP answer proves the API is up; 401 is the expected one here,
    // since this deliberately sends no refresh cookie.
    if (!response) {
      return { ok: false, reason: `No response from ${GO_API_URL}.` };
    }
  } catch {
    return {
      ok: false,
      reason: `No Go API reachable at ${GO_API_URL}. Start it first — see tests/parity/README.md.`,
    };
  }

  return { ok: true, reason: '' };
}

/** Opens an admin connection to the shared database. */
export async function connectAdmin(): Promise<Client> {
  const client = new Client({ connectionString: ADMIN_DATABASE_URL });
  await client.connect();
  return client;
}

/**
 * Registers, verifies and logs in a throwaway user, returning its access token
 * and id.
 *
 * Verification is done with SQL rather than by following the emailed link: the
 * token is stored hashed, so the raw value only exists in an email, and making
 * the harness depend on a mail catcher would tie a data-layer comparison to
 * SMTP being up.
 */
export async function provisionGoSession(
  admin: Client
): Promise<{ token: string; userId: string; email: string }> {
  const email = `parity-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`;
  const password = 'parity-harness-password';

  const registered = await fetch(`${GO_API_URL}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!registered.ok) {
    throw new Error(`parity: register failed with ${registered.status}`);
  }

  await admin.query(
    'UPDATE users SET email_verified_at = now() WHERE email = $1',
    [email]
  );

  const loggedIn = await fetch(`${GO_API_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!loggedIn.ok) {
    throw new Error(`parity: login failed with ${loggedIn.status}`);
  }

  // Auth responses are un-enveloped; only app resource operations use `{ data }`.
  const session = (await loggedIn.json()) as {
    accessToken: string;
    user: { id: string };
  };

  return { token: session.accessToken, userId: session.user.id, email };
}

/** Calls the Go API as the given session. */
export async function goRequest(
  token: string,
  path: string,
  init: RequestInit = {}
): Promise<{ status: number; body: unknown }> {
  const response = await fetch(`${GO_API_URL}/api${path}`, {
    ...init,
    headers: {
      ...init.headers,
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });

  const body = await response.json().catch(() => null);
  return { status: response.status, body };
}

/** Removes everything a run created, so repeat runs stay comparable. */
export async function cleanupUser(
  admin: Client,
  userId: string
): Promise<void> {
  // accounts and categories cascade to transactions through their own FKs.
  await admin.query('DELETE FROM users WHERE id = $1', [userId]);
}
