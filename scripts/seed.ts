import { neon } from '@neondatabase/serverless';
import { subDays } from 'date-fns';
import { config } from 'dotenv';
import { drizzle } from 'drizzle-orm/neon-http';
import { accounts, categories, transactions } from '../db/schema';

config({ path: '.env.local' });

const sql = neon(process.env.DATABASE_URL!);
const db = drizzle(sql);

const SEED_USER_ID = 'user_37tiuBEyYiNDFRk2anSrwOM5Oql';
const SEED_CATEGORIES = [
  { id: 'category_1', name: 'Food', userId: SEED_USER_ID, plaidId: null },
  { id: 'category_2', name: 'Transport', userId: SEED_USER_ID, plaidId: null },
  {
    id: 'category_3',
    name: 'Entertainment',
    userId: SEED_USER_ID,
    plaidId: null,
  },
  { id: 'category_7', name: 'Utilities', userId: SEED_USER_ID, plaidId: null },
];

const SEED_ACCOUNTS = [
  {
    id: 'account_1',
    name: 'Checking Account',
    userId: SEED_USER_ID,
    plaidId: null,
  },
  {
    id: 'account_2',
    name: 'Savings Account',
    userId: SEED_USER_ID,
    plaidId: null,
  },
];

const defaultTo = new Date();
const defaultFrom = subDays(defaultTo, 30);

const SEED_TRANSACTIONS: (typeof transactions.$inferSelect)[] = [];

import { convertAmountToMiliunits } from '@/lib/utils';
import { eachDayOfInterval, format } from 'date-fns';

const generateRandomAmount = (category: typeof categories.$inferInsert) => {
  switch (category.name) {
    case 'Food':
      return Math.random() * 30 + 10;
    case 'Transport':
      return Math.random() * 20 + 5;
    case 'Entertainment':
      return Math.random() * 50 + 20;
    case 'Utilities':
      return Math.random() * 100 + 50;
    default:
      return Math.random() * 20 + 10;
  }
};

const generateTransactionsForDay = (day: Date) => {
  const numTransactions = Math.floor(Math.random() * 4) + 1; // 1 to 3 transactions per day

  for (let i = 0; i < numTransactions; i++) {
    const category =
      SEED_CATEGORIES[Math.floor(Math.random() * SEED_CATEGORIES.length)];
    const isExpense = Math.random() > 0.6; // 60% chance of being an expense
    const amount = generateRandomAmount(category);
    const formattedAmount = convertAmountToMiliunits(
      isExpense ? -amount : amount
    );

    SEED_TRANSACTIONS.push({
      id: `transaction_${format(day, 'yyyy-MM-dd')}_${i}`,
      accountId: SEED_ACCOUNTS[0].id,
      categoryId: category.id,
      amount: formattedAmount,
      date: day,
      payee: 'Mercant',
      notes: 'Sample transaction',
    });
  }
};

const generateTransactions = () => {
  const days = eachDayOfInterval({ start: defaultFrom, end: defaultTo });
  days.forEach((day) => generateTransactionsForDay(day));
};
generateTransactions();

const main = async () => {
  try {
    await db.delete(transactions).execute();
    await db.delete(accounts).execute();
    await db.delete(categories).execute();

    await db.insert(categories).values(SEED_CATEGORIES).execute();
    await db.insert(accounts).values(SEED_ACCOUNTS).execute();
    await db.insert(transactions).values(SEED_TRANSACTIONS).execute();
  } catch (error) {
    console.error('Error seeding database:', error);
    process.exit(1);
  }
};

main();
