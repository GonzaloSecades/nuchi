import { describe, expect, it } from 'bun:test';

import { ApiError } from '@/lib/api-error';

import { accountMutationErrorMessage } from './account-mutation-error';
import { accountPathParams } from './account-path-params';

const apiError = (code: string, message: string) =>
  new ApiError(message, {
    status: 409,
    statusText: 'Conflict',
    url: '/api/accounts',
    resource: 'accounts',
    timestamp: '2026-08-08T00:00:00.000Z',
    errorData: { error: { code, message } },
  });

describe('accountMutationErrorMessage', () => {
  it('uses the API message for a duplicate account name', () => {
    const error = apiError(
      'DUPLICATE_ACCOUNT_NAME',
      'You already have an account with this name.'
    );

    expect(accountMutationErrorMessage(error, 'Error creating account')).toBe(
      'You already have an account with this name.'
    );
  });

  it('keeps structured messages for other API errors', () => {
    const error = apiError('DB_ERROR', 'Database unavailable');

    expect(accountMutationErrorMessage(error, 'Error editing account')).toBe(
      'Database unavailable'
    );
  });

  it('uses the existing fallback for non-API errors', () => {
    expect(
      accountMutationErrorMessage(new Error('boom'), 'Error creating account')
    ).toBe('Error creating account');
  });
});

describe('accountPathParams', () => {
  it('builds the generated client path parameters', () => {
    expect(accountPathParams('account-1')).toEqual({
      path: { id: 'account-1' },
    });
  });

  it('rejects an absent id before a request can retain the path placeholder', () => {
    expect(() => accountPathParams()).toThrow('Account id is required');
    expect(() => accountPathParams('')).toThrow('Account id is required');
  });
});
