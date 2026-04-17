import { db } from '@/db/drizzle';
import {
  InsertTransactionSchema,
  accounts,
  categories,
  transactions,
} from '@/db/schema';
import {
  checkTransactionMutationRateLimit,
  isContentLengthTooLarge,
  MAX_BULK_CREATE_BODY_BYTES,
  MAX_BULK_CREATE_TRANSACTIONS,
  MAX_BULK_DELETE_BODY_BYTES,
  MAX_BULK_DELETE_TRANSACTIONS,
  parseDateRangeQuery,
} from '@/lib/transaction-route-utils';
import { clerkMiddleware, getAuth } from '@hono/clerk-auth';
import { zValidator } from '@hono/zod-validator';
import { createId } from '@paralleldrive/cuid2';
import { subDays } from 'date-fns';
import { and, desc, eq, gte, inArray, lte, sql } from 'drizzle-orm';
import { Hono, type Context } from 'hono';
import type { MiddlewareHandler } from 'hono';
import { z } from 'zod';

type OwnedReferencesResult =
  | {
      ok: true;
    }
  | {
      ok: false;
      status: 404;
      error: string;
    };

async function validateTransactionReferences(
  userId: string,
  values: {
    accountId: string;
    categoryId?: string | null;
  }
): Promise<OwnedReferencesResult> {
  const [account] = await db
    .select({
      id: accounts.id,
    })
    .from(accounts)
    .where(and(eq(accounts.id, values.accountId), eq(accounts.userId, userId)))
    .limit(1);

  if (!account) {
    return {
      ok: false,
      status: 404,
      error: 'Account not found',
    };
  }

  if (values.categoryId == null) {
    return { ok: true };
  }

  const [category] = await db
    .select({
      id: categories.id,
    })
    .from(categories)
    .where(
      and(eq(categories.id, values.categoryId), eq(categories.userId, userId))
    )
    .limit(1);

  if (!category) {
    return {
      ok: false,
      status: 404,
      error: 'Category not found',
    };
  }

  return { ok: true };
}

async function validateBulkTransactionReferences(
  userId: string,
  values: Array<{
    accountId: string;
    categoryId?: string | null;
  }>
): Promise<OwnedReferencesResult> {
  if (values.length === 0) {
    return { ok: true };
  }

  const uniqueAccountIds = [...new Set(values.map((value) => value.accountId))];
  const uniqueCategoryIds = [
    ...new Set(
      values
        .map((value) => value.categoryId)
        .filter((categoryId): categoryId is string => categoryId != null)
    ),
  ];

  const [ownedAccounts, ownedCategories] = await Promise.all([
    db
      .select({
        id: accounts.id,
      })
      .from(accounts)
      .where(
        and(eq(accounts.userId, userId), inArray(accounts.id, uniqueAccountIds))
      ),
    uniqueCategoryIds.length > 0
      ? db
          .select({
            id: categories.id,
          })
          .from(categories)
          .where(
            and(
              eq(categories.userId, userId),
              inArray(categories.id, uniqueCategoryIds)
            )
          )
      : Promise.resolve([]),
  ]);

  if (ownedAccounts.length !== uniqueAccountIds.length) {
    return {
      ok: false,
      status: 404,
      error: 'Account not found',
    };
  }

  if (ownedCategories.length !== uniqueCategoryIds.length) {
    return {
      ok: false,
      status: 404,
      error: 'Category not found',
    };
  }

  return { ok: true };
}

function sendRateLimited(
  c: Pick<Context, 'header' | 'json'>,
  retryAfterMs: number
) {
  c.header('Retry-After', String(Math.max(1, Math.ceil(retryAfterMs / 1000))));

  return c.json(
    {
      error: {
        code: 'TRANSACTION_MUTATION_RATE_LIMITED',
        message: 'Too many transaction mutations. Please try again later.',
      },
    },
    429
  );
}

function enforceJsonBodyLimit(maxBytes: number): MiddlewareHandler {
  return async (c, next) => {
    if (isContentLengthTooLarge(c.req.header('content-length'), maxBytes)) {
      return c.json(
        {
          error: {
            code: 'REQUEST_BODY_TOO_LARGE',
            message: 'Request body is too large.',
          },
        },
        413
      );
    }

    await next();
  };
}

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

      const dateRange = parseDateRangeQuery({
        from,
        to,
        defaultFrom,
        defaultTo,
      });

      if (!dateRange.ok) {
        return c.json({ error: dateRange.error }, dateRange.status);
      }

      const { startDate, endDate } = dateRange;

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

      const rateLimit = checkTransactionMutationRateLimit(
        auth.userId,
        'create'
      );

      if (!rateLimit.allowed) {
        return sendRateLimited(c, rateLimit.retryAfterMs);
      }

      try {
        const ownership = await validateTransactionReferences(
          auth.userId,
          values
        );

        if (!ownership.ok) {
          return c.json({ error: ownership.error }, ownership.status);
        }

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
    enforceJsonBodyLimit(MAX_BULK_CREATE_BODY_BYTES),
    zValidator(
      'json',
      z
        .array(InsertTransactionSchema.omit({ id: true }))
        .min(1, 'At least one transaction is required')
        .max(
          MAX_BULK_CREATE_TRANSACTIONS,
          `At most ${MAX_BULK_CREATE_TRANSACTIONS} transactions can be created at once`
        )
    ),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      const rateLimit = checkTransactionMutationRateLimit(
        auth.userId,
        'bulk-create'
      );

      if (!rateLimit.allowed) {
        return sendRateLimited(c, rateLimit.retryAfterMs);
      }

      try {
        const ownership = await validateBulkTransactionReferences(
          auth.userId,
          values
        );

        if (!ownership.ok) {
          return c.json({ error: ownership.error }, ownership.status);
        }

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
    enforceJsonBodyLimit(MAX_BULK_DELETE_BODY_BYTES),
    zValidator(
      'json',
      z.object({
        ids: z
          .array(z.string().min(1))
          .min(1, 'At least one id is required')
          .max(
            MAX_BULK_DELETE_TRANSACTIONS,
            `At most ${MAX_BULK_DELETE_TRANSACTIONS} transactions can be deleted at once`
          ),
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      const rateLimit = checkTransactionMutationRateLimit(
        auth.userId,
        'bulk-delete'
      );

      if (!rateLimit.allowed) {
        return sendRateLimited(c, rateLimit.retryAfterMs);
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

      const rateLimit = checkTransactionMutationRateLimit(auth.userId, 'patch');

      if (!rateLimit.allowed) {
        return sendRateLimited(c, rateLimit.retryAfterMs);
      }

      try {
        const ownership = await validateTransactionReferences(
          auth.userId,
          values
        );

        if (!ownership.ok) {
          return c.json({ error: ownership.error }, ownership.status);
        }

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
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to update transaction',
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
        return c.json({ error: 'Missing transaction id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      const rateLimit = checkTransactionMutationRateLimit(
        auth.userId,
        'delete'
      );

      if (!rateLimit.allowed) {
        return sendRateLimited(c, rateLimit.retryAfterMs);
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
