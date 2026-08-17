# Performance and slow-query response

**Owner:** backend deployer. **Escalate:** repository owner before index,
pool-size, timeout, caching, or query-semantic changes.

## Signals and objective

Compare operation latency by stable operation ID and database work by stable
sqlc query name. The Phase 3 engineering objectives and the reproducible
100,000-row dataset live in the backend-improvement evidence; they are not a
user-facing SLO until production traffic is characterized.

## Safe response

1. Record UTC window, build version, instance, operation, status class, and
   range/batch-size histogram buckets—never user, resource, payee, or note.
2. Check database saturation first; a pool wait can make every query look slow.
3. Reproduce against the sanitized performance fixture using the runtime role
   and `EXPLAIN (ANALYZE, BUFFERS, SETTINGS, FORMAT JSON)`.
4. Compare the stored plan, rows removed, sort method, buffer reads, and query
   count. Confirm RLS remains active.
5. Test one measured change at a time. Re-run behavior, isolation, and maximum-
   range tests before proposing it.

Stop an experiment if it changes summary semantics, ordering, ownership, or
money/date behavior. Do not enable unauthenticated profiling/debug endpoints or
log raw SQL values.
