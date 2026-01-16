/**
 * Hono RPC Client
 *
 * This file sets up a type-safe API client using Hono's RPC feature.
 *
 * What is this?
 * -------------
 * While often compared to tRPC, this is actually Hono's built-in RPC client (`hc`).
 * It provides end-to-end type safety between your API routes and client code,
 * similar to what tRPC offers, but integrated directly with Hono.
 *
 * Why Hono RPC?
 * -------------
 * 1. Type Safety: The client is typed with `AppType`, which is inferred from
 *    your Hono API routes. This means TypeScript will catch errors if you
 *    try to call endpoints that don't exist or pass incorrect parameters.
 *
 * 2. Lightweight: Hono is a small, fast framework. Using its built-in RPC
 *    client avoids the overhead of adding a separate tRPC dependency.
 *
 * 3. Unified Stack: Since the API is already built with Hono (see /app/api/[[...route]]/),
 *    using Hono's RPC client keeps everything consistent and reduces complexity.
 *
 * How it works:
 * -------------
 * - `hc<AppType>()` creates a client that mirrors your API structure
 * - You can call API endpoints like: `client.api.accounts.$get()`
 * - All requests and responses are fully typed based on your route definitions
 *
 * Usage Example:
 * --------------
 * ```typescript
 * const response = await client.api.accounts.$get();
 * const data = await response.json();
 * // `data` is fully typed based on your API response
 * ```
 */

import { hc } from 'hono/client';

import { AppType } from '@/app/api/[[...route]]/route';

export const client = hc<AppType>(process.env.NEXT_PUBLIC_API_URL!);
