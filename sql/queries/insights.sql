-- ==========================================
-- AI_INSIGHT QUERIES
-- ==========================================

-- name: CreateInsight :one
INSERT INTO ai_insights (
    decision_id, model_version, trigger_context, insight_text,
    primary_bias, emotional_tone, risk_level, actionability_score,
    suggested_alternative, model_confidence
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, decision_id, model_version, trigger_context, insight_text,
    primary_bias, emotional_tone, risk_level, actionability_score,
    suggested_alternative, model_confidence, generated_at;

-- name: ListInsightsForDecision :many
-- Joins decisions so a user can only read insights for their own, non-deleted decisions.
SELECT i.id, i.decision_id, i.model_version, i.trigger_context, i.insight_text,
    i.primary_bias, i.emotional_tone, i.risk_level, i.actionability_score,
    i.suggested_alternative, i.model_confidence, i.generated_at
FROM ai_insights i
JOIN decisions d ON d.id = i.decision_id
WHERE i.decision_id = @decision_id AND d.user_id = @user_id AND d.deleted_at IS NULL
ORDER BY i.generated_at DESC;

-- name: CountInsightsForUserSince :one
-- Used for the per-user daily limit on Gemini calls.
SELECT COUNT(*)
FROM ai_insights i
JOIN decisions d ON d.id = i.decision_id
WHERE d.user_id = @user_id AND i.generated_at >= @since;
