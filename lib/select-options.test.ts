import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { mergeCreatedSelectOption } from './select-options';

describe('mergeCreatedSelectOption', () => {
  it('keeps a newly created option selectable before options refetch', () => {
    const options = [{ label: 'Cash', value: 'account_1' }];
    const createdOption = { label: 'Brokerage', value: 'account_2' };

    assert.deepEqual(mergeCreatedSelectOption(options, createdOption), [
      createdOption,
      ...options,
    ]);
  });
});
