# API changelog

## 2026-08-17 — Operational readiness (additive)

- Added public `GET /api/ready`. It returns 200 only while the instance accepts
  traffic and its required database dependency answers within the bounded
  readiness timeout; draining or dependency failure returns 503.
- `GET /api/health` remains the cheap process-liveness endpoint and does not
  claim database readiness.
- Responses now carry `X-Request-ID` for correlation. Request telemetry uses
  stable operation/route names and excludes headers, query values, bodies, and
  user/resource identifiers.
- Added CI guards for OpenAPI/sqlc generated drift and common backward-
  incompatible contract edits.

No existing endpoint status, body, authorization, atomicity, retry, or rate
limit semantics changed in this phase. The auth-internal-error contract remains
gated on [#119](https://github.com/GonzaloSecades/nuchi/issues/119), and bulk
body-limit contract work remains gated on
[#125](https://github.com/GonzaloSecades/nuchi/issues/125).
