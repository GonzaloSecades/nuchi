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

});
