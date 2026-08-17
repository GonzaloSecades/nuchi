# Elevated errors and incident coordination

**Owner:** backend deployer as incident lead until handed off. **Escalate:**
repository owner for security, data integrity, prolonged unavailability, or a
decision to alter production data.

## First fifteen minutes

1. State impact and UTC start time; identify affected operation IDs, status
   classes, versions, and instances.
2. Separate liveness, readiness, database saturation, dependency/mail failure,
   validation/client errors, and true internal errors.
3. Use request IDs to correlate sanitized boundary logs. Do not request or copy
   cookies, tokens, raw bodies, database URLs, emails, payees, or notes.
4. Drain only unhealthy versions/instances. If impact followed a deploy and the
   database is backward compatible, follow the deployment rollback.
5. Post a concise timeline of actions, observations, and owners.

## Recovery and follow-up

Confirm readiness, representative auth and resource flows, stable error rate,
and database headroom before declaring recovery. Preserve sanitized evidence.
Follow up with the initiating change, detection gap, user impact, and a tested
preventive action; do not treat deleting evidence or widening log collection as
a remediation.
