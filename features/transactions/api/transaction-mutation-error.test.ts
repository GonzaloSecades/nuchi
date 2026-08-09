import { describe, expect, it } from 'bun:test';

import { transactionMutationErrorMessage } from '@/features/transactions/api/transaction-mutation-error';
import { transactionPathParams } from '@/features/transactions/api/transaction-path-params';
import { ApiError } from '@/lib/api-error';

const apiError = (errorData: unknown, status = 409) =>
  new ApiError('Failed to fetch transactions: 409 Conflict', {
    status,
    statusText: 'Conflict',
    url: '/api/transactions',
    resource: 'transactions',
    timestamp: '2026-08-09T00:00:00.000Z',
    errorData,
  });

describe('transactionMutationErrorMessage', () => {
  it('prefers the API structured message', () => {
    const error = apiError({
      error: {
        code: 'ACCOUNT_NOT_FOUND',
        message: 'That account no longer exists.',
      },
    });

    expect(
      transactionMutationErrorMessage(error, 'Error creating transaction')
    ).toBe('That account no longer exists.');
  });

  // The regression this exists to prevent. `toApiError` sets `message` to
  // "Failed to fetch transactions: 500 Internal Server Error" when the response
  // carries no envelope, and that must never reach a toast.
  it('falls back rather than surfacing the generic technical message', () => {
    const error = new ApiError(
      'Failed to fetch transactions: 500 Internal Server Error',
      {
        status: 500,
        statusText: 'Internal Server Error',
        url: '/api/transactions',
        resource: 'transactions',
        timestamp: '2026-08-09T00:00:00.000Z',
        errorData: null,
      }
    );

    expect(
      transactionMutationErrorMessage(error, 'Error creating transaction')
    ).toBe('Error creating transaction');
  });

  it('falls back when the envelope carries an empty message', () => {
    const error = apiError({
      error: { code: 'VALIDATION_ERROR', message: '' },
    });

    expect(
      transactionMutationErrorMessage(error, 'Error editing transaction')
    ).toBe('Error editing transaction');
  });

  it('falls back when the message is not a string', () => {
    const error = apiError({ error: { code: 'X', message: { nested: true } } });

    expect(
      transactionMutationErrorMessage(error, 'Error editing transaction')
    ).toBe('Error editing transaction');
  });

  it('falls back for a non-API error', () => {
    expect(
      transactionMutationErrorMessage(
        new Error('network down'),
        'Error creating transaction'
      )
    ).toBe('Error creating transaction');
  });
});

describe('transactionPathParams', () => {
  it('builds the generated client path parameters', () => {
    expect(transactionPathParams('txn-1')).toEqual({ path: { id: 'txn-1' } });
  });

  // openapi-fetch leaves `{id}` in the URL when the value is missing, so
  // without this the request goes out as `/api/transactions/%7Bid%7D`.
  it('rejects an absent id before a request can keep the placeholder', () => {
    expect(() => transactionPathParams()).toThrow('Transaction id is required');
    expect(() => transactionPathParams('')).toThrow(
      'Transaction id is required'
    );
  });
});
