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
	createFunc              func(ctx context.Context, d domain.Decision, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error)
	getByIDFunc             func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error)
	getByIdempotencyKeyFunc func(ctx context.Context, userID uuid.UUID, key string) (domain.Decision, error)
	getActiveByUserIDFunc   func(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error)
	listUpdatedSinceFunc    func(ctx context.Context, userID uuid.UUID, updatedSince time.Time) ([]domain.Decision, error)
	updateWithLWWFunc       func(ctx context.Context, d domain.Decision, localUpdatedAt time.Time) (domain.Decision, error)
	softDeleteWithLWWFunc   func(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error
	syncDecisionFunc        func(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error)
	createOptionFunc        func(ctx context.Context, opt domain.Option, userID uuid.UUID) (domain.Option, error)
	deleteOptionFunc        func(ctx context.Context, optionID, decisionID, userID uuid.UUID) error
	getOptionByIDFunc       func(ctx context.Context, optionID, decisionID uuid.UUID) (domain.Option, error)
	upsertOutcomeFunc       func(ctx context.Context, outcome domain.DecisionOutcome, userID uuid.UUID) (domain.DecisionOutcome, error)
	upsertEvaluationFunc    func(ctx context.Context, eval domain.Evaluation, userID uuid.UUID) (domain.Evaluation, error)
	addTagFunc              func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	removeTagFunc           func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
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

func (m *mockDecisionRepo) GetByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (domain.Decision, error) {
	if m.getByIdempotencyKeyFunc != nil {
		return m.getByIdempotencyKeyFunc(ctx, userID, key)
	}
	return domain.Decision{}, domain.ErrDecisionNotFound
}

func (m *mockDecisionRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	if m.getActiveByUserIDFunc != nil {
		return m.getActiveByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockDecisionRepo) ListUpdatedSince(ctx context.Context, userID uuid.UUID, updatedSince time.Time) ([]domain.Decision, error) {
	if m.listUpdatedSinceFunc != nil {
		return m.listUpdatedSinceFunc(ctx, userID, updatedSince)
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

func (m *mockDecisionRepo) SyncDecision(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
	if m.syncDecisionFunc != nil {
		return m.syncDecisionFunc(ctx, id, userID, data)
	}
	return domain.Decision{ID: id, UserID: userID, Title: data.Title}, nil
}

func (m *mockDecisionRepo) CreateOption(ctx context.Context, opt domain.Option, userID uuid.UUID) (domain.Option, error) {
	if m.createOptionFunc != nil {
		return m.createOptionFunc(ctx, opt, userID)
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
	_, err := svc.CreateDecision(context.Background(), userID, "   ", nil, nil, nil, nil, nil, nil, nil)
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
			t.Fatalf("expected 1 option with title '%s'", optTitle)
		}
		return d, nil
	}

	_, err = svc.CreateDecision(context.Background(), userID, "Valid Decision", nil, nil, nil, []domain.Option{{Title: optTitle}}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDecision_Idempotency(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	key := "idemp-key-123"

	existingDec := domain.Decision{
		ID:     uuid.New(),
		UserID: userID,
		Title:  "Existing Decision",
	}

	repo.getByIdempotencyKeyFunc = func(ctx context.Context, uID uuid.UUID, k string) (domain.Decision, error) {
		if k == key {
			return existingDec, nil
		}
		return domain.Decision{}, domain.ErrDecisionNotFound
	}

	// Should return existing decision without calling createFunc
	created, err := svc.CreateDecision(context.Background(), userID, "New Decision", nil, nil, nil, nil, nil, nil, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != existingDec.ID {
		t.Fatalf("expected existing decision ID %s, got %s", existingDec.ID, created.ID)
	}
}

func TestUpdateDecision_Validation(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	// Empty Title
	_, err := svc.UpdateDecision(context.Background(), userID, decisionID, "   ", nil, nil, nil, time.Now())
	if err == nil {
		t.Fatal("expected error for empty title on update, got nil")
	}
}

func TestDeleteOption_ChosenOptionCheck(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()
	chosenOptionID := uuid.New()

	repo.getByIDFunc = func(ctx context.Context, id, uID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:     decisionID,
			UserID: userID,
			Outcome: &domain.DecisionOutcome{
				DecisionID:     decisionID,
				ChosenOptionID: chosenOptionID,
			},
		}, nil
	}

	// Deleting chosen option should fail with ErrCannotDeleteChosenOption
	err := svc.DeleteOption(context.Background(), chosenOptionID, decisionID, userID)
	if !errors.Is(err, domain.ErrCannotDeleteChosenOption) {
		t.Fatalf("expected ErrCannotDeleteChosenOption, got %v", err)
	}

	// Deleting non-chosen option should succeed
	otherOptionID := uuid.New()
	repo.deleteOptionFunc = func(ctx context.Context, optionID, dID, uID uuid.UUID) error {
		if optionID != otherOptionID {
			t.Fatalf("expected option %s, got %s", otherOptionID, optionID)
		}
		return nil
	}

	err = svc.DeleteOption(context.Background(), otherOptionID, decisionID, userID)
	if err != nil {
		t.Fatalf("unexpected error deleting non-chosen option: %v", err)
	}
}

func TestCreateEvaluation_RequiresCommitment(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	// Decision with no outcome (uncommitted)
	repo.getByIDFunc = func(ctx context.Context, id, uID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:      decisionID,
			UserID:  userID,
			Outcome: nil,
		}, nil
	}

	_, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "Great decision", nil)
	if !errors.Is(err, domain.ErrEvaluationRequiresCommitment) {
		t.Fatalf("expected ErrEvaluationRequiresCommitment, got: %v", err)
	}

	// Decision with outcome (committed)
	repo.getByIDFunc = func(ctx context.Context, id, uID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:     decisionID,
			UserID: userID,
			Outcome: &domain.DecisionOutcome{
				DecisionID:     decisionID,
				ChosenOptionID: uuid.New(),
			},
		}, nil
	}
	repo.upsertEvaluationFunc = func(ctx context.Context, eval domain.Evaluation, uID uuid.UUID) (domain.Evaluation, error) {
		return eval, nil
	}

	eval, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "Great decision", nil)
	if err != nil {
		t.Fatalf("unexpected error evaluating committed decision: %v", err)
	}
	if eval.SatisfactionAfter != 80 {
		t.Fatalf("expected satisfaction 80, got %d", eval.SatisfactionAfter)
	}
}

func TestListDecisions_UpdatedSince(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	since := time.Now().Add(-1 * time.Hour)

	calledUpdatedSince := false
	repo.listUpdatedSinceFunc = func(ctx context.Context, uID uuid.UUID, s time.Time) ([]domain.Decision, error) {
		calledUpdatedSince = true
		return []domain.Decision{{ID: uuid.New(), Title: "Updated Decision"}}, nil
	}

	res, err := svc.ListDecisions(context.Background(), userID, &since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledUpdatedSince || len(res) != 1 {
		t.Fatalf("expected ListUpdatedSince to be called, returned %d items", len(res))
	}
}

func TestSyncDecision(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	syncData := domain.SyncDecisionData{
		Title:          "Synced Title",
		LocalUpdatedAt: time.Now(),
	}

	synced, err := svc.SyncDecision(context.Background(), decisionID, userID, syncData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if synced.Title != "Synced Title" {
		t.Fatalf("expected title 'Synced Title', got '%s'", synced.Title)
	}
}

func TestCreateDecision_IdempotencyRace_ReturnsFirstDecision(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	key := "idemp-key-race"
	first := domain.Decision{ID: uuid.New(), UserID: userID, Title: "First"}

	// The key is unknown when this request checks it, then the other request
	// inserts first and this insert hits the unique index.
	inserted := false
	repo.getByIdempotencyKeyFunc = func(ctx context.Context, uID uuid.UUID, k string) (domain.Decision, error) {
		if inserted {
			return first, nil
		}
		return domain.Decision{}, domain.ErrDecisionNotFound
	}
	repo.createFunc = func(ctx context.Context, d domain.Decision, opts []domain.Option, tagIDs []uuid.UUID) (domain.Decision, error) {
		inserted = true
		return domain.Decision{}, errors.New("duplicate key value violates unique constraint")
	}

	got, err := svc.CreateDecision(context.Background(), userID, "Second", nil, nil, nil, nil, nil, nil, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != first.ID {
		t.Fatalf("expected the first decision %s, got %s", first.ID, got.ID)
	}
}

func TestCreateDecision_InsertErrorWithoutKey_IsReturned(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	insertErr := errors.New("db down")
	repo.createFunc = func(ctx context.Context, d domain.Decision, opts []domain.Option, tagIDs []uuid.UUID) (domain.Decision, error) {
		return domain.Decision{}, insertErr
	}

	_, err := svc.CreateDecision(context.Background(), uuid.New(), "Title", nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, insertErr) {
		t.Fatalf("expected the insert error, got %v", err)
	}
}

func TestCreateEvaluation_KeepsClientCreatedAt(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()
	writtenAt := time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)

	repo.getByIDFunc = func(ctx context.Context, id, uID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{
			ID:      decisionID,
			UserID:  userID,
			Outcome: &domain.DecisionOutcome{DecisionID: decisionID, ChosenOptionID: uuid.New()},
		}, nil
	}
	var saved domain.Evaluation
	repo.upsertEvaluationFunc = func(ctx context.Context, eval domain.Evaluation, uID uuid.UUID) (domain.Evaluation, error) {
		saved = eval
		return eval, nil
	}

	if _, err := svc.CreateEvaluation(context.Background(), decisionID, userID, 80, 75, "Notes", &writtenAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !saved.CreatedAt.Equal(writtenAt) {
		t.Fatalf("expected created_at %v, got %v", writtenAt, saved.CreatedAt)
	}
}

func TestAddOption_PassesUserToRepository(t *testing.T) {
	repo := &mockDecisionRepo{}
	svc := service.NewDecisionService(repo)
	userID := uuid.New()
	decisionID := uuid.New()

	repo.getByIDFunc = func(ctx context.Context, id, uID uuid.UUID) (domain.Decision, error) {
		return domain.Decision{ID: id, UserID: uID}, nil
	}
	var gotUser uuid.UUID
	repo.createOptionFunc = func(ctx context.Context, opt domain.Option, uID uuid.UUID) (domain.Option, error) {
		gotUser = uID
		return opt, nil
	}

	if _, err := svc.AddOption(context.Background(), decisionID, userID, "New option", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser != userID {
		t.Fatalf("expected user %s, got %s", userID, gotUser)
	}
}
