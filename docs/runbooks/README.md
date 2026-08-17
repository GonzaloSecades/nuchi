# Backend runbooks

These runbooks define the safe first response for Nuchi's production-facing
failure modes. Until a named rotation exists, the owner is the backend deployer
for the affected release and escalation goes to the repository owner.

- [Security and session response](security.md)
- [Database saturation and migration response](database.md)
- [Performance and slow-query response](performance.md)
- [Deployment, drain, and rollback](deployment.md)
- [Elevated errors and incident coordination](incident-response.md)

Never paste tokens, cookies, database URLs, raw request bodies, financial notes,
or user-identifying values into a ticket or chat. Preserve sanitized timestamps,
request IDs, operation IDs, build versions, and aggregate counts instead.
