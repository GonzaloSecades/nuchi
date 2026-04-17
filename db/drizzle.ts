import { drizzle } from 'drizzle-orm/node-postgres';

import { createPool } from '@/db/connection';

export const pool = createPool();
export const db = drizzle(pool);
