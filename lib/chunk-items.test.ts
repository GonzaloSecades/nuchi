import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { chunkItems } from './chunk-items';

describe('chunkItems', () => {
  it('splits items into fixed-size batches', () => {
    const items = Array.from({ length: 501 }, (_, index) => index);

    const chunks = chunkItems(items, 500);

    assert.equal(chunks.length, 2);
    assert.equal(chunks[0].length, 500);
    assert.deepEqual(chunks[1], [500]);
  });
});
