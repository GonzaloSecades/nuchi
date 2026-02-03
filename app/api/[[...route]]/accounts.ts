import { db } from '@/db/drizzle';
import { accounts, InsertAccountSchema } from '@/db/schema';
import { clerkMiddleware, getAuth } from '@hono/clerk-auth';
import { zValidator } from '@hono/zod-validator';
import { createId } from '@paralleldrive/cuid2';
import { and, eq, inArray } from 'drizzle-orm';
import { Hono } from 'hono';
import z from 'zod';

const app = new Hono()
  .get('/', clerkMiddleware(), async (c) => {
    const auth = getAuth(c);

    if (!auth?.userId) {
      return c.json({ error: 'Unauthorized' }, 401);
    }

    try {
      const data = await db
        .select({
          id: accounts.id,
          name: accounts.name,
        })
        .from(accounts)
        .where(eq(accounts.userId, auth.userId));
      return c.json({ data });
    } catch {
      return c.json(
        {
          error: {
            code: 'DB_ERROR',
            message: 'DatabaseError - Failed to fetch accounts',
          },
        },
        500
      );
    }
  })
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
            id: accounts.id,
            name: accounts.name,
          })
          .from(accounts)
          .where(and(eq(accounts.userId, auth.userId), eq(accounts.id, id)))
          .limit(1);
        if (!data) {
          return c.json({ error: 'Account not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to fetch account',
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
    zValidator('json', InsertAccountSchema.pick({ name: true })),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }
      try {
        const [data] = await db
          .insert(accounts)
          .values({
            id: createId(),
            userId: auth.userId,
            ...values,
          })
          .returning();

        return c.json({ data });
      } catch (error) {
        const err = error as { code?: string; constraint?: string };
        if (err?.code === '23505') {
          return c.json(
            {
              error: {
                code: 'DUPLICATE_ACCOUNT_NAME',
                message: 'You already have an account with this name.',
                constraint: err.constraint,
              },
            },
            409
          );
        }
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to create account',
            },
          },
          500
        );
      }
    }
  )
  .post(
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
      try {
        const data = await db
          .delete(accounts)
          .where(
            and(
              eq(accounts.userId, auth.userId),
              inArray(accounts.id, values.ids)
            )
          )
          .returning({
            id: accounts.id,
          });
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete accounts',
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
      InsertAccountSchema.pick({
        name: true,
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const { id } = c.req.valid('param');
      const values = c.req.valid('json');

      if (!id) {
        return c.json({ error: 'Missing account id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const [data] = await db
          .update(accounts)
          .set(values)
          .where(and(eq(accounts.id, id), eq(accounts.userId, auth.userId)))
          .returning();

        if (!data) {
          return c.json({ error: 'Account not found' }, 404);
        }
        return c.json({ data });
      } catch (error) {
        const err = error as { code?: string; constraint?: string };
        if (err?.code === '23505') {
          return c.json(
            {
              error: {
                code: 'DUPLICATE_ACCOUNT_NAME',
                message: 'You already have an account with this name.',
                constraint: err.constraint,
              },
            },
            409
          );
        }
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to update account',
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
        return c.json({ error: 'Missing account id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const [data] = await db
          .delete(accounts)
          .where(and(eq(accounts.id, id), eq(accounts.userId, auth.userId)))
          .returning({ id: accounts.id });

        if (!data) {
          return c.json({ error: 'Account not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete account',
            },
          },
          500
        );
      }
    }
  );

export default app;
