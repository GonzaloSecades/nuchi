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
      } catch {
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
        ids: z.array(z.string()),
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
  );

export default app;
