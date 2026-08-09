import { describe, expect, it } from 'bun:test';

import { ApiError } from '@/lib/api-error';

import { categoryMutationErrorMessage } from './category-mutation-error';
import { categoryPathParams } from './category-path-params';

const apiError = (message: string) =>
  new ApiError(message, {
    status: 409,
    statusText: 'Conflict',
    url: '/api/categories',
    resource: 'categories',
    timestamp: '2026-08-09T00:00:00.000Z',
    errorData: {
      error: { code: 'DUPLICATE_CATEGORY_NAME', message },
    },
  });

describe('categoryMutationErrorMessage', () => {
  it('keeps the API message used by the existing create hook', () => {
    expect(
      categoryMutationErrorMessage(
        apiError('You already have a category with this name.'),
        'Error creating category'
      )
    ).toBe('You already have a category with this name.');
  });

  it('uses the existing fallback when no structured message exists', () => {
    expect(
      categoryMutationErrorMessage(
        new Error('network failure'),
        'Error creating category'
      )
    ).toBe('Error creating category');
  });
});

describe('categoryPathParams', () => {
  it('builds generated-client path parameters', () => {
    expect(categoryPathParams('category-1')).toEqual({
      path: { id: 'category-1' },
    });
  });

  it('rejects a missing id before the path placeholder reaches fetch', () => {
    expect(() => categoryPathParams()).toThrow('Category id is required');
    expect(() => categoryPathParams('')).toThrow('Category id is required');
  });
});
