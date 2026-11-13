package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
)

type decisionService struct {
	repo domain.DecisionRepository
}

func NewDecisionService(repo domain.DecisionRepository) domain.DecisionService {
	return &decisionService{repo: repo}
}

func (s *decisionService) CreateDecision(
	ctx context.Context,
	userID uuid.UUID,
	title string,
	contextStr *string,
	confidenceBefore *int,
	reviewDate *time.Time,
	initialOptions []domain.Option,
	initialTagIDs []uuid.UUID,
	createdAt *time.Time,
	idempotencyKey *string,
) (domain.Decision, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return domain.Decision{}, errors.New("title is required")
	}

	// Idempotency check: if an idempotency key was supplied, check if already created
	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, err := s.repo.GetByIdempotencyKey(ctx, userID, *idempotencyKey)
		if err == nil {
			return existing, nil
		}
	}

	cleanOptions := make([]domain.Option, 0, len(initialOptions))
	for _, opt := range initialOptions {
		optTitle := strings.TrimSpace(opt.Title)
		if optTitle == "" {
			continue
		}
		cleanOptions = append(cleanOptions, domain.Option{
			Title: optTitle,
			Pros:  opt.Pros,
			Cons:  opt.Cons,
		})
	}

	d := domain.Decision{
		UserID:              userID,
		Title:               cleanTitle,
		Context:             contextStr,
		ConfidenceBefore:    confidenceBefore,
		ScheduledReviewDate: reviewDate,
		IdempotencyKey:      idempotencyKey,
	}
	if createdAt != nil && !createdAt.IsZero() {
		d.CreatedAt = *createdAt
	}

	created, err := s.repo.Create(ctx, d, cleanOptions, initialTagIDs)
	if err != nil && idempotencyKey != nil && *idempotencyKey != "" {
		// Two requests with the same key at once: the other one inserted first
		// and this insert hit the unique index. Return the first decision.
		if existing, getErr := s.repo.GetByIdempotencyKey(ctx, userID, *idempotencyKey); getErr == nil {
			return existing, nil
		}
	}
	return created, err
}

func (s *decisionService) GetDecision(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *decisionService) ListDecisions(ctx context.Context, userID uuid.UUID, updatedSince *time.Time) ([]domain.Decision, error) {
	if updatedSince != nil && !updatedSince.IsZero() {
		return s.repo.ListUpdatedSince(ctx, userID, *updatedSince)
	}
	return s.repo.GetActiveByUserID(ctx, userID)
}

func (s *decisionService) UpdateDecision(
	ctx context.Context,
	userID uuid.UUID,
	id uuid.UUID,
	title string,
	contextStr *string,
	confidenceBefore *int,
	reviewDate *time.Time,
	localUpdatedAt time.Time,
) (domain.Decision, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return domain.Decision{}, errors.New("title is required")
	}

	d := domain.Decision{
		ID:                  id,
		UserID:              userID,
		Title:               cleanTitle,
		Context:             contextStr,
		ConfidenceBefore:    confidenceBefore,
		ScheduledReviewDate: reviewDate,
	}

	return s.repo.UpdateWithLWW(ctx, d, localUpdatedAt)
}

func (s *decisionService) DeleteDecision(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error {
	return s.repo.SoftDeleteWithLWW(ctx, id, userID, localUpdatedAt)
}

func (s *decisionService) AddOption(ctx context.Context, decisionID, userID uuid.UUID, title string, pros, cons *string) (domain.Option, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return domain.Option{}, errors.New("option title is required")
	}

	// Verify parent decision belongs to user
	if _, err := s.repo.GetByID(ctx, decisionID, userID); err != nil {
		return domain.Option{}, err
	}

	return s.repo.CreateOption(ctx, domain.Option{
		DecisionID: decisionID,
		Title:      cleanTitle,
		Pros:       pros,
		Cons:       cons,
	}, userID)
}

func (s *decisionService) DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error {
	decision, err := s.repo.GetByID(ctx, decisionID, userID)
	if err != nil {
		return err
	}
	if decision.Outcome != nil && decision.Outcome.ChosenOptionID == optionID {
		return domain.ErrCannotDeleteChosenOption
	}
	return s.repo.DeleteOption(ctx, optionID, decisionID, userID)
}

func (s *decisionService) CommitDecision(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string, decidedAt *time.Time) (domain.DecisionOutcome, error) {
	outcome := domain.DecisionOutcome{
		DecisionID:     decisionID,
		ChosenOptionID: chosenOptionID,
		CommitmentNote: commitmentNote,
	}
	if decidedAt != nil && !decidedAt.IsZero() {
		outcome.DecidedAt = *decidedAt
	}

	return s.repo.UpsertOutcome(ctx, outcome, userID)
}

func (s *decisionService) CreateEvaluation(
	ctx context.Context,
	decisionID, userID uuid.UUID,
	satisfactionAfter, confidenceAfter int,
	hindsightNotes string,
	createdAt *time.Time,
) (domain.Evaluation, error) {
	cleanNotes := strings.TrimSpace(hindsightNotes)
	if cleanNotes == "" {
		return domain.Evaluation{}, errors.New("hindsight notes are required")
	}

	decision, err := s.repo.GetByID(ctx, decisionID, userID)
	if err != nil {
		return domain.Evaluation{}, err
	}
	if decision.Outcome == nil {
		return domain.Evaluation{}, domain.ErrEvaluationRequiresCommitment
	}

	eval := domain.Evaluation{
		DecisionID:        decisionID,
		SatisfactionAfter: satisfactionAfter,
		ConfidenceAfter:   confidenceAfter,
		HindsightNotes:    cleanNotes,
	}
	if createdAt != nil && !createdAt.IsZero() {
		eval.CreatedAt = *createdAt
	}

	return s.repo.UpsertEvaluation(ctx, eval, userID)
}

func (s *decisionService) AttachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	return s.repo.AddTag(ctx, decisionID, tagID, userID)
}

func (s *decisionService) DetachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	// Verify decision belongs to user
	if _, err := s.repo.GetByID(ctx, decisionID, userID); err != nil {
		return err
	}
	return s.repo.RemoveTag(ctx, decisionID, tagID, userID)
}

func (s *decisionService) SyncDecision(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
	cleanTitle := strings.TrimSpace(data.Title)
	if cleanTitle == "" {
		return domain.Decision{}, errors.New("title is required")
	}
	data.Title = cleanTitle

	return s.repo.SyncDecision(ctx, id, userID, data)
}
