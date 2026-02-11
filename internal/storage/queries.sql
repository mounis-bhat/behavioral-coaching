-- Users

-- name: CreateUser :one
INSERT INTO users (email, email_verified, name, picture, password_hash, provider, google_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at FROM users WHERE email = $1;

-- name: GetUserByGoogleID :one
SELECT id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at FROM users WHERE google_id = $1;

-- name: UpsertUserByGoogleID :one
INSERT INTO users (email, email_verified, name, picture, password_hash, provider, google_id)
VALUES ($1, $2, $3, $4, NULL, 'google', $5)
ON CONFLICT (google_id) DO UPDATE
SET email = EXCLUDED.email,
    email_verified = EXCLUDED.email_verified,
    name = EXCLUDED.name,
    picture = EXCLUDED.picture,
    provider = 'google'
RETURNING id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET name = COALESCE(sqlc.narg('name'), name),
    picture = COALESCE(sqlc.narg('picture'), picture),
    email_verified = COALESCE(sqlc.narg('email_verified'), email_verified),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = sqlc.arg('id')
RETURNING id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at;

-- name: SetEmailVerificationToken :exec
UPDATE users
SET email_verification_token_hash = $2,
    email_verification_expires_at = $3
WHERE id = $1;

-- name: GetUserByEmailVerificationTokenHash :one
SELECT id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at
FROM users
WHERE email_verification_token_hash = $1;

-- name: VerifyUserEmail :one
UPDATE users
SET email_verified = TRUE,
    email_verification_token_hash = NULL,
    email_verification_expires_at = NULL
WHERE id = $1
RETURNING id, email, email_verified, name, picture, password_hash, provider, google_id, email_verification_token_hash, email_verification_expires_at, failed_login_attempts, locked_until, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: IncrementFailedLoginAttempts :one
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1
WHERE id = $1
RETURNING *;

-- name: ResetFailedLoginAttempts :exec
UPDATE users
SET failed_login_attempts = 0
WHERE id = $1;

-- name: LockUser :exec
UPDATE users
SET locked_until = $2
WHERE id = $1;

-- name: UnlockUser :exec
UPDATE users
SET locked_until = NULL, failed_login_attempts = 0
WHERE id = $1;

-- Sessions

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT s.*, u.id AS "user.id", u.email AS "user.email", u.email_verified AS "user.email_verified",
       u.name AS "user.name", u.picture AS "user.picture", u.provider AS "user.provider"
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token_hash = $1 AND s.expires_at > NOW();

-- name: UpdateSessionLastActive :exec
UPDATE sessions
SET last_active_at = NOW()
WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: CountUserSessions :one
SELECT COUNT(*) FROM sessions WHERE user_id = $1;

-- name: GetOldestUserSession :one
SELECT * FROM sessions
WHERE user_id = $1
ORDER BY created_at ASC
LIMIT 1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();

-- Audit logs

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, event_type, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5);

-- name: PurgeAuditLogsBefore :one
WITH deleted AS (
    DELETE FROM audit_logs
    WHERE created_at < $1
    RETURNING 1
)
SELECT COUNT(*) FROM deleted;
