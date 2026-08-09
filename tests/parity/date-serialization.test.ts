import { afterAll, beforeAll, describe, expect, it } from 'bun:test';
import type { Client } from 'pg';

import {
  cleanupUser,
  connectAdmin,
  goRequest,
  parityPrerequisites,
  provisionGoSession,
} from './support/stacks';

/**
 * Differential check on how each stack turns a stored `date` into JSON.
 *
 * `transactions.date` is `timestamp without time zone`. Both stacks read the
 * same bytes and disagree about what timezone those bytes are in, which is
 * exactly the kind of difference no single-stack test can see: the Go tests
 * assert Go against the contract and pass, the fixtures record Hono and pass,
 * and the gap between them goes unnoticed until the flag flips.
 *
 * The Hono side is represented by **node-postgres**, the driver Hono's Drizzle
 * setup uses, reading the same row. That is the layer where the difference
 * originates — Hono's handler passes the value straight through — so it is a
 * faithful stand-in and avoids making this test depend on a Clerk session. The
 * limitation is real and stated in the README: this compares the data layer,
 * not the full Hono handler.
 */

const prerequisites = await parityPrerequisites();

describe.if(prerequisites.ok)('date serialization parity', () => {
  let admin: Client;
  let token: string;
  let userId: string;
  let accountId: string;

  /** The calendar day under test, stored as naive midnight. */
  const CALENDAR_DATE = '2026-08-07';

  beforeAll(async () => {
    admin = await connectAdmin();
    const session = await provisionGoSession(admin);
    token = session.token;
    userId = session.userId;

    const account = await goRequest(token, '/accounts', {
      method: 'POST',
      body: JSON.stringify({ name: 'Parity Account' }),
    });
    accountId = (account.body as { data: { id: string } }).data.id;

    await goRequest(token, '/transactions', {
      method: 'POST',
      body: JSON.stringify({
        amount: -12500,
        payee: 'Parity Market',
        notes: null,
        categoryId: null,
        date: CALENDAR_DATE,
        accountId,
        currency: 'ARS',
      }),
    });
  });

  afterAll(async () => {
    if (admin) {
      if (userId) {
        await cleanupUser(admin, userId);
      }
      await admin.end();
    }
  });

  it('stores the calendar date as naive midnight', async () => {
    const stored = await admin.query(
      "SELECT date FROM transactions WHERE payee = 'Parity Market' LIMIT 1"
    );

    expect(stored.rows).toHaveLength(1);
    const raw = stored.rows[0].date as Date;
    expect(raw.getFullYear()).toBe(2026);
    expect(raw.getMonth()).toBe(7);
    expect(raw.getDate()).toBe(7);
  });

  /**
   * The finding. Asserted as a difference rather than as equality, because
   * asserting equality would fail today and a failing test nobody can fix
   * inside the parity freeze gets deleted or ignored. This documents the
   * divergence, proves it is still present, and will fail loudly the day
   * someone changes either side — which is when it needs attention.
   */
  it('the two stacks disagree about the timezone of a naive timestamp', async () => {
    const viaDriver = (
      await admin.query(
        "SELECT date FROM transactions WHERE payee = 'Parity Market' LIMIT 1"
      )
    ).rows[0].date as Date;

    const listed = await goRequest(token, '/transactions');
    const [transaction] = (listed.body as { data: { date: string }[] }).data;

    const honoJson = viaDriver.toISOString();
    const goJson = transaction.date;

    // Both describe the same stored row.
    expect(goJson.slice(0, 10)).toBe(CALENDAR_DATE);

    // Go labels naive midnight as UTC; the driver labels it host-local. On a
    // host at UTC+0 these coincide, so the assertion is conditional on the
    // runner actually having an offset — otherwise it would fail in CI for a
    // reason unrelated to the defect.
    const hostOffsetMinutes = new Date(
      `${CALENDAR_DATE}T00:00:00`
    ).getTimezoneOffset();
    if (hostOffsetMinutes !== 0) {
      expect(honoJson).not.toBe(goJson);
    }
  });

  /**
   * Why the difference matters, expressed the way a user meets it: the same
   * row renders as two different calendar days.
   */
  it('renders as an earlier calendar day west of Greenwich', async () => {
    const listed = await goRequest(token, '/transactions');
    const [transaction] = (listed.body as { data: { date: string }[] }).data;

    const hostOffsetMinutes = new Date(
      `${CALENDAR_DATE}T00:00:00`
    ).getTimezoneOffset();
    // getTimezoneOffset is positive for zones behind UTC.
    if (hostOffsetMinutes > 0) {
      const renderedDay = new Date(transaction.date).getDate();
      expect(renderedDay).toBe(6);
    }
  });
});

if (!prerequisites.ok) {
  describe('date serialization parity', () => {
    it.skip(`skipped: ${prerequisites.reason}`, () => {});
  });
}
