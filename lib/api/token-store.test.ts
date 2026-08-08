import { beforeEach, describe, expect, it } from 'bun:test';

import {
  clearAccessToken,
  getAccessToken,
  resetAccessTokenStore,
  setAccessToken,
  subscribeToAccessToken,
} from '@/lib/api/token-store';

describe('token store', () => {
  beforeEach(() => {
    resetAccessTokenStore();
  });

  it('stores and clears the token', () => {
    expect(getAccessToken()).toBeNull();
    setAccessToken('abc');
    expect(getAccessToken()).toBe('abc');
    clearAccessToken();
    expect(getAccessToken()).toBeNull();
  });

  it('notifies subscribers on set and clear', () => {
    const seen: Array<string | null> = [];
    subscribeToAccessToken((token) => seen.push(token));

    setAccessToken('abc');
    clearAccessToken();

    expect(seen).toEqual(['abc', null]);
  });

  it('stops notifying after unsubscribe', () => {
    const seen: Array<string | null> = [];
    const unsubscribe = subscribeToAccessToken((token) => seen.push(token));

    setAccessToken('one');
    unsubscribe();
    setAccessToken('two');

    expect(seen).toEqual(['one']);
  });

  // setAccessToken runs inside the refresh path, so a throwing listener must
  // not fail the renewal or stop the other listeners from hearing about it.
  it('isolates a throwing subscriber', () => {
    const seen: Array<string | null> = [];
    subscribeToAccessToken(() => {
      throw new Error('listener blew up');
    });
    subscribeToAccessToken((token) => seen.push(token));

    expect(() => setAccessToken('abc')).not.toThrow();
    expect(seen).toEqual(['abc']);
    expect(getAccessToken()).toBe('abc');
  });
});
