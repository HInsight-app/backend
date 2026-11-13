package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// InsightTrigger says when an insight was requested. Values match the
// trigger_context CHECK constraint on ai_insights.
type InsightTrigger string

const (
	TriggerPreDecision    InsightTrigger = "pre_decision"
	TriggerPostEvaluation InsightTrigger = "post_evaluation"
)

func (t InsightTrigger) Valid() bool {
	return t == TriggerPreDecision || t == TriggerPostEvaluation
}

// Allowed values for the enum-like insight columns. They must stay in sync with
// the CHECK constraints in sql/migrations/000001_init_schema.up.sql.
var (
	InsightBiases         = []string{"confirmation", "loss_aversion", "sunk_cost", "overconfidence", "recency"}
	InsightEmotionalTones = []string{"neutral", "anxious", "confident", "uncertain", "regretful"}
	InsightRiskLevels     = []string{"low", "medium", "high"}
)

var (
	ErrInsightsDisabled          = errors.New("AI insights are not configured on this server")
	ErrInvalidInsightTrigger     = errors.New("trigger_context must be pre_decision or post_evaluation")
	ErrInsightRequiresEvaluation = errors.New("a post_evaluation insight needs an evaluation first")
	ErrInsightRateLimited        = errors.New("daily AI insight limit reached")
	ErrInsightGeneration         = errors.New("the AI model could not generate an insight")
)

// InsightAnalysis is what the AI model returns, already validated against the
// allowed values. Optional fields are nil when the model had nothing to say.
type InsightAnalysis struct {
	InsightText          string
	PrimaryBias          *string
	EmotionalTone        *string
	RiskLevel            *string
	ActionabilityScore   *int
	SuggestedAlternative *string
	ModelConfidence      *float32
}

// Insight is a stored AI analysis of a decision.
type Insight struct {
	ID             uuid.UUID
	DecisionID     uuid.UUID
	ModelVersion   string
	TriggerContext InsightTrigger
	InsightAnalysis
	GeneratedAt time.Time
}

// InsightGenerator produces an analysis for a decision. The Gemini client in
// internal/ai implements it; tests use a fake.
type InsightGenerator interface {
	Analyze(ctx context.Context, d Decision, trigger InsightTrigger) (InsightAnalysis, error)
	ModelVersion() string
}

type InsightRepository interface {
	Create(ctx context.Context, insight Insight) (Insight, error)
	ListByDecision(ctx context.Context, decisionID, userID uuid.UUID) ([]Insight, error)
	CountForUserSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
}

type InsightService interface {
	GenerateInsight(ctx context.Context, decisionID, userID uuid.UUID, trigger InsightTrigger) (Insight, error)
	ListInsights(ctx context.Context, decisionID, userID uuid.UUID) ([]Insight, error)
}
