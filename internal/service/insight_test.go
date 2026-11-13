package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/domain"
	"backend/internal/service"

	"github.com/google/uuid"
)

type fakeInsightRepo struct {
	count   int64
	created []domain.Insight
	listed  []domain.Insight
}

func (f *fakeInsightRepo) Create(ctx context.Context, in domain.Insight) (domain.Insight, error) {
	in.ID = uuid.New()
	in.GeneratedAt = time.Now()
	f.created = append(f.created, in)
	return in, nil
}

func (f *fakeInsightRepo) ListByDecision(ctx context.Context, decisionID, userID uuid.UUID) ([]domain.Insight, error) {
	return f.listed, nil
}

func (f *fakeInsightRepo) CountForUserSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	return f.count, nil
}

type fakeGenerator struct {
	calls int
	err   error
}

func (f *fakeGenerator) Analyze(ctx context.Context, d domain.Decision, trigger domain.InsightTrigger) (domain.InsightAnalysis, error) {
	f.calls++
	if f.err != nil {
		return domain.InsightAnalysis{}, f.err
	}
	return domain.InsightAnalysis{InsightText: "You seem confident."}, nil
}

func (f *fakeGenerator) ModelVersion() string { return "test-model" }

func TestGenerateInsight_PreDecision_Saves(t *testing.T) {
	userID, decisionID := uuid.New(), uuid.New()
	repo := &fakeInsightRepo{}
	gen := &fakeGenerator{}
	svc := service.NewInsightService(&mockDecisionRepo{}, repo, gen, 5)

	got, err := svc.GenerateInsight(context.Background(), decisionID, userID, domain.TriggerPreDecision)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen.calls != 1 || len(repo.created) != 1 {
		t.Fatalf("expected one generator call and one saved insight, got %d and %d", gen.calls, len(repo.created))
	}
	if got.ModelVersion != "test-model" || got.TriggerContext != domain.TriggerPreDecision || got.DecisionID != decisionID {
		t.Fatalf("saved insight has wrong metadata: %+v", got)
	}
}

func TestGenerateInsight_Errors(t *testing.T) {
	notFound := &mockDecisionRepo{getByIDFunc: func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{}, domain.ErrDecisionNotFound
	}}

	tests := []struct {
		name      string
		decisions *mockDecisionRepo
		repo      *fakeInsightRepo
		gen       domain.InsightGenerator
		trigger   domain.InsightTrigger
		want      error
		wantCalls int
	}{
		{"invalid trigger", &mockDecisionRepo{}, &fakeInsightRepo{}, &fakeGenerator{}, "later", domain.ErrInvalidInsightTrigger, 0},
		{"no api key", &mockDecisionRepo{}, &fakeInsightRepo{}, nil, domain.TriggerPreDecision, domain.ErrInsightsDisabled, 0},
		{"foreign or missing decision", notFound, &fakeInsightRepo{}, &fakeGenerator{}, domain.TriggerPreDecision, domain.ErrDecisionNotFound, 0},
		{"post evaluation without evaluation", &mockDecisionRepo{}, &fakeInsightRepo{}, &fakeGenerator{}, domain.TriggerPostEvaluation, domain.ErrInsightRequiresEvaluation, 0},
		{"daily limit reached", &mockDecisionRepo{}, &fakeInsightRepo{count: 5}, &fakeGenerator{}, domain.TriggerPreDecision, domain.ErrInsightRateLimited, 0},
		{"model failure", &mockDecisionRepo{}, &fakeInsightRepo{}, &fakeGenerator{err: errors.New("timeout")}, domain.TriggerPreDecision, domain.ErrInsightGeneration, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewInsightService(tt.decisions, tt.repo, tt.gen, 5)
			_, err := svc.GenerateInsight(context.Background(), uuid.New(), uuid.New(), tt.trigger)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if fg, ok := tt.gen.(*fakeGenerator); ok && fg.calls != tt.wantCalls {
				t.Fatalf("expected %d generator calls, got %d", tt.wantCalls, fg.calls)
			}
			if len(tt.repo.created) != 0 {
				t.Fatalf("nothing should be saved on error")
			}
		})
	}
}

func TestGenerateInsight_PostEvaluation_WithEvaluation(t *testing.T) {
	decisions := &mockDecisionRepo{getByIDFunc: func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{ID: id, UserID: userID, Evaluation: &domain.Evaluation{SatisfactionAfter: 70}}, nil
	}}
	svc := service.NewInsightService(decisions, &fakeInsightRepo{}, &fakeGenerator{}, 5)

	if _, err := svc.GenerateInsight(context.Background(), uuid.New(), uuid.New(), domain.TriggerPostEvaluation); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListInsights_ChecksOwnership(t *testing.T) {
	decisions := &mockDecisionRepo{getByIDFunc: func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{}, domain.ErrDecisionNotFound
	}}
	repo := &fakeInsightRepo{listed: []domain.Insight{{ID: uuid.New()}}}
	svc := service.NewInsightService(decisions, repo, nil, 5)

	if _, err := svc.ListInsights(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, domain.ErrDecisionNotFound) {
		t.Fatalf("expected ErrDecisionNotFound, got %v", err)
	}
}

func TestListInsights_WorksWithoutAPIKey(t *testing.T) {
	repo := &fakeInsightRepo{listed: []domain.Insight{{ID: uuid.New()}, {ID: uuid.New()}}}
	svc := service.NewInsightService(&mockDecisionRepo{}, repo, nil, 5)

	got, err := svc.ListInsights(context.Background(), uuid.New(), uuid.New())
	if err != nil || len(got) != 2 {
		t.Fatalf("expected 2 insights and no error, got %d, %v", len(got), err)
	}
}
