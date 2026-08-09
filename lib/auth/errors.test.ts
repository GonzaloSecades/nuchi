import { describe, expect, it } from 'bun:test';

import { ApiError } from '@/lib/api-error';
import { authErrorMessage, validationFieldErrors } from '@/lib/auth/errors';

const apiError = (errorData: unknown, message = 'Request failed.') =>
  new ApiError(message, {
    status: 400,
    statusText: 'Bad Request',
    url: '/api/auth/login',
    resource: 'session',
    timestamp: '2026-08-09T00:00:00.000Z',
    errorData,
  });

describe('authErrorMessage', () => {
  it('prefers an override for a known code', () => {
    const error = apiError({
      error: { code: 'EMAIL_NOT_VERIFIED', message: 'Server wording.' },
    });

    expect(
      authErrorMessage(error, { EMAIL_NOT_VERIFIED: 'Page wording.' })
    ).toBe('Page wording.');
  });

  it('falls back to the API message when no override matches', () => {
    const error = apiError(
      {
        error: { code: 'UNAUTHORIZED', message: 'Invalid email or password.' },
      },
      'Invalid email or password.'
    );

    expect(authErrorMessage(error, { EMAIL_NOT_VERIFIED: 'unused' })).toBe(
      'Invalid email or password.'
    );
  });

  it('falls back to a generic message for a non-API error', () => {
    expect(authErrorMessage(new Error('network down'))).toBe(
      'Something went wrong. Please try again.'
    );
  });

  /**
   * `in` walks the prototype chain, so these codes would match an override
   * nobody wrote and return an Object member — a function — where the UI
   * expects a message.
   */
  for (const inherited of ['toString', 'constructor', 'hasOwnProperty']) {
    it(`does not treat the inherited property ${inherited} as an override`, () => {
      const error = apiError(
        { error: { code: inherited, message: 'Server wording.' } },
        'Server wording.'
      );

      const message = authErrorMessage(error, { SOMETHING_ELSE: 'unused' });

      // With `in`, this returned `Object.prototype[inherited]` — a function.
      expect(typeof message).toBe('string');
      expect(message).toBe('Server wording.');
    });
  }

  it('still honours an override deliberately named after an Object member', () => {
    const error = apiError({
      error: { code: 'toString', message: 'Server wording.' },
    });

    expect(authErrorMessage(error, { toString: 'Explicit override.' })).toBe(
      'Explicit override.'
    );
  });
});

describe('validationFieldErrors', () => {
  it('extracts per-field messages', () => {
    const error = apiError({
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Request validation failed.',
        details: {
          fields: [{ path: 'email', message: 'Invalid email address.' }],
        },
      },
    });

    expect(validationFieldErrors(error)).toEqual([
      { path: 'email', message: 'Invalid email address.' },
    ]);
  });

  // A malformed body carries no `details` at all.
  it('returns nothing when details are absent', () => {
    const error = apiError({
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Request validation failed.',
      },
    });

    expect(validationFieldErrors(error)).toEqual([]);
  });

  it('returns nothing for a non-API error', () => {
    expect(validationFieldErrors(new Error('boom'))).toEqual([]);
  });

  it('skips malformed entries but keeps the good ones', () => {
    const error = apiError({
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Request validation failed.',
        details: {
          fields: [
            { path: 42, message: 'nonsense' },
            null,
            { path: 'password', message: 'Too short.' },
          ],
        },
      },
    });

    expect(validationFieldErrors(error)).toEqual([
      { path: 'password', message: 'Too short.' },
    ]);
  });
});
