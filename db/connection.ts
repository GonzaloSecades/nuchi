import { Pool, type PoolConfig } from 'pg';

const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1', '0.0.0.0']);

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

export function createPool(databaseUrl = getDatabaseUrl()) {
  const config: PoolConfig = {
    connectionString: databaseUrl,
  };

  if (shouldUseSsl(databaseUrl)) {
    config.ssl = {
      rejectUnauthorized: false,
    };
  }

  return new Pool(config);
}
