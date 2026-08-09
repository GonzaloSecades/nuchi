import { describe, expect, it } from 'bun:test';

import {
  bulkFieldErrors,
  formatBulkErrorSummary,
} from '@/features/transactions/api/bulk-errors';
import { ApiError } from '@/lib/api-error';

function validationError(fields: unknown, status = 400): ApiError {
  return new ApiError('Request validation failed.', {
    status,
    statusText: 'Bad Request',
    url: '/api/transactions/bulk-create',
    resource: 'transactions',
    timestamp: new Date().toISOString(),
    errorData: {
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Request validation failed.',
        details: { fields },
      },
    },
  });
}

describe('bulkFieldErrors', () => {
  it('decodes an indexed row path', () => {
    const errors = bulkFieldErrors(
      validationError([{ path: '[3].amount', message: 'Amount is required.' }])
    );

    expect(errors).toEqual([
      { index: 3, field: 'amount', message: 'Amount is required.' },
    ]);
  });

  it('decodes a whole-array path', () => {
    const errors = bulkFieldErrors(
      validationError([{ path: '$', message: 'At most 500 rows.' }])
    );

    expect(errors).toEqual([
      { index: null, field: null, message: 'At most 500 rows.' },
    ]);
  });

  it('decodes bulk-delete ids paths', () => {
    const errors = bulkFieldErrors(
      validationError([
        { path: 'ids', message: 'At least one id is required.' },
        { path: 'ids[2]', message: 'Id must not be empty.' },
      ])
    );

    expect(errors).toEqual([
      { index: null, field: 'ids', message: 'At least one id is required.' },
      { index: 2, field: 'ids', message: 'Id must not be empty.' },
    ]);
  });

  it('decodes a bare indexed path with no field', () => {
    const errors = bulkFieldErrors(
      validationError([{ path: '[7]', message: 'Row is not an object.' }])
    );

    expect(errors).toEqual([
      { index: 7, field: null, message: 'Row is not an object.' },
    ]);
  });

  it('keeps multi-digit indexes intact', () => {
    expect(
      bulkFieldErrors(
        validationError([{ path: '[412].date', message: 'Bad date.' }])
      )[0].index
    ).toBe(412);
  });

  // A body the API could not parse carries no `details` at all, so callers
  // must be able to tell "no rows decoded" apart from "no error".
  it('returns nothing for a malformed body with no details', () => {
    const error = new ApiError('Request validation failed.', {
      status: 400,
      statusText: 'Bad Request',
      url: '/api/transactions/bulk-create',
      resource: 'transactions',
      timestamp: new Date().toISOString(),
      errorData: {
        error: {
          code: 'VALIDATION_ERROR',
          message: 'Request validation failed.',
        },
      },
    });

    expect(bulkFieldErrors(error)).toEqual([]);
  });

  it('returns nothing for a non-ApiError', () => {
    expect(bulkFieldErrors(new Error('network down'))).toEqual([]);
    expect(bulkFieldErrors(null)).toEqual([]);
  });

  it('skips malformed field entries but keeps the good ones', () => {
    const errors = bulkFieldErrors(
      validationError([
        { path: 123, message: 'nonsense' },
        null,
        { path: '[1].payee', message: 'Payee is required.' },
      ])
    );

    expect(errors).toEqual([
      { index: 1, field: 'payee', message: 'Payee is required.' },
    ]);
  });
});

describe('formatBulkErrorSummary', () => {
  it('returns null when there is nothing to report', () => {
    expect(formatBulkErrorSummary([])).toBeNull();
  });

  it('numbers rows from one', () => {
    expect(
      formatBulkErrorSummary([
        { index: 0, field: 'amount', message: 'Amount is required.' },
      ])
    ).toBe('Row 1 (amount): Amount is required.');
  });

  // The import screen submits in chunks of 500, so an error at index 3 of the
  // second chunk is row 504 of the user's file — not row 4.
  it('offsets rows by the number already submitted', () => {
    expect(
      formatBulkErrorSummary(
        [{ index: 3, field: 'amount', message: 'Amount is required.' }],
        { rowOffset: 500 }
      )
    ).toBe('Row 504 (amount): Amount is required.');
  });

  it('reports a whole-array problem without a row', () => {
    expect(
      formatBulkErrorSummary([
        { index: null, field: null, message: 'At most 500 rows.' },
      ])
    ).toBe('At most 500 rows.');
  });

  it('counts the tail instead of listing everything', () => {
    const many = Array.from({ length: 7 }, (_, index) => ({
      index,
      field: 'amount',
      message: 'Bad.',
    }));

    expect(formatBulkErrorSummary(many)).toBe(
      'Row 1 (amount): Bad.; Row 2 (amount): Bad.; Row 3 (amount): Bad.; and 4 more'
    );
  });
});
