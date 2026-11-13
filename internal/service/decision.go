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
) (domain.Decision, error) {
	cleanTitle := strings.TrimSpace(title)
	if cleanTitle == "" {
		return domain.Decision{}, errors.New("title is required")
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
	}

	return s.repo.Create(ctx, d, cleanOptions, initialTagIDs)
}

func (s *decisionService) GetDecision(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *decisionService) ListDecisions(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
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
	})
}

func (s *decisionService) DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error {
	decision, err := s.repo.GetByID(ctx, decisionID, userID)
	if err != nil {
		return err
	}
	if decision.Outcome != nil && decision.Outcome.ChosenOptionID == optionID {
		return errors.New("cannot delete option chosen for this decision's commitment")
	}
	return s.repo.DeleteOption(ctx, optionID, decisionID, userID)
}

func (s *decisionService) CommitDecision(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string) (domain.DecisionOutcome, error) {
	outcome := domain.DecisionOutcome{
		DecisionID:     decisionID,
		ChosenOptionID: chosenOptionID,
		CommitmentNote: commitmentNote,
	}

	return s.repo.UpsertOutcome(ctx, outcome, userID)
}

func (s *decisionService) CreateEvaluation(
	ctx context.Context,
	decisionID, userID uuid.UUID,
	satisfactionAfter, confidenceAfter int,
	hindsightNotes string,
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
		return domain.Evaluation{}, errors.New("decision must be committed before it can be evaluated")
	}

	eval := domain.Evaluation{
		DecisionID:        decisionID,
		SatisfactionAfter: satisfactionAfter,
		ConfidenceAfter:   confidenceAfter,
		HindsightNotes:    cleanNotes,
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
