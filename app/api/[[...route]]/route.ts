import { Hono } from 'hono';
import { handle } from 'hono/vercel';

import accounts from './accounts';
//routes

export const runtime = 'edge';

const app = new Hono().basePath('/api');

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const routes = app.route('/accounts', accounts);

export const GET = handle(app);
export const POST = handle(app);
export const PATCH = handle(app);
export const DELETE = handle(app);
export const PUT = handle(app);
export const OPTIONS = handle(app);

export default app;
export type AppType = typeof routes;

/**
 We keep routes only to extract the AppType. ESLint sees it as unused at runtime, so the disable comment avoids a false positive.
 */
