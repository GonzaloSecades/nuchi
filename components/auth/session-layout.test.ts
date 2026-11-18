import { describe, expect, it } from 'bun:test';
import { readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';

describe('protected dashboard layout', () => {
  it('mounts query-owning sheets only inside the session guard', () => {
    const root = readFileSync(resolve('app/layout.tsx'), 'utf8');
    const dashboard = readFileSync(
      resolve('app/(dashboard)/layout.tsx'),
      'utf8'
    );

    // Sheets request accounts/categories even while closed. Mounting them at
    // the root races those requests against refresh-cookie session bootstrap.
    expect(root).not.toContain('SheetProvider');
    expect(dashboard).toMatch(
      /<SessionGuard>[\s\S]*<SheetProvider\s*\/>[\s\S]*<\/SessionGuard>/
    );
  });

  it('has a dashboard page for every navigation destination', () => {
    const navigation = readFileSync(
      resolve('components/navigation.tsx'),
      'utf8'
    );
    const destinations = [...navigation.matchAll(/href: '([^']+)'/g)];
    expect(destinations.length).toBeGreaterThan(0);
    for (const [, href] of destinations) {
      expect(existsSync(resolve(`app/(dashboard)${href}/page.tsx`))).toBe(true);
    }
  });
});
