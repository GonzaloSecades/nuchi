import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { resolveApiBaseUrl } from './api-base-url';

describe('resolveApiBaseUrl', () => {
  it('uses same-origin requests in the browser', () => {
    assert.equal(
      resolveApiBaseUrl({
        isBrowser: true,
        publicApiUrl: 'http://localhost:3000',
      }),
      ''
    );
  });
});
