import { describe, expect, it } from 'bun:test';

import {
  DEFAULT_SIGNED_IN_PATH,
  isAuthPath,
  safeRedirectTarget,
} from '@/lib/auth/redirect';

describe('safeRedirectTarget', () => {
  it('keeps a same-origin path', () => {
    expect(safeRedirectTarget('/transactions')).toBe('/transactions');
  });

  it('keeps the query string and hash of a same-origin path', () => {
    expect(safeRedirectTarget('/transactions?accountId=abc#row-3')).toBe(
      '/transactions?accountId=abc#row-3'
    );
  });

  it('falls back when nothing was requested', () => {
    expect(safeRedirectTarget(null)).toBe(DEFAULT_SIGNED_IN_PATH);
    expect(safeRedirectTarget(undefined)).toBe(DEFAULT_SIGNED_IN_PATH);
    expect(safeRedirectTarget('')).toBe(DEFAULT_SIGNED_IN_PATH);
  });

  // The whole point of the helper. Each of these navigates off-site while
  // looking path-like enough to survive a naive `startsWith('/')` check.
  const offSiteTargets: Array<[label: string, target: string]> = [
    ['absolute https URL', 'https://evil.example/login'],
    ['protocol-relative URL', '//evil.example/login'],
    ['single-slash protocol', 'https:/evil.example'],
    ['backslash prefix', '\\\\evil.example'],
    ['slash-backslash prefix', '/\\evil.example'],
    ['scheme with no slashes', 'javascript:alert(1)'],
  ];

  for (const [label, target] of offSiteTargets) {
    it(`rejects an off-site target (${label})`, () => {
      expect(safeRedirectTarget(target)).toBe(DEFAULT_SIGNED_IN_PATH);
    });
  }

  // A stale `redirect` pointing back at an auth page would return the user to
  // sign-in immediately after signing in.
  for (const target of [
    '/sign-in',
    '/sign-up',
    '/reset-password',
    '/verify-email',
  ]) {
    it(`rejects the auth page ${target} as a target`, () => {
      expect(safeRedirectTarget(target)).toBe(DEFAULT_SIGNED_IN_PATH);
    });
  }

  it('rejects an auth page carrying a query string', () => {
    expect(safeRedirectTarget('/sign-in?redirect=/transactions')).toBe(
      DEFAULT_SIGNED_IN_PATH
    );
  });

  // Only exact matches are auth pages; a dashboard route that merely starts
  // with the same characters must still be reachable.
  it('keeps a path that only shares a prefix with an auth page', () => {
    expect(safeRedirectTarget('/sign-in-help')).toBe('/sign-in-help');
  });
});

describe('isAuthPath', () => {
  it('recognizes the signed-out pages', () => {
    expect(isAuthPath('/sign-in')).toBe(true);
    expect(isAuthPath('/reset-password')).toBe(true);
  });

  it('does not match dashboard routes or prefixes', () => {
    expect(isAuthPath('/transactions')).toBe(false);
    expect(isAuthPath('/sign-in-help')).toBe(false);
    expect(isAuthPath('/')).toBe(false);
  });
});
