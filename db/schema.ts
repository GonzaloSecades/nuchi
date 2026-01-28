import { pgTable, text, index } from 'drizzle-orm/pg-core';
import { createInsertSchema } from 'drizzle-zod';

export const accounts = pgTable(
  'accounts',
  {
    id: text('id').primaryKey(),
    plaidId: text('plaid_id'), // bank data aggregation api.
    name: text('name').notNull(),
    userId: text('user_id').notNull(),
  },
  (table) => ({
    userIdIdx: index('accounts_user_id_idx').on(table.userId),
  })
);

//for transactions we will index a child table like accounts using accountID

export const InsertAccountSchema = createInsertSchema(accounts);
