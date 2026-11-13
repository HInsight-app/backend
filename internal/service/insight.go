package service

import (
	"context"
	"log"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
)

// DefaultDailyInsightLimit caps Gemini calls per user per 24 hours, since every
// call is billed to the server's API key.
const DefaultDailyInsightLimit = 20

type insightService struct {
	decisions  domain.DecisionRepository
	insights   domain.InsightRepository
	generator  domain.InsightGenerator // nil when GEMINI_API_KEY is not set
	dailyLimit int
	now        func() time.Time
}

// NewInsightService wires the insight feature. generator may be nil, in which
// case generating returns domain.ErrInsightsDisabled but listing still works.
// dailyLimit <= 0 uses DefaultDailyInsightLimit.
func NewInsightService(decisions domain.DecisionRepository, insights domain.InsightRepository, generator domain.InsightGenerator, dailyLimit int) domain.InsightService {
	if dailyLimit <= 0 {
		dailyLimit = DefaultDailyInsightLimit
	}
	return &insightService{
		decisions:  decisions,
		insights:   insights,
		generator:  generator,
		dailyLimit: dailyLimit,
		now:        time.Now,
	}
}

func (s *insightService) GenerateInsight(ctx context.Context, decisionID, userID uuid.UUID, trigger domain.InsightTrigger) (domain.Insight, error) {
	if !trigger.Valid() {
		return domain.Insight{}, domain.ErrInvalidInsightTrigger
	}
	if s.generator == nil {
		return domain.Insight{}, domain.ErrInsightsDisabled
	}

	// Loads only the caller's own, non-deleted decision, so nothing belonging to
	// another user is ever sent to the model.
	decision, err := s.decisions.GetByID(ctx, decisionID, userID)
	if err != nil {
		return domain.Insight{}, err
	}
	if trigger == domain.TriggerPostEvaluation && decision.Evaluation == nil {
		return domain.Insight{}, domain.ErrInsightRequiresEvaluation
	}

	used, err := s.insights.CountForUserSince(ctx, userID, s.now().Add(-24*time.Hour))
	if err != nil {
		return domain.Insight{}, err
	}
	if used >= int64(s.dailyLimit) {
		return domain.Insight{}, domain.ErrInsightRateLimited
	}

	analysis, err := s.generator.Analyze(ctx, decision, trigger)
	if err != nil {
		// Log the cause (never the prompt or decision text) and return a stable error.
		log.Printf("insight generation failed for decision %s: %v", decisionID, err)
		return domain.Insight{}, domain.ErrInsightGeneration
	}

	return s.insights.Create(ctx, domain.Insight{
		DecisionID:      decisionID,
		ModelVersion:    s.generator.ModelVersion(),
		TriggerContext:  trigger,
		InsightAnalysis: analysis,
	})
}

func (s *insightService) ListInsights(ctx context.Context, decisionID, userID uuid.UUID) ([]domain.Insight, error) {
	// Return 404 for a missing or foreign decision instead of an empty list.
	if _, err := s.decisions.GetByID(ctx, decisionID, userID); err != nil {
		return nil, err
	}
	return s.insights.ListByDecision(ctx, decisionID, userID)
}
