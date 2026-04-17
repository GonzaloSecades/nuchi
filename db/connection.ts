import { Pool, type PoolConfig } from 'pg';

const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1', '0.0.0.0']);
const INSECURE_TLS_OPT_OUT_ENV = 'ALLOW_INSECURE_DATABASE_TLS';

type DatabasePoolCache = {
  databaseUrl?: string;
  pool?: Pool;
};

declare global {
  // Keep a single app pool across Next.js dev HMR module re-evaluations.
  var __nuchiDatabasePool: DatabasePoolCache | undefined;
}

export function getDatabaseUrl() {
  const databaseUrl = process.env.DATABASE_URL;

  if (!databaseUrl) {
    throw new Error('DATABASE_URL is required');
  }

  return databaseUrl;
}

export function isLocalDatabaseUrl(databaseUrl: string) {
  try {
    const url = new URL(databaseUrl);
    return LOCAL_HOSTS.has(url.hostname) || url.hostname.endsWith('.local');
  } catch {
    return false;
  }
}

function shouldUseSsl(databaseUrl: string) {
  try {
    const url = new URL(databaseUrl);
    const sslMode = url.searchParams.get('sslmode');

    if (sslMode === 'disable') {
      return false;
    }

    if (isLocalDatabaseUrl(databaseUrl)) {
      return false;
    }

    return sslMode === 'require' || url.hostname.includes('neon.tech');
  } catch {
    return false;
  }
}

function shouldAllowInsecureTls(env: Partial<NodeJS.ProcessEnv>) {
  return env[INSECURE_TLS_OPT_OUT_ENV] === 'true';
}

export function buildPoolConfig(
  databaseUrl = getDatabaseUrl(),
  env: Partial<NodeJS.ProcessEnv> = process.env
) {
  const config: PoolConfig = {
    connectionString: databaseUrl,
  };

  if (shouldUseSsl(databaseUrl)) {
    config.ssl = {
      rejectUnauthorized: !shouldAllowInsecureTls(env),
    };
  }

  return config;
}

export function createPool(databaseUrl = getDatabaseUrl()) {
  const config = buildPoolConfig(databaseUrl);

  return new Pool(config);
}

export function getOrCreateCachedPool({
  databaseUrl = getDatabaseUrl(),
  env = process.env,
  cache = (globalThis.__nuchiDatabasePool ??= {}),
  poolFactory = createPool,
}: {
  databaseUrl?: string;
  env?: Partial<NodeJS.ProcessEnv>;
  cache?: DatabasePoolCache;
  poolFactory?: (databaseUrl: string) => Pool;
} = {}) {
  if (env.NODE_ENV === 'production') {
    return poolFactory(databaseUrl);
  }

  if (cache.pool && cache.databaseUrl === databaseUrl) {
    return cache.pool;
  }

  if (cache.pool) {
    void cache.pool.end().catch(() => undefined);
  }

  const pool = poolFactory(databaseUrl);
  cache.databaseUrl = databaseUrl;
  cache.pool = pool;

  return pool;
}
