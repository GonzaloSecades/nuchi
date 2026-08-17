# Database and migration response

**Owner:** backend deployer. **Escalate:** repository owner before destructive
recovery, restore, migration rollback, or a production role/grant change.

## Saturation signals

- readiness 503 with liveness 200;
- rising pool acquire timeout/error counts;
- database connection exhaustion or elevated query duration; and
- transaction rollback growth.

## Safe diagnostics

Run read-only queries as an administrative observer, never by changing the API
runtime role:

```sql
SELECT state, wait_event_type, wait_event, count(*)
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY 1, 2, 3;
```

```sql
SELECT pid, now() - query_start AS age, state, wait_event_type, wait_event
FROM pg_stat_activity
WHERE datname = current_database() AND state <> 'idle'
ORDER BY query_start;
```

Correlate with stable sqlc query names; do not copy raw parameter values.

## Migration procedure

1. Snapshot/backup and verify the restore target before a production migration.
2. Drain application instances, run `goose status`, then apply the reviewed
   migration with the dedicated migrator identity.
3. Run the migration's validation queries and deploy compatible application
   code. Use expand/migrate/contract across releases for incompatible changes.
4. Prefer a reviewed forward-fix. Run `goose down` only when the migration's
   checked-in rollback is proven safe for production data.

Stop if lock duration exceeds the migration's stated budget, validation counts
diverge, or readiness does not recover. Never terminate sessions or restore a
backup as a routine first response.
