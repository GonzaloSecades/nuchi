import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import type { Pool } from 'pg';

import { buildPoolConfig, getOrCreateCachedPool } from './connection';

describe('buildPoolConfig', () => {
  it('verifies TLS certificates by default for remote SSL databases', () => {
    const config = buildPoolConfig(
      'postgres://user:pass@example.neon.tech/db?sslmode=require'
    );

    assert.deepEqual(config.ssl, {
      rejectUnauthorized: true,
    });
  });

  it('only disables TLS certificate verification with an explicit opt-out', () => {
    const config = buildPoolConfig(
      'postgres://user:pass@example.neon.tech/db?sslmode=require',
      {
        ALLOW_INSECURE_DATABASE_TLS: 'true',
      }
    );

    assert.deepEqual(config.ssl, {
      rejectUnauthorized: false,
    });
  });

  it('does not enable TLS for local databases', () => {
    const config = buildPoolConfig(
      'postgres://nuchi:nuchi_dev_password@127.0.0.1:54329/nuchi_dev'
    );

    assert.equal(config.ssl, undefined);
  });
});

describe('getOrCreateCachedPool', () => {
  it('reuses one pool for repeated development module evaluations', () => {
    const cache = {};
    const createdPools: Pool[] = [];

    const poolFactory = () => {
      const pool = { end: async () => undefined } as Pool;
      createdPools.push(pool);
      return pool;
    };

    const first = getOrCreateCachedPool({
      databaseUrl: 'postgres://user:pass@127.0.0.1:54329/nuchi_dev',
      env: { NODE_ENV: 'development' },
      cache,
      poolFactory,
    });
    const second = getOrCreateCachedPool({
      databaseUrl: 'postgres://user:pass@127.0.0.1:54329/nuchi_dev',
      env: { NODE_ENV: 'development' },
      cache,
      poolFactory,
    });

    assert.equal(first, second);
    assert.equal(createdPools.length, 1);
  });

  it('replaces and closes a cached pool when the database URL changes', async () => {
    const cache = {};
    let closedPools = 0;

    const poolFactory = () =>
      ({
        end: async () => {
          closedPools += 1;
        },
      }) as Pool;

    const first = getOrCreateCachedPool({
      databaseUrl: 'postgres://user:pass@127.0.0.1:54329/nuchi_dev',
      env: { NODE_ENV: 'development' },
      cache,
      poolFactory,
    });
    const second = getOrCreateCachedPool({
      databaseUrl: 'postgres://user:pass@127.0.0.1:54330/nuchi_dev',
      env: { NODE_ENV: 'development' },
      cache,
      poolFactory,
    });

    await Promise.resolve();

    assert.notEqual(first, second);
    assert.equal(closedPools, 1);
  });
});
