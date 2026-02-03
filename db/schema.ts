import {
  customType,
  index,
  pgTable,
  text,
  uniqueIndex,
} from 'drizzle-orm/pg-core';
import { createInsertSchema } from 'drizzle-zod';

// Postgres CITEXT gives case-insensitive comparisons (and therefore uniqueness).
// Drizzle pg-core doesn't export citext in all versions, so we map it via customType.
const citext = customType<{ data: string }>({
  dataType() {
    return 'citext';
  },
});

export const accounts = pgTable(
  'accounts',
  {
    id: text('id').primaryKey(),
    plaidId: text('plaid_id'), // bank data aggregation api.
    name: citext('name').notNull(),
    userId: text('user_id').notNull(),
  },
  (table) => ({
    userIdIdx: index('accounts_user_id_idx').on(table.userId),
    userIdNameUniq: uniqueIndex('accounts_user_id_name_uniq').on(
      table.userId,
      table.name
    ),
  })
);

export const categories = pgTable(
  'categories',
  {
    id: text('id').primaryKey(),
    plaidId: text('plaid_id'), // bank data aggregation api.
    name: citext('name').notNull(),
    userId: text('user_id').notNull(),
  },
  (table) => ({
    userIdIdx: index('categories_user_id_idx').on(table.userId),
    userIdNameUniq: uniqueIndex('categories_user_id_name_uniq').on(
      table.userId,
      table.name
    ),
  })
);

//for transactions we will index a child table like accounts using accountID

export const InsertAccountSchema = createInsertSchema(accounts);
export const InsertCategorySchema = createInsertSchema(categories);
