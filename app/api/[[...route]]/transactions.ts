import { db } from '@/db/drizzle';
import {
  InsertTransactionSchema,
  accounts,
  categories,
  transactions,
} from '@/db/schema';
import { clerkMiddleware, getAuth } from '@hono/clerk-auth';
import { zValidator } from '@hono/zod-validator';
import { createId } from '@paralleldrive/cuid2';
import { parse, subDays } from 'date-fns';
import { and, desc, eq, gte, inArray, lte, sql } from 'drizzle-orm';
import { Hono } from 'hono';
import { z } from 'zod';

const app = new Hono()
  .get(
    '/',
    zValidator(
      'query',
      z.object({
        from: z.string().optional(),
        to: z.string().optional(),
        accountId: z.string().optional(),
      })
    ),
    clerkMiddleware(),
    async (c) => {
      const auth = getAuth(c);
      const { from, to, accountId } = c.req.valid('query');

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      const defaultTo = new Date();
      const defaultFrom = subDays(defaultTo, 30);

      const startDate = from
        ? parse(from, 'yyyy-MM-dd', new Date())
        : defaultFrom;

      const endDate = to ? parse(to, 'yyyy-MM-dd', new Date()) : defaultTo;

      try {
        const data = await db
          .select({
            id: transactions.id,
            date: transactions.date,
            category: categories.name,
            categoryId: transactions.categoryId,
            payee: transactions.payee,
            amount: transactions.amount,
            notes: transactions.notes,
            account: accounts.name,
            accountId: transactions.accountId,
          })
          .from(transactions)
          .innerJoin(accounts, eq(transactions.accountId, accounts.id))
          .leftJoin(categories, eq(transactions.categoryId, categories.id))
          .where(
            and(
              accountId ? eq(transactions.accountId, accountId) : undefined,
              eq(accounts.userId, auth.userId), //transactions dont belong to users belongs to accounts.
              gte(transactions.date, startDate),
              lte(transactions.date, endDate)
            )
          )
          .orderBy(desc(transactions.date));

        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to fetch transaction',
            },
          },
          500
        );
      }
    }
  )
  .get(
    '/:id',
    clerkMiddleware(),
    zValidator('param', z.object({ id: z.string().optional() })),
    async (c) => {
      const auth = getAuth(c);
      const { id } = c.req.valid('param');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }
      if (!id) {
        return c.json({ error: 'Missing id' }, 400);
      }

      try {
        const [data] = await db
          .select({
            id: transactions.id,
            date: transactions.date,
            categoryId: transactions.categoryId,
            payee: transactions.payee,
            amount: transactions.amount,
            notes: transactions.notes,
            accountId: transactions.accountId,
          })
          .from(transactions)
          .innerJoin(accounts, eq(transactions.accountId, accounts.id))
          .where(and(eq(transactions.id, id), eq(accounts.userId, auth.userId)))
          .limit(1);
        if (!data) {
          return c.json({ error: 'Transaction not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to fetch transaction',
            },
          },
          500
        );
      }
    }
  )
  .post(
    '/',
    clerkMiddleware(),
    zValidator('json', InsertTransactionSchema.omit({ id: true })),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }
      try {
        const [data] = await db
          .insert(transactions)
          .values({
            id: createId(),
            ...values,
          })
          .returning();

        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to create transaction',
            },
          },
          500
        );
      }
    }
  )
  .post(
    '/bulk-create',
    clerkMiddleware(),
    zValidator('json', z.array(InsertTransactionSchema.omit({ id: true }))),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const data = await db
          .insert(transactions)
          .values(
            values.map((value) => ({
              id: createId(),
              ...value,
            }))
          )
          .returning();

        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to create transactions',
            },
          },
          500
        );
      }
    }
  )
  .post(
    //orm.drizzle.team/docs/delete - WITH DELETE clause
    '/bulk-delete',
    clerkMiddleware(),
    zValidator(
      'json',
      z.object({
        ids: z.array(z.string().min(1)).min(1, 'At least one id is required'),
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      const transactionsToDelete = db.$with('transactions_to_delete').as(
        db
          .select({
            id: transactions.id,
          })
          .from(transactions)
          .innerJoin(accounts, eq(transactions.accountId, accounts.id))
          .where(
            and(
              inArray(transactions.id, values.ids),
              eq(accounts.userId, auth.userId)
            )
          )
      );

      try {
        const data = await db
          .with(transactionsToDelete)
          .delete(transactions)
          .where(
            inArray(
              transactions.id,
              sql`(select id from ${transactionsToDelete})`
            )
          )
          .returning({
            id: transactions.id,
          });

        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete transactions',
            },
          },
          500
        );
      }
    }
  )
  .patch(
    '/:id',
    clerkMiddleware(),
    zValidator(
      'param',
      z.object({
        id: z.string().optional(),
      })
    ),
    zValidator(
      'json',
      InsertTransactionSchema.omit({
        id: true,
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const { id } = c.req.valid('param');
      const values = c.req.valid('json');

      if (!id) {
        return c.json({ error: 'Missing Transaction id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const transactionsToUpdate = db.$with('transactions_to_edit').as(
          db
            .select({
              id: transactions.id,
            })
            .from(transactions)
            .innerJoin(accounts, eq(transactions.accountId, accounts.id))
            .where(
              and(eq(transactions.id, id), eq(accounts.userId, auth.userId))
            )
        );
        const [data] = await db
          .with(transactionsToUpdate)
          .update(transactions)
          .set(values)
          .where(
            inArray(
              transactions.id,
              sql`(select id from ${transactionsToUpdate})`
            )
          )
          .returning();

        if (!data) {
          return c.json({ error: 'Transaction not found' }, 404);
        }

        return c.json({ data });
      } catch (e) {
        console.log(e);
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to update category',
            },
          },
          500
        );
      }
    }
  )
  .delete(
    '/:id',
    clerkMiddleware(),
    zValidator(
      'param',
      z.object({
        id: z.string().optional(),
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const { id } = c.req.valid('param');

      if (!id) {
        return c.json({ error: 'Missing category id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const transactionToDelete = db.$with('transactions_to_delete').as(
          db
            .select({
              id: transactions.id,
            })
            .from(transactions)
            .innerJoin(accounts, eq(transactions.accountId, accounts.id))
            .where(
              and(eq(transactions.id, id), eq(accounts.userId, auth.userId))
            )
        );
        const [data] = await db
          .with(transactionToDelete)
          .delete(transactions)
          .where(
            inArray(
              transactions.id,
              sql`(select id from ${transactionToDelete})`
            )
          )
          .returning({
            id: transactions.id,
          });

        if (!data) {
          return c.json({ error: 'Transaction not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete transaction',
            },
          },
          500
        );
      }
    }
  );

export default app;
