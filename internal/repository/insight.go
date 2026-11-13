package repository

import (
	"context"
	"time"

	"backend/internal/domain"
	"backend/internal/repository/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type insightRepository struct {
	q *database.Queries
}

func NewInsightRepository(db *pgxpool.Pool) domain.InsightRepository {
	return &insightRepository{q: database.New(db)}
}

func (r *insightRepository) Create(ctx context.Context, in domain.Insight) (domain.Insight, error) {
	row, err := r.q.CreateInsight(ctx, database.CreateInsightParams{
		DecisionID:           in.DecisionID,
		ModelVersion:         in.ModelVersion,
		TriggerContext:       string(in.TriggerContext),
		InsightText:          in.InsightText,
		PrimaryBias:          textFromPtr(in.PrimaryBias),
		EmotionalTone:        textFromPtr(in.EmotionalTone),
		RiskLevel:            textFromPtr(in.RiskLevel),
		ActionabilityScore:   int4FromPtr(in.ActionabilityScore),
		SuggestedAlternative: textFromPtr(in.SuggestedAlternative),
		ModelConfidence:      float4FromPtr(in.ModelConfidence),
	})
	if err != nil {
		return domain.Insight{}, err
	}
	return insightFromRow(row), nil
}

func (r *insightRepository) ListByDecision(ctx context.Context, decisionID, userID uuid.UUID) ([]domain.Insight, error) {
	rows, err := r.q.ListInsightsForDecision(ctx, database.ListInsightsForDecisionParams{
		DecisionID: decisionID,
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Insight, len(rows))
	for i, row := range rows {
		out[i] = insightFromRow(database.AiInsight(row))
	}
	return out, nil
}

func (r *insightRepository) CountForUserSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	return r.q.CountInsightsForUserSince(ctx, database.CountInsightsForUserSinceParams{
		UserID: userID,
		Since:  pgtype.Timestamptz{Time: since, Valid: true},
	})
}

func insightFromRow(row database.AiInsight) domain.Insight {
	return domain.Insight{
		ID:             row.ID,
		DecisionID:     row.DecisionID,
		ModelVersion:   row.ModelVersion,
		TriggerContext: domain.InsightTrigger(row.TriggerContext),
		InsightAnalysis: domain.InsightAnalysis{
			InsightText:          row.InsightText,
			PrimaryBias:          ptrFromText(row.PrimaryBias),
			EmotionalTone:        ptrFromText(row.EmotionalTone),
			RiskLevel:            ptrFromText(row.RiskLevel),
			ActionabilityScore:   ptrFromInt4(row.ActionabilityScore),
			SuggestedAlternative: ptrFromText(row.SuggestedAlternative),
			ModelConfidence:      ptrFromFloat4(row.ModelConfidence),
		},
		GeneratedAt: row.GeneratedAt.Time,
	}
}

func textFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func int4FromPtr(n *int) pgtype.Int4 {
	if n == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*n), Valid: true}
}

func float4FromPtr(f *float32) pgtype.Float4 {
	if f == nil {
		return pgtype.Float4{}
	}
	return pgtype.Float4{Float32: *f, Valid: true}
}

func ptrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func ptrFromInt4(n pgtype.Int4) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

func ptrFromFloat4(f pgtype.Float4) *float32 {
	if !f.Valid {
		return nil
	}
	return &f.Float32
}
