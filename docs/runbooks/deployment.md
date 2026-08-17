# Deployment, drain, and rollback

**Owner:** release deployer. **Escalate:** repository owner for migration,
security, or data-integrity failures.

## Pre-deploy

- required CI gates are green, generated artifacts have no drift, and OpenAPI
  compatibility is reviewed;
- secrets are supplied outside the image; `AUTH_COOKIE_SECURE=true` behind
  HTTPS; and
- the database migration and application versions are mutually compatible.

## Deploy and verify

1. Deploy one instance with `APP_ENVIRONMENT`, `APP_VERSION`, and
   `APP_INSTANCE_ID` set to stable, non-sensitive values.
2. Wait for `/api/ready` 200. `/api/health` 200 alone proves only process
   liveness and is insufficient to route traffic.
3. Verify a public auth request and an authenticated read without placing
   credentials in logs or shell history.
4. Roll forward gradually while watching readiness, 5xx by operation, database
   acquire latency, and auth outcomes.

## Drain and rollback

SIGTERM flips readiness to 503 before the server waits up to ten seconds for
in-flight requests. Remove the instance from routing, then replace it. Roll back
application code only while the database remains compatible; otherwise use the
reviewed forward-fix plan. Stop the rollout on isolation, migration, sustained
readiness, or elevated-5xx failures.
