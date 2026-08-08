import createClient from 'openapi-fetch';
import { describe, expect, it } from 'bun:test';

import type { paths } from '@/lib/api/generated/schema';
import { API_BASE_URL, apiErrorCode, toApiError, unwrap } from '@/lib/api/client';
import { ApiError } from '@/lib/api-error';

describe('API_BASE_URL', () => {
  // The regression net for a bug the fetch-layer tests could not see: the
  // contract's paths are rooted at `/` and the `/api` prefix lives in
  // `servers[].url`, which openapi-fetch does not read. A bare origin sends
  // GET('/accounts') to `/accounts`, which 404s and never reaches the proxy.
  it('ends in /api so contract paths resolve under it', () => {
    expect(API_BASE_URL.endsWith('/api')).toBe(true);
  });

  it('is relative in a browser-like environment, keeping calls same-origin', () => {
    expect(API_BASE_URL.startsWith('/')).toBe(true);
  });
});

describe('generated client URL composition', () => {
  // Exercises the composition itself rather than the constant, so a future
  // change to how baseUrl is assembled is caught here.
  it('resolves an operation path under /api', async () => {
    // openapi-fetch builds a Request, which needs an absolute URL outside a
    // browser. Anchoring the (relative) API_BASE_URL against a dummy origin
    // keeps the composition under test — base + '/accounts' — while giving the
    // runtime something it can parse.
    const absoluteBase = new URL(API_BASE_URL, 'http://localhost:3000').toString();

    let requested = '';
    const client = createClient<paths>({
      baseUrl: absoluteBase,
      fetch: (async (input: RequestInfo | URL) => {
        requested = String(input instanceof Request ? input.url : input);
        return new Response(JSON.stringify({ data: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }) as unknown as typeof globalThis.fetch,
    });

    await client.GET('/accounts');

    expect(requested).toContain('/api/accounts');
    expect(requested).not.toMatch(/\/api\/api\//);
  });
});

function failed(status: number, statusText: string): Response {
  return new Response(null, { status, statusText });
}

describe('toApiError', () => {
  it('prefers the API message over the generic one', () => {
    const error = toApiError(failed(409, 'Conflict'), 'accounts', {
      error: {
        code: 'DUPLICATE_ACCOUNT_NAME',
        message: 'You already have an account with this name.',
      },
    });

    expect(error).toBeInstanceOf(ApiError);
    expect(error.message).toBe('You already have an account with this name.');
  });

  it('falls back to the generic message when there is no envelope', () => {
    const error = toApiError(failed(500, 'Internal Server Error'), 'accounts', null);
    expect(error.message).toBe(
      'Failed to fetch accounts: 500 Internal Server Error'
    );
  });

  it('falls back when the envelope carries no message', () => {
    const error = toApiError(failed(400, 'Bad Request'), 'transactions', {
      error: { code: 'VALIDATION_ERROR' },
    });
    expect(error.message).toBe('Failed to fetch transactions: 400 Bad Request');
  });

  // The existing helpers are what call sites already branch on, so they have to
  // keep working against errors built the new way.
  it('preserves the status helpers downstream code uses', () => {
    const unauthorized = toApiError(failed(401, 'Unauthorized'), 'accounts', null);
    expect(unauthorized.isUnauthorized()).toBe(true);
    expect(unauthorized.isClientError()).toBe(true);

    const notFound = toApiError(failed(404, 'Not Found'), 'accounts', null);
    expect(notFound.isNotFound()).toBe(true);

    const serverError = toApiError(failed(503, 'Service Unavailable'), 'accounts', null);
    expect(serverError.isServerError()).toBe(true);
  });

  it('keeps the raw error body for logging', () => {
    const body = { error: { code: 'DB_ERROR', message: 'boom' } };
    expect(toApiError(failed(500, 'x'), 'accounts', body).details.errorData).toEqual(
      body
    );
  });
});

describe('apiErrorCode', () => {
  it('extracts the machine-readable code', () => {
    expect(apiErrorCode({ error: { code: 'ACCESS_TOKEN_EXPIRED' } })).toBe(
      'ACCESS_TOKEN_EXPIRED'
    );
  });

  it('returns null for anything without one', () => {
    expect(apiErrorCode(null)).toBeNull();
    expect(apiErrorCode(undefined)).toBeNull();
    expect(apiErrorCode({})).toBeNull();
    expect(apiErrorCode({ error: {} })).toBeNull();
    expect(apiErrorCode('not an envelope')).toBeNull();
  });
});

describe('unwrap', () => {
  it('returns the data on success', () => {
    const response = new Response(null, { status: 200 });
    expect(unwrap({ data: { id: 'a' }, response }, 'accounts')).toEqual({ id: 'a' });
  });

  it('throws an ApiError when the call failed', () => {
    const response = failed(404, 'Not Found');
    expect(() =>
      unwrap({ error: { error: { code: 'ACCOUNT_NOT_FOUND' } }, response }, 'accounts')
    ).toThrow(ApiError);
  });

  // A non-2xx with no parsed error body must still throw rather than returning
  // undefined as if it had succeeded.
  it('throws on a non-ok response even without an error body', () => {
    expect(() => unwrap({ response: failed(500, 'boom') }, 'accounts')).toThrow(
      ApiError
    );
  });
});
