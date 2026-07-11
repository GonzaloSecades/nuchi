-- Queries for the three token tables (email_verification_tokens,
-- password_reset_tokens, refresh_tokens). None of these tables carry RLS
-- (auth-layer, decided in #38); ownership is still expressed via explicit
-- user_id predicates everywhere a query is scoped to a user.

-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(expires_at))
RETURNING *;

-- name: ConsumeEmailVerificationToken :one
-- Atomic one-time consume: a single UPDATE guarded by used_at IS NULL AND
-- expires_at > now() so two concurrent submits of the same token cannot both
-- succeed. Returns pgx.ErrNoRows when the token is unknown, already used, or
-- expired.
UPDATE email_verification_tokens
SET used_at = now()
WHERE token_hash = sqlc.arg(token_hash)
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_id;

-- name: InvalidateUserEmailVerificationTokens :exec
-- Marks all of a user's outstanding email verification tokens as used.
-- Issuing a new token invalidates prior ones (#41 decision).
UPDATE email_verification_tokens
SET used_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND used_at IS NULL;

-- name: CountRecentEmailVerificationTokens :one
-- Rate-limit support: how many tokens have been issued to this user since a
-- given timestamp.
SELECT count(*)
FROM email_verification_tokens
WHERE user_id = sqlc.arg(user_id)
  AND created_at > sqlc.arg(since);

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(expires_at))
RETURNING *;

-- name: ConsumePasswordResetToken :one
-- Same atomic one-time consume shape as ConsumeEmailVerificationToken.
UPDATE password_reset_tokens
SET used_at = now()
WHERE token_hash = sqlc.arg(token_hash)
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_id;

-- name: InvalidateUserPasswordResetTokens :exec
-- Issuing a new reset token invalidates prior ones; a completed password
-- change also calls this to kill any other outstanding reset tokens (#41
-- decision).
UPDATE password_reset_tokens
SET used_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND used_at IS NULL;

-- name: CountRecentPasswordResetTokens :one
SELECT count(*)
FROM password_reset_tokens
WHERE user_id = sqlc.arg(user_id)
  AND created_at > sqlc.arg(since);

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(expires_at))
RETURNING *;

-- name: GetRefreshTokenByHash :one
-- Only returns a row when the token is currently valid (not revoked, not
-- expired); filtering happens in SQL rather than leaving expiry/revocation
-- checks to the caller.
SELECT *
FROM refresh_tokens
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeRefreshToken :exec
-- Revokes a single refresh token by hash (e.g. logout). Idempotent: revoking
-- an already-revoked token affects zero rows rather than erroring.
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
-- Logout-everywhere / post-password-reset revocation of every outstanding
-- refresh token for a user.
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL;
