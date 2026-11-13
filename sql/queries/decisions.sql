-- ==========================================
-- DECISION QUERIES
-- ==========================================

-- name: CreateDecision :one
INSERT INTO decisions (user_id, title, context, confidence_before, scheduled_review_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at;

-- name: GetDecisionByID :one
SELECT id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at, deleted_at
FROM decisions
WHERE id = $1 AND user_id = $2;

-- name: GetActiveDecisionsByUserID :many
SELECT id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at
FROM decisions
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateDecision :one
UPDATE decisions
SET title = $3,
    context = $4,
    confidence_before = $5,
    scheduled_review_date = $6,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at;

-- name: UpdateDecisionWithLWW :one
UPDATE decisions
SET title = $3,
    context = $4,
    confidence_before = $5,
    scheduled_review_date = $6,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL AND updated_at <= $7
RETURNING id, user_id, title, context, confidence_before, scheduled_review_date, created_at, updated_at;

-- name: SoftDeleteDecision :exec
UPDATE decisions
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: SoftDeleteDecisionWithLWW :execrows
UPDATE decisions
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL AND (sqlc.narg('local_updated_at')::timestamptz IS NULL OR updated_at <= @local_updated_at);

-- ==========================================
-- OPTION QUERIES
-- ==========================================

-- name: CreateOption :one
INSERT INTO options (decision_id, title, pros, cons)
VALUES ($1, $2, $3, $4)
RETURNING id, decision_id, title, pros, cons, created_at;

-- name: GetOptionsByDecisionID :many
SELECT id, decision_id, title, pros, cons, created_at
FROM options
WHERE decision_id = $1
ORDER BY created_at ASC;

-- name: GetOptionByID :one
SELECT id, decision_id, title, pros, cons, created_at
FROM options
WHERE id = $1 AND decision_id = $2;

-- name: DeleteOptionByID :exec
DELETE FROM options
WHERE id = $1 AND decision_id = $2;

-- ==========================================
-- DECISION OUTCOME QUERIES
-- ==========================================

-- name: UpsertDecisionOutcome :one
INSERT INTO decision_outcomes (decision_id, chosen_option_id, commitment_note, decided_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (decision_id) DO UPDATE
SET chosen_option_id = EXCLUDED.chosen_option_id,
    commitment_note = EXCLUDED.commitment_note,
    decided_at = EXCLUDED.decided_at
RETURNING decision_id, chosen_option_id, commitment_note, decided_at;

-- name: GetDecisionOutcome :one
SELECT decision_id, chosen_option_id, commitment_note, decided_at
FROM decision_outcomes
WHERE decision_id = $1;

-- ==========================================
-- EVALUATION QUERIES
-- ==========================================

-- name: CreateEvaluation :one
INSERT INTO evaluations (decision_id, satisfaction_after, confidence_after, hindsight_notes)
VALUES ($1, $2, $3, $4)
RETURNING id, decision_id, satisfaction_after, confidence_after, hindsight_notes, created_at;

-- name: UpsertEvaluation :one
INSERT INTO evaluations (decision_id, satisfaction_after, confidence_after, hindsight_notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (decision_id) DO UPDATE
SET satisfaction_after = EXCLUDED.satisfaction_after,
    confidence_after = EXCLUDED.confidence_after,
    hindsight_notes = EXCLUDED.hindsight_notes
RETURNING id, decision_id, satisfaction_after, confidence_after, hindsight_notes, created_at;

-- name: GetEvaluationByDecisionID :one
SELECT id, decision_id, satisfaction_after, confidence_after, hindsight_notes, created_at
FROM evaluations
WHERE decision_id = $1;
