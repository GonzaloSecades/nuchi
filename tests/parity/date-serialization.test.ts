import { afterAll, beforeAll, describe, expect, it } from 'bun:test';
import type { Client } from 'pg';

import { classifyDivergence, DATE_SERIALIZATION_TIMEZONE } from './divergences';
import {
  CAN_OBSERVE_TIMEZONE_DIVERGENCE,
  cleanupUser,
  connectAdmin,
  expectData,
  expectOk,
  goRequest,
  HOST_UTC_OFFSET_MINUTES,
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

/** The calendar day under test, stored as naive midnight. */
const CALENDAR_DATE = '2026-08-07';

describe.if(prerequisites.ok)('date serialization parity', () => {
  let admin: Client;
  let token: string;
  let userId: string;
  let accountId: string;

  beforeAll(async () => {
    admin = await connectAdmin();
    const session = await provisionGoSession(admin);
    token = session.token;
    userId = session.userId;

    const account = await goRequest(token, '/accounts', {
      method: 'POST',
      body: JSON.stringify({ name: 'Parity Account' }),
    });
    accountId = expectData<{ id: string }>(account, 'create parity account').id;

    const created = await goRequest(token, '/transactions', {
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
    expectOk(created, 'create parity transaction');
  });

  afterAll(async () => {
    if (admin) {
      if (userId) {
        await cleanupUser(admin, userId);
      }
      await admin.end();
    }
  });

  /**
   * Pinned to the account this run created. Selecting on `payee` alone would
   * pick up a row from an earlier run that failed to clean up, and compare the
   * wrong transaction.
   */
  async function storedDate(): Promise<Date> {
    const stored = await admin.query(
      'SELECT date FROM transactions WHERE account_id = $1 AND payee = $2 LIMIT 1',
      [accountId, 'Parity Market']
    );
    expect(stored.rows).toHaveLength(1);
    return stored.rows[0].date as Date;
  }

  async function goDate(): Promise<string> {
    const listed = await goRequest(token, '/transactions');
    const data = expectData<Array<{ date: string; accountId: string }>>(
      listed,
      'list parity transactions'
    );
    const match = data.find(
      (transaction) => transaction.accountId === accountId
    );
    if (match === undefined) {
      throw new Error('parity: the created transaction was not listed');
    }
    return match.date;
  }

  it('stores the calendar date as naive midnight', async () => {
    const raw = await storedDate();

    expect(raw.getFullYear()).toBe(2026);
    expect(raw.getMonth()).toBe(7);
    expect(raw.getDate()).toBe(7);
  });

  it('reports the same calendar day through the Go API', async () => {
    expect((await goDate()).slice(0, 10)).toBe(CALENDAR_DATE);
  });

  /**
   * The finding, asserted as a difference rather than as equality: an
   * assertion that fails today, inside a parity freeze that forbids fixing it,
   * gets ignored and then deleted. Written this way it documents the
   * divergence, proves it is still present, and fails loudly the day someone
   * changes either side — which is when it needs attention.
   *
   * Skipped rather than passed at UTC. There both stacks produce the identical
   * string, so there is no difference to observe and an assertion here would be
   * a green that compared nothing.
   */
  describe.if(CAN_OBSERVE_TIMEZONE_DIVERGENCE)(
    `observed at UTC offset ${-HOST_UTC_OFFSET_MINUTES / 60}h`,
    () => {
      it('the two stacks disagree about the timezone of a naive timestamp', async () => {
        const honoJson = (await storedDate()).toISOString();
        const goJson = await goDate();

        expect(honoJson).not.toBe(goJson);
        // Same instant-of-day claim, different labels: Go says UTC midnight,
        // the driver says local midnight.
        expect(goJson).toContain('T00:00:00');
        expect(honoJson).not.toContain('T00:00:00');
      });
    }
  );

  /**
   * Why the difference matters, expressed the way a user meets it. Formatted in
   * a fixed zone rather than the host's, so this asserts the same thing on
   * every machine instead of depending on where the runner happens to be.
   */
  it('renders as the previous calendar day in Argentina', async () => {
    const goJson = await goDate();
    const rendered = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'America/Argentina/Buenos_Aires',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(new Date(goJson));

    expect(rendered).toBe('2026-08-06');
  });

  /**
   * Wires the observation to the registry. If someone reclassifies this as an
   * expected divergence, this fails — which is the intent: it is a regression
   * against Hono, and marking it expected would retire the only check for it.
   */
  it('is registered as an open divergence, not a decision', () => {
    expect(classifyDivergence(DATE_SERIALIZATION_TIMEZONE)).toBe('open');
  });
});

if (!prerequisites.ok) {
  describe('date serialization parity', () => {
    it.skip(`skipped — ${prerequisites.reason}`, () => {});
  });
}

if (prerequisites.ok && !CAN_OBSERVE_TIMEZONE_DIVERGENCE) {
  describe('date serialization parity', () => {
    it.skip('skipped the stack comparison — the runner is at UTC, where both stacks serialize a naive timestamp identically', () => {});
  });
}
