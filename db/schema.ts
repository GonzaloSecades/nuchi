import { relations } from 'drizzle-orm';
import {
  customType,
  index,
  integer,
  pgTable,
  text,
  timestamp,
  uniqueIndex,
} from 'drizzle-orm/pg-core';
import { createInsertSchema } from 'drizzle-zod';
import { z } from 'zod';

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

export const accountsRelations = relations(accounts, ({ many }) => ({
  transactions: many(transactions),
}));

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

export const categoriesRelations = relations(categories, ({ many }) => ({
  transactions: many(transactions),
}));

/**
 * Defines the schema for the transactions table.
 * The `amount` column uses a PostgreSQL integer to represent the smallest unit of currency (e.g., cents),
 * and values are stored in miliunits to avoid floating point precision issues.
 * Example: $10.50 => 10500.
 * The `date` column is the date of the transaction; users can add dates in the past or future, and it is not pre-filled.
 */

export const transactions = pgTable(
  'transactions',
  {
    id: text('id').primaryKey(),
    amount: integer('amount').notNull(),
    payee: text('payee').notNull(),
    notes: text('notes'),
    date: timestamp('date', { mode: 'date' }).notNull(),
    accountId: text('account_id')
      .references(() => accounts.id, {
        onDelete: 'cascade',
      })
      .notNull(),
    categoryId: text('category_id').references(() => categories.id, {
      onDelete: 'set null',
    }),
  },
  (table) => ({
    accountIdIdx: index('transactions_account_id_idx').on(table.accountId),
    categoryIdIdx: index('transactions_category_id_idx').on(table.categoryId),
  })
);

export const transactionsRelations = relations(transactions, ({ one }) => ({
  account: one(accounts, {
    fields: [transactions.accountId],
    references: [accounts.id],
  }),
  categories: one(categories, {
    fields: [transactions.categoryId],
    references: [categories.id],
  }),
}));

export const InsertAccountSchema = createInsertSchema(accounts);
export const InsertCategorySchema = createInsertSchema(categories);
export const InsertTransactionSchema = createInsertSchema(transactions, {
  date: z.coerce.date(),
});
