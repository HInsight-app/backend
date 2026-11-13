package response

import (
	"time"

	"github.com/google/uuid"
)

type InsightResponse struct {
	ID                   uuid.UUID `json:"id"`
	DecisionID           uuid.UUID `json:"decision_id"`
	ModelVersion         string    `json:"model_version"`
	TriggerContext       string    `json:"trigger_context"`
	InsightText          string    `json:"insight_text"`
	PrimaryBias          *string   `json:"primary_bias"`
	EmotionalTone        *string   `json:"emotional_tone"`
	RiskLevel            *string   `json:"risk_level"`
	ActionabilityScore   *int      `json:"actionability_score"`
	SuggestedAlternative *string   `json:"suggested_alternative"`
	ModelConfidence      *float32  `json:"model_confidence"`
	GeneratedAt          time.Time `json:"generated_at"`
}
