import { describe, expect, it } from 'bun:test';

import {
  DEFAULT_GO_API_URL,
  goApiRewrites,
  normalizeGoApiUrl,
} from '@/lib/go-api-rewrite';

describe('normalizeGoApiUrl', () => {
  it('accepts a bare origin', () => {
    expect(normalizeGoApiUrl('http://localhost:8080')).toBe(
      'http://localhost:8080'
    );
  });

  it('accepts https and a non-default port', () => {
    expect(normalizeGoApiUrl('https://api.example.com:8443')).toBe(
      'https://api.example.com:8443'
    );
  });

  // Next appends the matched path to the destination, so a trailing slash would
  // produce `//api/...`. Normalizing rather than rejecting keeps the common
  // copy-paste mistake from being a build failure.
  it('strips a trailing slash', () => {
    expect(normalizeGoApiUrl('http://localhost:8080/')).toBe(
      'http://localhost:8080'
    );
  });

  it('rejects a URL carrying a path, because the destination would nest it', () => {
    expect(() => normalizeGoApiUrl('http://localhost:8080/api')).toThrow(
      /origin-only/
    );
  });

  it('rejects a query string or fragment', () => {
    expect(() => normalizeGoApiUrl('http://localhost:8080?x=1')).toThrow(
      /origin-only/
    );
    expect(() => normalizeGoApiUrl('http://localhost:8080#frag')).toThrow(
      /origin-only/
    );
  });

  it('rejects a non-http protocol', () => {
    expect(() => normalizeGoApiUrl('ftp://localhost:8080')).toThrow(
      /http or https/
    );
  });

  it('rejects embedded credentials instead of silently dropping them', () => {
    expect(() =>
      normalizeGoApiUrl('http://deploy-user@localhost:8080')
    ).toThrow(/must not include username or password credentials/);
    expect(() =>
      normalizeGoApiUrl('https://deploy-user:secret@api.example.com')
    ).toThrow(/must not include username or password credentials/);
  });

  it('rejects a relative or malformed value', () => {
    // `new URL('localhost:8080')` parses: "localhost:" is read as the scheme,
    // not a host. So this is caught by the protocol check rather than the
    // parse, which is worth pinning because the message a user sees differs.
    expect(() => normalizeGoApiUrl('localhost:8080')).toThrow(/http or https/);
    expect(() => normalizeGoApiUrl('')).toThrow(/absolute URL/);
    expect(() => normalizeGoApiUrl('not a url')).toThrow(/absolute URL/);
  });

  it('names the offending value so the failure is actionable', () => {
    expect(() => normalizeGoApiUrl('http://localhost:8080/api')).toThrow(
      /http:\/\/localhost:8080\/api/
    );
  });
});

describe('goApiRewrites', () => {
  // #49/#50/#51 have moved the frontend and auth to Go, so leaving the
  // variable unset must produce a usable app rather than silently selecting
  // the legacy Clerk/Hono stack.
  it('rewrites to Go by default after the cutover', () => {
    const rewrites = goApiRewrites({});
    if (Array.isArray(rewrites)) throw new Error('expected phased rewrites');

    expect(rewrites.beforeFiles[0].destination).toBe(
      `${DEFAULT_GO_API_URL}/api/:path*`
    );
  });

  it('allows an explicit non-true value to disable the rewrite', () => {
    for (const USE_GO_API of ['false', 'TRUE', '1', 'yes', '']) {
      expect(goApiRewrites({ USE_GO_API })).toEqual([]);
    }
  });

  it('rewrites /api/* to the backend when enabled', () => {
    const rewrites = goApiRewrites({
      USE_GO_API: 'true',
      GO_API_URL: 'http://localhost:9999',
    });

    expect(Array.isArray(rewrites)).toBe(false);
    if (Array.isArray(rewrites)) return;

    expect(rewrites.beforeFiles).toEqual([
      {
        source: '/api/:path*',
        destination: 'http://localhost:9999/api/:path*',
      },
    ]);
  });

  // beforeFiles is the only phase that works here: afterFiles runs solely when
  // no filesystem route matched, and app/api/[[...route]] matches everything
  // under /api, so an afterFiles entry would never fire.
  it('places the rewrite in beforeFiles, not afterFiles', () => {
    const rewrites = goApiRewrites({ USE_GO_API: 'true' });
    if (Array.isArray(rewrites)) throw new Error('expected phased rewrites');

    expect(rewrites.beforeFiles).toHaveLength(1);
    expect(rewrites.afterFiles).toEqual([]);
    expect(rewrites.fallback).toEqual([]);
  });

  it('falls back to the documented default backend URL', () => {
    const rewrites = goApiRewrites({ USE_GO_API: 'true' });
    if (Array.isArray(rewrites)) throw new Error('expected phased rewrites');

    expect(rewrites.beforeFiles[0].destination).toBe(
      `${DEFAULT_GO_API_URL}/api/:path*`
    );
  });

  it('surfaces a bad backend URL at build time rather than as a 404 later', () => {
    expect(() =>
      goApiRewrites({ USE_GO_API: 'true', GO_API_URL: 'http://x/base' })
    ).toThrow(/origin-only/);
  });
});
