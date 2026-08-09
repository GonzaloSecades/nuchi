import { describe, expect, it } from 'bun:test';

import { createAccessTokenStore } from '@/lib/api/token-store';

describe('access token store', () => {
  // A fresh store per case rather than resetting a shared one, which is why the
  // module no longer needs a test-only reset export.
  it('stores and clears the token', () => {
    const store = createAccessTokenStore();

    expect(store.get()).toBeNull();
    store.set('abc');
    expect(store.get()).toBe('abc');
    store.clear();
    expect(store.get()).toBeNull();
  });

  it('notifies subscribers on set and clear', () => {
    const store = createAccessTokenStore();
    const seen: Array<string | null> = [];
    store.subscribe((token) => seen.push(token));

    store.set('abc');
    store.clear();

    expect(seen).toEqual(['abc', null]);
  });

  it('stops notifying after unsubscribe', () => {
    const store = createAccessTokenStore();
    const seen: Array<string | null> = [];
    const unsubscribe = store.subscribe((token) => seen.push(token));

    store.set('one');
    unsubscribe();
    store.set('two');

    expect(seen).toEqual(['one']);
  });

  // set() runs inside the refresh path, so a throwing listener must not fail the
  // renewal or stop the other listeners from hearing about it.
  it('isolates a throwing subscriber', () => {
    const store = createAccessTokenStore();
    const seen: Array<string | null> = [];
    store.subscribe(() => {
      throw new Error('listener blew up');
    });
    store.subscribe((token) => seen.push(token));

    expect(() => store.set('abc')).not.toThrow();
    expect(seen).toEqual(['abc']);
    expect(store.get()).toBe('abc');
  });

  it('keeps separate stores independent', () => {
    const first = createAccessTokenStore();
    const second = createAccessTokenStore();

    first.set('first-token');

    expect(second.get()).toBeNull();
  });
});
