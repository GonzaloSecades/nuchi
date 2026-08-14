import { readFile } from 'node:fs/promises';
import path from 'node:path';

import { describe, expect, it } from 'bun:test';

const HEADER_FILTER_FILES = [
  'components/account-filter.tsx',
  'components/date-filter.tsx',
  'components/filters.tsx',
] as const;

describe('header filter architecture', () => {
  it('does not couple interactive filters to summary query state', async () => {
    const sources = await Promise.all(
      HEADER_FILTER_FILES.map((file) =>
        readFile(path.join(process.cwd(), file), 'utf8')
      )
    );

    for (const source of sources) {
      expect(source).not.toContain('useGetSummary');
      expect(source).not.toContain('@/features/summary');
    }
  });
});
