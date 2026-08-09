import { describe, expect, it } from 'bun:test';

import { requiredPathParams } from '@/lib/api/path-params';

describe('requiredPathParams', () => {
  it('builds the generated client path parameters', () => {
    expect(requiredPathParams('Account', 'account-1')).toEqual({
      path: { id: 'account-1' },
    });
  });

  for (const resource of ['Account', 'Category', 'Transaction']) {
    it(`rejects an absent ${resource.toLowerCase()} id`, () => {
      expect(() => requiredPathParams(resource)).toThrow(
        `${resource} id is required`
      );
      expect(() => requiredPathParams(resource, '')).toThrow(
        `${resource} id is required`
      );
    });
  }
});
