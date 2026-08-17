# Security and session response

**Owner:** backend deployer. **Escalate:** immediately to the repository owner
for suspected token/key compromise or cross-user data access.

## Signals

- abnormal `loginUser`, `refreshSession`, or token-reuse outcomes;
- sustained rate-limit spikes by policy class;
- reports of a session surviving logout/password reset; or
- any evidence that one user can address another user's resource.

## Safe response

1. Record the UTC window, build version, instance, operation IDs, and public
   request IDs. Do not record token or cookie values.
2. For an isolation report, stop the affected instance from receiving traffic
   (`/api/ready` must return 503) and preserve database/log evidence.
3. Confirm the runtime database role is neither superuser nor `BYPASSRLS`:
   `SELECT current_user, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user;`.
4. For key compromise, rotate `AUTH_JWT_SECRET`, deploy all instances together,
   and expect every access token to become invalid. The current single-key JWT
   design has no overlap window; coordinate the forced sign-in explicitly.
5. For refresh-token compromise, revoke affected hashed token rows through an
   approved operator change. Never query or export raw submitted tokens.

## Stop conditions

Do not resume traffic when RLS is bypassed, cross-owner access is reproducible,
or all instances do not share the intended JWT key. Do not improvise direct
production data updates without an reviewed, reversible statement.
