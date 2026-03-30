-- ==========================================
-- USER & AUTH QUERIES
-- ==========================================

-- name: CreateUser :one
INSERT INTO users (email, display_name) 
VALUES ($1, $2) 
RETURNING id, email, display_name, created_at;

-- name: CreateLocalCredential :exec
INSERT INTO local_credentials (user_id, password_hash) 
VALUES ($1, $2);

-- name: GetUserByEmail :one
SELECT id, email, display_name, created_at 
FROM users 
WHERE email = $1 LIMIT 1;

-- name: GetPasswordHash :one
SELECT password_hash 
FROM local_credentials 
WHERE user_id = $1 LIMIT 1;

-- name: CreateSession :exec
INSERT INTO sessions (user_id, token, user_agent, ip_address, expires_at) 
VALUES ($1, $2, $3, $4, $5);

-- name: GetSessionByToken :one
SELECT user_id, expires_at 
FROM sessions 
WHERE token = $1 AND expires_at > NOW() LIMIT 1;


-- ==========================================
-- DECISION QUERIES
-- ==========================================

-- name: CreateDecision :one
INSERT INTO decisions (user_id, title) 
VALUES ($1, $2) 
RETURNING id;

-- name: GetDecisionsByUserId :many
SELECT id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at 
FROM decisions 
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;