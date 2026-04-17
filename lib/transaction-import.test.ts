import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { parseImportedTransactionRows } from './transaction-import';

describe('parseImportedTransactionRows', () => {
  it('converts valid imported rows to API transaction rows', () => {
    const result = parseImportedTransactionRows([
      {
        amount: '12.34',
        date: '2026-04-17 13:45:00',
        payee: 'Coffee Shop',
      },
    ]);

    assert.deepEqual(result.errors, []);
    assert.deepEqual(result.data, [
      {
        amount: 12340,
        date: '2026-04-17',
        payee: 'Coffee Shop',
      },
    ]);
  });

  it('returns row errors instead of throwing for invalid dates and amounts', () => {
    const result = parseImportedTransactionRows([
      {
        amount: '',
        date: 'not-a-date',
        payee: '',
      },
    ]);

    assert.deepEqual(result.data, []);
    assert.equal(result.errors.length, 3);
    assert.deepEqual(
      result.errors.map((error) => error.field),
      ['amount', 'date', 'payee']
    );
  });
});
