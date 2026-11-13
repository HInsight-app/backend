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

type mockDecisionRepo struct {
	createFunc            func(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error)
	getByIDFunc           func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error)
	getActiveByUserIDFunc func(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error)
	updateWithLWWFunc     func(ctx context.Context, d domain.Decision, localUpdatedAt time.Time) (domain.Decision, error)
	softDeleteWithLWWFunc func(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error
	createOptionFunc      func(ctx context.Context, opt domain.Option) (domain.Option, error)
	deleteOptionFunc      func(ctx context.Context, optionID, decisionID, userID uuid.UUID) error
	getOptionByIDFunc     func(ctx context.Context, optionID, decisionID uuid.UUID) (domain.Option, error)
	upsertOutcomeFunc     func(ctx context.Context, outcome domain.DecisionOutcome, userID uuid.UUID) (domain.DecisionOutcome, error)
	upsertEvaluationFunc  func(ctx context.Context, eval domain.Evaluation, userID uuid.UUID) (domain.Evaluation, error)
	addTagFunc            func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	removeTagFunc         func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
}

func (m *mockDecisionRepo) Create(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, d, initialOptions, initialTagIDs)
	}
	return d, nil
}

func (m *mockDecisionRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, userID)
	}
	return domain.Decision{ID: id, UserID: userID}, nil
}

func (m *mockDecisionRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	if m.getActiveByUserIDFunc != nil {
		return m.getActiveByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockDecisionRepo) UpdateWithLWW(ctx context.Context, d domain.Decision, localUpdatedAt time.Time) (domain.Decision, error) {
	if m.updateWithLWWFunc != nil {
		return m.updateWithLWWFunc(ctx, d, localUpdatedAt)
	}
	return d, nil
}

func (m *mockDecisionRepo) SoftDeleteWithLWW(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error {
	if m.softDeleteWithLWWFunc != nil {
		return m.softDeleteWithLWWFunc(ctx, id, userID, localUpdatedAt)
	}
	return nil
}

func (m *mockDecisionRepo) CreateOption(ctx context.Context, opt domain.Option) (domain.Option, error) {
	if m.createOptionFunc != nil {
		return m.createOptionFunc(ctx, opt)
	}
	return opt, nil
}

func (m *mockDecisionRepo) DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error {
	if m.deleteOptionFunc != nil {
		return m.deleteOptionFunc(ctx, optionID, decisionID, userID)
	}
	return nil
}

func (m *mockDecisionRepo) GetOptionByID(ctx context.Context, optionID, decisionID uuid.UUID) (domain.Option, error) {
	if m.getOptionByIDFunc != nil {
		return m.getOptionByIDFunc(ctx, optionID, decisionID)
	}
	return domain.Option{ID: optionID, DecisionID: decisionID}, nil
}

func (m *mockDecisionRepo) UpsertOutcome(ctx context.Context, outcome domain.DecisionOutcome, userID uuid.UUID) (domain.DecisionOutcome, error) {
	if m.upsertOutcomeFunc != nil {
		return m.upsertOutcomeFunc(ctx, outcome, userID)
	}
	return outcome, nil
}

func (m *mockDecisionRepo) UpsertEvaluation(ctx context.Context, eval domain.Evaluation, userID uuid.UUID) (domain.Evaluation, error) {
	if m.upsertEvaluationFunc != nil {
		return m.upsertEvaluationFunc(ctx, eval, userID)
	}
	return eval, nil
}

func (m *mockDecisionRepo) AddTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	if m.addTagFunc != nil {
		return m.addTagFunc(ctx, decisionID, tagID, userID)
	}
	return nil
}

func (m *mockDecisionRepo) RemoveTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	if m.removeTagFunc != nil {
		return m.removeTagFunc(ctx, decisionID, tagID, userID)
	}
	return nil
}

