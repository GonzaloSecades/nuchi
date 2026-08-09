import { describe, expect, it } from 'bun:test';

import {
  toBulkTransactionInput,
  toTransactionInput,
} from '@/features/transactions/api/transaction-payload';

const baseValues = {
  date: new Date(2026, 7, 9, 14, 30),
  accountId: 'account-1',
  categoryId: 'category-1',
  payee: 'Market',
  amount: -12500,
  notes: 'weekly shop',
};

describe('toTransactionInput', () => {
  it('produces the contract shape', () => {
    expect(toTransactionInput(baseValues)).toEqual({
      amount: -12500,
      payee: 'Market',
      notes: 'weekly shop',
      date: '2026-08-09',
      accountId: 'account-1',
      categoryId: 'category-1',
      currency: 'ARS',
    });
  });

  // The contract rejects a missing currency rather than defaulting it, and
  // nothing upstream of the hooks supplies one.
  it('always sets the currency', () => {
    expect(toTransactionInput(baseValues).currency).toBe('ARS');
  });

  // `toISOString().slice(0, 10)` would convert local midnight to the previous
  // day for any user east of Greenwich. Local midnight must stay the same
  // calendar day it was picked as, in every zone the test runs in.
  it('keeps the local calendar date at midnight', () => {
    const midnight = new Date(2026, 7, 9, 0, 0, 0);
    expect(toTransactionInput({ ...baseValues, date: midnight }).date).toBe(
      '2026-08-09'
    );
  });

  it('keeps the local calendar date just before midnight', () => {
    const lateEvening = new Date(2026, 7, 9, 23, 59, 59);
    expect(toTransactionInput({ ...baseValues, date: lateEvening }).date).toBe(
      '2026-08-09'
    );
  });

  it('zero-pads single-digit months and days', () => {
    const january = new Date(2026, 0, 5, 12, 0);
    expect(toTransactionInput({ ...baseValues, date: january }).date).toBe(
      '2026-01-05'
    );
  });

  // The contract types both as nullable; the form leaves them undefined when
  // untouched, and `undefined` would drop the key rather than send null.
  it('normalizes omitted notes and category to null', () => {
    const sparse = toTransactionInput({
      date: baseValues.date,
      accountId: 'account-1',
      payee: 'Market',
      amount: 1000,
    });

    expect(sparse.notes).toBeNull();
    expect(sparse.categoryId).toBeNull();
  });

  it('passes an explicit null through unchanged', () => {
    const cleared = toTransactionInput({
      ...baseValues,
      notes: null,
      categoryId: null,
    });

    expect(cleared.notes).toBeNull();
    expect(cleared.categoryId).toBeNull();
  });

  it('preserves the sign of an expense', () => {
    expect(toTransactionInput({ ...baseValues, amount: -1 }).amount).toBe(-1);
  });
});

describe('toBulkTransactionInput', () => {
  // The CSV parser already emits `yyyy-MM-dd`, so this path must pass the date
  // through untouched rather than re-parsing it into a Date and back.
  it('passes the imported date string through unchanged', () => {
    const [row] = toBulkTransactionInput([
      { amount: -5000, date: '2026-08-09', payee: 'Market', accountId: 'a1' },
    ]);

    expect(row.date).toBe('2026-08-09');
  });

  it('adds the required currency and null defaults to every row', () => {
    const rows = toBulkTransactionInput([
      { amount: 1000, date: '2026-01-01', payee: 'One', accountId: 'a1' },
      { amount: -2000, date: '2026-01-02', payee: 'Two', accountId: 'a1' },
    ]);

    expect(rows).toHaveLength(2);
    for (const row of rows) {
      expect(row.currency).toBe('ARS');
      expect(row.notes).toBeNull();
      expect(row.categoryId).toBeNull();
    }
  });

  it('returns an empty array unchanged', () => {
    expect(toBulkTransactionInput([])).toEqual([]);
  });

  /**
   * The legacy bulk hook accepted the full transaction insert shape, so
   * hardcoding these to null silently discarded whatever a caller passed —
   * a payload accepted and then quietly altered.
   */
  it('preserves notes and categoryId when supplied', () => {
    const [row] = toBulkTransactionInput([
      {
        amount: -5000,
        date: '2026-08-09',
        payee: 'Market',
        accountId: 'a1',
        notes: 'weekly shop',
        categoryId: 'c1',
      },
    ]);

    expect(row.notes).toBe('weekly shop');
    expect(row.categoryId).toBe('c1');
  });

  it('preserves them independently of one another', () => {
    const [onlyNotes] = toBulkTransactionInput([
      {
        amount: 1,
        date: '2026-08-09',
        payee: 'A',
        accountId: 'a1',
        notes: 'kept',
      },
    ]);
    const [onlyCategory] = toBulkTransactionInput([
      {
        amount: 1,
        date: '2026-08-09',
        payee: 'A',
        accountId: 'a1',
        categoryId: 'c1',
      },
    ]);

    expect(onlyNotes).toMatchObject({ notes: 'kept', categoryId: null });
    expect(onlyCategory).toMatchObject({ notes: null, categoryId: 'c1' });
  });

  it('normalizes an explicit null the same as an omitted value', () => {
    const [row] = toBulkTransactionInput([
      {
        amount: 1,
        date: '2026-08-09',
        payee: 'A',
        accountId: 'a1',
        notes: null,
        categoryId: null,
      },
    ]);

    expect(row.notes).toBeNull();
    expect(row.categoryId).toBeNull();
  });

  // The CSV importer supplies neither, and must keep working untouched.
  it('still normalizes omitted fields for the CSV path', () => {
    const [row] = toBulkTransactionInput([
      { amount: 1, date: '2026-08-09', payee: 'A', accountId: 'a1' },
    ]);

    expect(row.notes).toBeNull();
    expect(row.categoryId).toBeNull();
    expect(row.currency).toBe('ARS');
  });
});
