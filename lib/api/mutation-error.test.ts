import { describe, expect, it } from 'bun:test';

import { ApiError } from '@/lib/api-error';
import { mutationErrorMessage } from '@/lib/api/mutation-error';

const apiError = (errorData: unknown) =>
  new ApiError('Failed to fetch resources: 409 Conflict', {
    status: 409,
    statusText: 'Conflict',
    url: '/api/resources',
    resource: 'resources',
    timestamp: '2026-08-09T00:00:00.000Z',
    errorData,
  });

describe('mutationErrorMessage', () => {
  it('prefers the API structured message', () => {
    const error = apiError({
      error: {
        code: 'RESOURCE_CONFLICT',
        message: 'That resource already exists.',
      },
    });

    expect(mutationErrorMessage(error, 'Error creating resource')).toBe(
      'That resource already exists.'
    );
  });

  it('prefers a caller override for a matching API code', () => {
    const error = apiError({
      error: { code: 'RESOURCE_CONFLICT', message: 'Server wording.' },
    });

    expect(
      mutationErrorMessage(error, 'Error creating resource', {
        RESOURCE_CONFLICT: 'Caller wording.',
      })
    ).toBe('Caller wording.');
  });

  it('falls back rather than surfacing a generic technical message', () => {
    const error = new ApiError(
      'Failed to fetch resources: 500 Internal Server Error',
      {
        status: 500,
        statusText: 'Internal Server Error',
        url: '/api/resources',
        resource: 'resources',
        timestamp: '2026-08-09T00:00:00.000Z',
        errorData: null,
      }
    );

    expect(mutationErrorMessage(error, 'Error creating resource')).toBe(
      'Error creating resource'
    );
  });

  it('falls back when the structured message is empty', () => {
    const error = apiError({
      error: { code: 'VALIDATION_ERROR', message: '' },
    });

    expect(mutationErrorMessage(error, 'Error editing resource')).toBe(
      'Error editing resource'
    );
  });

  it('falls back when the structured message is not a string', () => {
    const error = apiError({
      error: { code: 'VALIDATION_ERROR', message: { nested: true } },
    });

    expect(mutationErrorMessage(error, 'Error editing resource')).toBe(
      'Error editing resource'
    );
  });

  it('falls back for a non-API error', () => {
    expect(
      mutationErrorMessage(new Error('network down'), 'Error creating resource')
    ).toBe('Error creating resource');
  });

  for (const inherited of ['toString', 'constructor', 'hasOwnProperty']) {
    it(`does not treat the inherited property ${inherited} as an override`, () => {
      const error = apiError({
        error: { code: inherited, message: 'Server wording.' },
      });

      expect(
        mutationErrorMessage(error, 'Error creating resource', {
          SOMETHING_ELSE: 'unused',
        })
      ).toBe('Server wording.');
    });
  }

  it('honours an override deliberately named after an Object member', () => {
    const error = apiError({
      error: { code: 'toString', message: 'Server wording.' },
    });

    expect(
      mutationErrorMessage(error, 'Error creating resource', {
        toString: 'Caller wording.',
      })
    ).toBe('Caller wording.');
  });
});