func TestCreateDecision_Validation(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()

	// Test 1: Empty Title
	_, err := svc.CreateDecision(context.Background(), userID, "   ", nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}

	// Test 2: Valid Title and Options
	optTitle := "Option A"
	repo.createFunc = func(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error) {
		if d.Title != "Valid Decision" {
			t.Fatalf("expected title 'Valid Decision', got '%s'", d.Title)
		}
		if len(initialOptions) != 1 || initialOptions[0].Title != optTitle {
			t.Fatalf("expected 1 option with title '%s', got %v", optTitle, initialOptions)
		}
		d.ID = uuid.New()
		d.Options = initialOptions
		return d, nil
	}

	created, err := svc.CreateDecision(context.Background(), userID, "  Valid Decision  ", nil, nil, nil, []domain.Option{
		{Title: "  Option A  "},
		{Title: "   "}, // Should be filtered out
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Title != "Valid Decision" {
		t.Fatalf("expected 'Valid Decision', got '%s'", created.Title)
	}
}

func TestCreateDecision_WithInitialTags(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	tagID := uuid.New()

	repo.createFunc = func(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error) {
		if len(initialTagIDs) != 1 || initialTagIDs[0] != tagID {
			t.Fatalf("expected initial tag %s, got %v", tagID, initialTagIDs)
		}
		d.ID = uuid.New()
		d.Tags = []domain.Tag{{ID: tagID, Name: "Tag1", ColorHex: "#FFFFFF"}}
		return d, nil
	}

	created, err := svc.CreateDecision(context.Background(), userID, "Decision With Tag", nil, nil, nil, nil, []uuid.UUID{tagID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created.Tags) != 1 || created.Tags[0].ID != tagID {
		t.Fatalf("expected created decision to have tag %s", tagID)
	}
}

func TestUpdateDecision_LWWConflict(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	serverTime := time.Now().UTC()
	staleClientTime := serverTime.Add(-10 * time.Minute)

	repo.updateWithLWWFunc = func(ctx context.Context, d domain.Decision, localUpdatedAt time.Time) (domain.Decision, error) {
		if localUpdatedAt.Before(serverTime) {
			return domain.Decision{}, &domain.ErrConflict{
				ServerDecision: domain.Decision{
					ID:        decisionID,
					UserID:    userID,
					Title:     "Server New Title",
					UpdatedAt: serverTime,
				},
			}
		}
		d.UpdatedAt = localUpdatedAt
		return d, nil
	}

	_, err := svc.UpdateDecision(context.Background(), userID, decisionID, "Client Update", nil, nil, nil, staleClientTime)
	if err == nil {
		t.Fatal("expected ErrConflict, got nil")
	}

	var conflictErr *domain.ErrConflict
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *domain.ErrConflict, got %T: %v", err, err)
	}
	if conflictErr.ServerDecision.Title != "Server New Title" {
		t.Fatalf("expected server decision title 'Server New Title', got '%s'", conflictErr.ServerDecision.Title)
	}
}

func TestDeleteOption_CannotDeleteChosenOption(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()
	chosenOptionID := uuid.New()

	repo.getByIDFunc = func(ctx context.Context, id, uid uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:     decisionID,
			UserID: userID,
			Outcome: &domain.DecisionOutcome{
				DecisionID:     decisionID,
				ChosenOptionID: chosenOptionID,
			},
		}, nil
	}

	// Try deleting the chosen option
	err := svc.DeleteOption(context.Background(), chosenOptionID, decisionID, userID)
	if err == nil {
		t.Fatal("expected error deleting chosen option, got nil")
	}
	if err.Error() != "cannot delete option chosen for this decision's commitment" {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Deleting another option should succeed
	otherOptionID := uuid.New()
	deleteCalled := false
	repo.deleteOptionFunc = func(ctx context.Context, optionID, decID, uid uuid.UUID) error {
		if optionID == otherOptionID {
			deleteCalled = true
		}
		return nil
	}

	if err := svc.DeleteOption(context.Background(), otherOptionID, decisionID, userID); err != nil {
		t.Fatalf("unexpected error deleting non-chosen option: %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected repo.DeleteOption to be called for other option")
	}
}

func TestCommitDecision(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()
	chosenOptionID := uuid.New()
	note := "Final choice"

	repo.upsertOutcomeFunc = func(ctx context.Context, outcome domain.DecisionOutcome, uid uuid.UUID) (domain.DecisionOutcome, error) {
		if outcome.ChosenOptionID != chosenOptionID {
			t.Fatalf("expected chosen option %s, got %s", chosenOptionID, outcome.ChosenOptionID)
		}
		if outcome.CommitmentNote == nil || *outcome.CommitmentNote != note {
			t.Fatalf("expected commitment note '%s', got %v", note, outcome.CommitmentNote)
		}
		outcome.DecidedAt = time.Now()
		return outcome, nil
	}

	outcome, err := svc.CommitDecision(context.Background(), decisionID, userID, chosenOptionID, &note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.ChosenOptionID != chosenOptionID {
		t.Fatalf("expected chosen option %s, got %s", chosenOptionID, outcome.ChosenOptionID)
	}
}

func TestCreateEvaluation_RequiresCommitment(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	// Decision has NO outcome
	repo.getByIDFunc = func(ctx context.Context, id, uid uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:      decisionID,
			UserID:  userID,
			Outcome: nil,
		}, nil
	}

	_, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "Hindsight notes")
	if err == nil {
		t.Fatal("expected error evaluating uncommitted decision, got nil")
	}
	if err.Error() != "decision must be committed before it can be evaluated" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateEvaluation_Validation(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	// Empty hindsight notes
	_, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "   ")
	if err == nil {
		t.Fatal("expected error for empty hindsight notes, got nil")
	}

	// Decision has an outcome (committed)
	repo.getByIDFunc = func(ctx context.Context, id, uid uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:     decisionID,
			UserID: userID,
			Outcome: &domain.DecisionOutcome{
				DecisionID:     decisionID,
				ChosenOptionID: uuid.New(),
			},
		}, nil
	}

	// Valid evaluation
	repo.upsertEvaluationFunc = func(ctx context.Context, eval domain.Evaluation, uid uuid.UUID) (domain.Evaluation, error) {
		eval.ID = uuid.New()
		eval.CreatedAt = time.Now()
		return eval, nil
	}

	eval, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "Learned a lot from this choice.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eval.SatisfactionAfter != 80 || eval.ConfidenceAfter != 75 {
		t.Fatalf("expected scores 80 and 75, got %d and %d", eval.SatisfactionAfter, eval.ConfidenceAfter)
	}
}
