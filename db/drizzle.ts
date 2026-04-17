import { drizzle } from 'drizzle-orm/node-postgres';

import { getOrCreateCachedPool } from '@/db/connection';

export const pool = getOrCreateCachedPool();
export const db = drizzle(pool);
