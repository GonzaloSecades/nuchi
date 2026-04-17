import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
  isContentLengthTooLarge,
  parseDateRangeQuery,
  parseStrictDate,
} from './transaction-route-utils';

describe('parseStrictDate', () => {
  it('accepts yyyy-MM-dd dates', () => {
    const parsed = parseStrictDate('2026-04-17');

    assert.ok(parsed instanceof Date);
    assert.equal(parsed?.getFullYear(), 2026);
    assert.equal(parsed?.getMonth(), 3);
    assert.equal(parsed?.getDate(), 17);
  });

  it('rejects malformed and impossible dates', () => {
    assert.equal(parseStrictDate('2026-4-17'), null);
    assert.equal(parseStrictDate('2026-02-31'), null);
    assert.equal(parseStrictDate(undefined), null);
  });

  it('can parse the end of a calendar day for inclusive to filters', () => {
    const parsed = parseStrictDate('2026-04-17', { boundary: 'end' });

    assert.equal(parsed?.getHours(), 23);
    assert.equal(parsed?.getMinutes(), 59);
    assert.equal(parsed?.getSeconds(), 59);
    assert.equal(parsed?.getMilliseconds(), 999);
  });
});

describe('isContentLengthTooLarge', () => {
  it('detects content length values above the limit', () => {
    assert.equal(isContentLengthTooLarge('101', 100), true);
    assert.equal(isContentLengthTooLarge('100', 100), false);
  });

  it('ignores absent or non-numeric content length values', () => {
    assert.equal(isContentLengthTooLarge(undefined, 100), false);
    assert.equal(isContentLengthTooLarge('not-a-number', 100), false);
  });
});

describe('parseDateRangeQuery', () => {
  const defaultFrom = new Date(2026, 3, 1);
  const defaultTo = new Date(2026, 3, 30);

  it('rejects malformed date filters', () => {
    const result = parseDateRangeQuery({
      from: '2026-02-31',
      to: undefined,
      defaultFrom,
      defaultTo,
    });

    assert.equal(result.ok, false);
    assert.equal(result.ok ? undefined : result.error.code, 'INVALID_QUERY');
  });

  it('parses to filters as inclusive end-of-day dates', () => {
    const result = parseDateRangeQuery({
      from: '2026-04-17',
      to: '2026-04-17',
      defaultFrom,
      defaultTo,
    });

    assert.equal(result.ok, true);
    assert.equal(result.ok ? result.endDate.getHours() : undefined, 23);
    assert.equal(result.ok ? result.endDate.getMilliseconds() : undefined, 999);
  });
});
