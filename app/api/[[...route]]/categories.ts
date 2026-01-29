import { db } from '@/db/drizzle';
import { categories, InsertCategorySchema } from '@/db/schema';
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
          id: categories.id,
          name: categories.name,
        })
        .from(categories)
        .where(eq(categories.userId, auth.userId));
      return c.json({ data });
    } catch {
      return c.json(
        {
          error: {
            code: 'DB_ERROR',
            message: 'DatabaseError - Failed to fetch categories',
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
            id: categories.id,
            name: categories.name,
          })
          .from(categories)
          .where(and(eq(categories.userId, auth.userId), eq(categories.id, id)))
          .limit(1);
        if (!data) {
          return c.json({ error: 'Category not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to fetch category',
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
    zValidator('json', InsertCategorySchema.pick({ name: true })),
    async (c) => {
      const auth = getAuth(c);
      const values = c.req.valid('json');
      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }
      try {
        const [data] = await db
          .insert(categories)
          .values({
            id: createId(),
            userId: auth.userId,
            ...values,
          })
          .returning();

        return c.json({ data });
      } catch (error) {
        // Postgres unique-constraint violation (e.g. categories_user_id_name_uniq)
        const err = error as { code?: string; constraint?: string };
        if (err?.code === '23505') {
          return c.json(
            {
              error: {
                code: 'DUPLICATE_CATEGORY_NAME',
                message: 'You already have a category with this name.',
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
              message: 'DatabaseError - Failed to create category',
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
          .delete(categories)
          .where(
            and(
              eq(categories.userId, auth.userId),
              inArray(categories.id, values.ids)
            )
          )
          .returning({
            id: categories.id,
          });
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete categories',
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
      InsertCategorySchema.pick({
        name: true,
      })
    ),
    async (c) => {
      const auth = getAuth(c);
      const { id } = c.req.valid('param');
      const values = c.req.valid('json');

      if (!id) {
        return c.json({ error: 'Missing category id' }, 400);
      }

      if (!auth?.userId) {
        return c.json({ error: 'Unauthorized' }, 401);
      }

      try {
        const [data] = await db
          .update(categories)
          .set(values)
          .where(and(eq(categories.id, id), eq(categories.userId, auth.userId)))
          .returning();

        if (!data) {
          return c.json({ error: 'Category not found' }, 404);
        }
        return c.json({ data });
      } catch {
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
        const [data] = await db
          .delete(categories)
          .where(and(eq(categories.id, id), eq(categories.userId, auth.userId)))
          .returning({ id: categories.id });

        if (!data) {
          return c.json({ error: 'Category not found' }, 404);
        }
        return c.json({ data });
      } catch {
        return c.json(
          {
            error: {
              code: 'DB_ERROR',
              message: 'DatabaseError - Failed to delete category',
            },
          },
          500
        );
      }
    }
  );

export default app;
