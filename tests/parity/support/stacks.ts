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

/**
 * The **origin** the Go API is served from — scheme and host only, no path.
 *
 * The `/api` prefix is added by this module (`${GO_API_URL}/api${path}`), so a
 * value that already includes it produces `/api/api/...`.
 */
export const GO_API_URL = process.env.GO_API_URL ?? 'http://localhost:8080';

/**
 * Whether the host timezone can observe a naive-timestamp divergence at all.
 *
 * The two stacks differ by labelling the same bytes UTC or host-local, so at
 * UTC there is nothing to see: both produce the identical string and a
 * comparison would be trivially equal. Tests that depend on the difference skip
 * rather than pass, because a passing assertion that compared nothing is
 * exactly the false green this harness exists to avoid.
 */
export const HOST_UTC_OFFSET_MINUTES = new Date().getTimezoneOffset();
export const CAN_OBSERVE_TIMEZONE_DIVERGENCE = HOST_UTC_OFFSET_MINUTES !== 0;

export type Prerequisites = { ok: boolean; reason: string };

/**
 * Checks that the harness can actually reach both stacks.
 *
 * Postgres is *connected to*, not merely configured: a wrong URL or a stopped
 * server would otherwise surface much later as a confusing failure inside
 * `connectAdmin`, which is the opposite of the "skip rather than look like
 * evidence" rule this harness is built on.
 */
export async function parityPrerequisites(): Promise<Prerequisites> {
  if (!ADMIN_DATABASE_URL) {
    return {
      ok: false,
      reason:
        'ADMIN_DATABASE_URL is unset. See tests/parity/README.md for the values.',
    };
  }

  let probe: Client | undefined;
  try {
    probe = new Client({ connectionString: ADMIN_DATABASE_URL });
    await probe.connect();
    await probe.query('SELECT 1');
  } catch (error) {
    return {
      ok: false,
      reason: `Postgres is not reachable at ADMIN_DATABASE_URL: ${describe(error)}`,
    };
  } finally {
    await probe?.end().catch(() => {});
  }

  try {
    // Any HTTP answer proves the API is up; 401 is the expected one here, since
    // this deliberately sends no refresh cookie.
    await fetch(`${GO_API_URL}/api/auth/refresh`, { method: 'POST' });
  } catch (error) {
    return {
      ok: false,
      reason: `No Go API reachable at ${GO_API_URL} (${describe(error)}). Start it first — see tests/parity/README.md.`,
    };
  }

  return { ok: true, reason: '' };
}

function describe(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Opens an admin connection to the shared database. */
export async function connectAdmin(): Promise<Client> {
  const client = new Client({ connectionString: ADMIN_DATABASE_URL });
  await client.connect();
  return client;
}

export type GoResult = { status: number; body: unknown };

/**
 * Calls the Go API as the given session.
 *
 * Headers go through `Headers` rather than object spread: `HeadersInit` may be
 * a `Headers` instance or an array of pairs, and spreading either silently
 * drops every entry. Content-Type is only defaulted, not forced, so a caller
 * can send something other than JSON.
 */
export async function goRequest(
  token: string,
  path: string,
  init: RequestInit = {}
): Promise<GoResult> {
  const headers = new Headers(init.headers);
  headers.set('Authorization', `Bearer ${token}`);
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${GO_API_URL}/api${path}`, {
    ...init,
    headers,
  });
  const body = await response.json().catch(() => null);
  return { status: response.status, body };
}

/**
 * Asserts a Go response succeeded, reporting the status and body when it did
 * not.
 *
 * Without this a setup failure surfaces indirectly — "no row found", or
 * `Cannot read properties of undefined` — and the real cause (a schema
 * mismatch, an auth problem) has to be reconstructed. A differential test that
 * fails should say which stack refused and why.
 */
export function expectOk(result: GoResult, what: string): unknown {
  if (result.status < 200 || result.status >= 300) {
    throw new Error(
      `parity: ${what} failed with ${result.status}: ${JSON.stringify(result.body)}`
    );
  }
  return result.body;
}

/** Reads the `data` envelope from a successful app resource response. */
export function expectData<T>(result: GoResult, what: string): T {
  const body = expectOk(result, what) as { data?: T } | null;
  if (body === null || body.data === undefined) {
    throw new Error(
      `parity: ${what} returned no data envelope: ${JSON.stringify(result.body)}`
    );
  }
  return body.data;
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
    throw new Error(
      `parity: register failed with ${registered.status}: ${await registered.text()}`
    );
  }

  const verified = await admin.query(
    'UPDATE users SET email_verified_at = now() WHERE email = $1',
    [email]
  );
  // Asserted rather than assumed: if the row is not there, the harness is
  // talking to a different database than the API is, and "login failed" three
  // lines down would send the reader looking in entirely the wrong place.
  if (verified.rowCount !== 1) {
    throw new Error(
      `parity: expected to verify exactly one user, updated ${verified.rowCount}. Is ADMIN_DATABASE_URL the same database the Go API uses?`
    );
  }

  const loggedIn = await fetch(`${GO_API_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!loggedIn.ok) {
    throw new Error(
      `parity: login failed with ${loggedIn.status}: ${await loggedIn.text()}`
    );
  }

  // Auth responses are un-enveloped; only app resource operations use `{ data }`.
  const session = (await loggedIn.json()) as {
    accessToken: string;
    user: { id: string };
  };

  return { token: session.accessToken, userId: session.user.id, email };
}

/** Removes everything a run created, so repeat runs stay comparable. */
export async function cleanupUser(
  admin: Client,
  userId: string
): Promise<void> {
  // accounts and categories cascade to transactions through their own FKs.
  await admin.query('DELETE FROM users WHERE id = $1', [userId]);
}
