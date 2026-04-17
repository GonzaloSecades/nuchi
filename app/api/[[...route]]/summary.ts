import { clerkMiddleware, getAuth } from '@hono/clerk-auth';
import { zValidator } from '@hono/zod-validator';
import { differenceInCalendarDays, subDays } from 'date-fns';
import { Hono } from 'hono';
import { z } from 'zod';

import { db } from '@/db/drizzle';
import { accounts, categories, transactions } from '@/db/schema';
import { calculatePercentageChange, fillMissingDays } from '@/lib/utils';
import { parseStrictDate } from '@/lib/transaction-route-utils';
import { and, desc, eq, gte, lt, lte, sql, sum } from 'drizzle-orm';

const app = new Hono().get(
  '/',
  clerkMiddleware(),
  zValidator(
    'query',
    z.object({
      from: z.string().optional(),
      to: z.string().optional(),
      accountId: z.string().optional(),
    })
  ),
  async (c) => {
    const auth = getAuth(c);
    const { from, to, accountId } = c.req.valid('query');
    if (!auth?.userId) {
      return c.json({ error: 'Unauthorized' }, 401);
    }

    const defaultTo = new Date();
    const defaultFrom = subDays(defaultTo, 30);
    const parsedFrom = parseStrictDate(from);
    const parsedTo = parseStrictDate(to, { boundary: 'end' });
    const startDate = parsedFrom ?? defaultFrom;
    const endDate = parsedTo ?? defaultTo;

    if ((from && !parsedFrom) || (to && !parsedTo)) {
      return c.json(
        {
          error: {
            code: 'INVALID_QUERY',
            message: 'from and to must use yyyy-MM-dd dates.',
          },
        },
        400
      );
    }

    if (startDate > endDate) {
      return c.json(
        {
          error: {
            code: 'INVALID_QUERY',
            message: 'from must be less than or equal to to.',
          },
        },
        400
      );
    }

    if (differenceInCalendarDays(endDate, startDate) + 1 > 366) {
      return c.json(
        {
          error: {
            code: 'INVALID_QUERY',
            message: 'Date range cannot exceed 366 days.',
          },
        },
        400
      );
    }

    const periodLength = differenceInCalendarDays(endDate, startDate) + 1;
    const lastPeriodStart = subDays(startDate, periodLength);
    const lastPeriodEnd = subDays(endDate, periodLength);

    try {
      async function fetchFinancialData(
        userId: string,
        startDate: Date,
        endDate: Date
      ) {
        return await db
          .select({
            income:
              sql`SUM(CASE WHEN ${transactions.amount} >= 0 THEN ${transactions.amount} ELSE 0 END)`.mapWith(
                Number
              ),
            expenses:
              sql`SUM(CASE WHEN ${transactions.amount} < 0 THEN ABS(${transactions.amount}) ELSE 0 END)`.mapWith(
                Number
              ),
            remaining: sum(transactions.amount).mapWith(Number),
          })
          .from(transactions)
          .innerJoin(accounts, eq(transactions.accountId, accounts.id))
          .where(
            and(
              accountId ? eq(transactions.accountId, accountId) : undefined,
              eq(accounts.userId, userId),
              gte(transactions.date, startDate),
              lte(transactions.date, endDate)
            )
          );
      }

      const [currentPeriod] = await fetchFinancialData(
        auth.userId,
        startDate,
        endDate
      );
      const [lastPeriod] = await fetchFinancialData(
        auth.userId,
        lastPeriodStart,
        lastPeriodEnd
      );

      const incomeChange = calculatePercentageChange(
        currentPeriod.income || 0,
        lastPeriod.income || 0
      );

      const expensesChange = calculatePercentageChange(
        currentPeriod.expenses || 0,
        lastPeriod.expenses || 0
      );

      const remainingChange = calculatePercentageChange(
        currentPeriod.remaining || 0,
        lastPeriod.remaining || 0
      );

      const category = await db
        .select({
          name: categories.name,
          value: sql`SUM(ABS(${transactions.amount}))`.mapWith(Number),
        })
        .from(transactions)
        .innerJoin(accounts, eq(transactions.accountId, accounts.id))
        .innerJoin(categories, eq(transactions.categoryId, categories.id))
        .where(
          and(
            accountId ? eq(transactions.accountId, accountId) : undefined,
            eq(accounts.userId, auth.userId),
            lt(transactions.amount, 0),
            gte(transactions.date, startDate),
            lte(transactions.date, endDate)
          )
        )
        .groupBy(categories.name)
        .orderBy(desc(sql`SUM(ABS(${transactions.amount}))`));

      const topCategories = category.slice(0, 3);
      const otherCategories = category.slice(3);
      const otherSum = otherCategories.reduce(
        (sum, current) => sum + current.value,
        0
      );

      const finalCategories = topCategories;
      if (otherCategories.length > 0) {
        finalCategories.push({
          name: 'Other',
          value: otherSum,
        });
      }

      const activeDays = await db
        .select({
          date: transactions.date,
          income:
            sql`SUM(CASE WHEN ${transactions.amount} >= 0 THEN ${transactions.amount} ELSE 0 END)`.mapWith(
              Number
            ),
          expenses:
            sql`SUM(CASE WHEN ${transactions.amount} < 0 THEN ABS(${transactions.amount}) ELSE 0 END)`.mapWith(
              Number
            ),
        })
        .from(transactions)
        .innerJoin(accounts, eq(transactions.accountId, accounts.id))
        .where(
          and(
            accountId ? eq(transactions.accountId, accountId) : undefined,
            eq(accounts.userId, auth.userId),
            gte(transactions.date, startDate),
            lte(transactions.date, endDate)
          )
        )
        .groupBy(transactions.date)
        .orderBy(transactions.date);

      const days = fillMissingDays(activeDays, startDate, endDate);

      return c.json({
        data: {
          remainingAmount: currentPeriod.remaining,
          remainingChange,
          incomeAmount: currentPeriod.income,
          incomeChange,
          expensesAmount: currentPeriod.expenses,
          expensesChange,
          categories: finalCategories,
          days,
        },
      });
    } catch {
      return c.json(
        {
          error: {
            code: 'DB_ERROR',
            message: 'DatabaseError - Failed to fetch summary',
          },
        },
        500
      );
    }
  }
);

export default app;
