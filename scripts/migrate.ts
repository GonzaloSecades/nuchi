import { config } from 'dotenv';
import { drizzle } from 'drizzle-orm/node-postgres';
import { migrate } from 'drizzle-orm/node-postgres/migrator';

import { createPool } from '../db/connection';

config({
  path: '.env.local',
});

const pool = createPool();
const db = drizzle(pool);

const main = async () => {
  try {
    await migrate(db, { migrationsFolder: './drizzle' });
  } catch (error) {
    console.error('Migration failed:', error);
    process.exit(1);
  } finally {
    await pool.end();
  }
};

main();
