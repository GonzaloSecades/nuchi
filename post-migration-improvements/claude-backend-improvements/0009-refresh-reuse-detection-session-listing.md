# 0009 — No refresh-token reuse detection or session listing

- **Migration ticket:** #38 (schema), #41 (flows) (https://github.com/GonzaloSecades/nuchi/issues/41)
- **Area:** auth
- **Priority guess:** low

## How it was migrated

Refresh rotation is atomic (`ConsumeRefreshToken`: exactly one concurrent
winner) and revoked rows are kept, but there is no `replaced_by` lineage:
if a *rotated-away* token is replayed later, the API answers 401 and nothing
more. There is also no "active sessions" surface — the data exists
(`refresh_tokens` rows with `created_at`/`expires_at`/`revoked_at`, indexed
by `user_id`) but no query/endpoint lists it.

## Why it was done this way

Deliberately deferred in the #38 and #41 design reviews: reuse detection and
device management are real features with real UX, not one-liners, and the
migration ships a working session core first. The schema was shaped so both
remain additive.

## The concern

Replay of an old refresh token is a strong signal the token family is
compromised; best practice is to revoke the entire family on detection.
Today that signal is silently discarded. Separately, users cannot see or
kill individual sessions ("logout everywhere" exists as a query but has no
endpoint).

## Proposed improvement

Add `replaced_by uuid NULL` (or `family_id`) to `refresh_tokens`; on replay
of a consumed token, `RevokeAllUserRefreshTokens` for that user (query
already exists). Optionally surface `GET /auth/sessions` +
`DELETE /auth/sessions/:id` for a sessions screen. Both additive: one
column, no data migration, no contract break (new endpoints only).
