import { describe, expect, it } from 'bun:test';

import {
  calendarDateFromApi,
  calendarDateFromLocalDate,
  localDateFromApi,
  localDateFromCalendarDate,
} from '@/features/transactions/api/transaction-date';

/** The value the Go API returns for a transaction stored on 2026-08-07. */
const API_INSTANT = '2026-08-07T00:00:00Z';

describe('calendarDateFromApi', () => {
  it('reads the calendar date the API labelled', () => {
    expect(calendarDateFromApi(API_INSTANT)).toBe('2026-08-07');
  });

  it('passes an already-normalized calendar date through', () => {
    expect(calendarDateFromApi('2026-08-07')).toBe('2026-08-07');
  });

  it('zero-pads single-digit months and days', () => {
    expect(calendarDateFromApi('2026-01-05T00:00:00Z')).toBe('2026-01-05');
  });

  it('reads UTC parts even for a non-midnight instant', () => {
    expect(calendarDateFromApi('2026-08-07T23:30:00Z')).toBe('2026-08-07');
  });

  it('rejects a value that is not a date', () => {
    expect(() => calendarDateFromApi('not-a-date')).toThrow(
      'Not a valid transaction date'
    );
  });
});

describe('localDateFromCalendarDate', () => {
  it('builds local midnight on the intended day', () => {
    const date = localDateFromCalendarDate('2026-08-07');

    expect(date.getFullYear()).toBe(2026);
    expect(date.getMonth()).toBe(7);
    expect(date.getDate()).toBe(7);
    expect(date.getHours()).toBe(0);
  });

  it('rejects anything that is not yyyy-MM-dd', () => {
    expect(() => localDateFromCalendarDate(API_INSTANT)).toThrow(
      'Not a calendar date'
    );
  });
});

describe('calendarDateFromLocalDate', () => {
  it('reads the day the user picked', () => {
    expect(calendarDateFromLocalDate(new Date(2026, 7, 7))).toBe('2026-08-07');
  });

  it('is unaffected by the time of day', () => {
    expect(calendarDateFromLocalDate(new Date(2026, 7, 7, 23, 59, 59))).toBe(
      '2026-08-07'
    );
  });

  it('rejects an invalid date', () => {
    expect(() => calendarDateFromLocalDate(new Date('nope'))).toThrow(
      'Not a valid date'
    );
  });
});

describe('the edit round trip', () => {
  /**
   * The finding. Loading a Go transaction into the edit sheet and saving it
   * unchanged used to move it back a day for anyone west of Greenwich, once
   * per save.
   */
  it('returns the same calendar date it was given', () => {
    const picker = localDateFromApi(API_INSTANT);

    expect(calendarDateFromLocalDate(picker)).toBe('2026-08-07');
  });

  it('is stable across repeated edits', () => {
    let value = API_INSTANT;
    for (let i = 0; i < 5; i += 1) {
      value = calendarDateFromLocalDate(localDateFromApi(value));
    }

    expect(value).toBe('2026-08-07');
  });

  it('keeps a newly picked date on its selected day', () => {
    const picked = new Date(2026, 7, 9, 14, 30);

    expect(calendarDateFromLocalDate(picked)).toBe('2026-08-09');
  });
});
