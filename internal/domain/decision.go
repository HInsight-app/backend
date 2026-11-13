package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDecisionNotFound             = errors.New("decision not found")
	ErrCannotDeleteChosenOption     = errors.New("cannot delete chosen option")
	ErrAlreadyDeleted               = errors.New("decision already deleted")
	ErrEvaluationRequiresCommitment = errors.New("cannot evaluate an uncommitted decision")
	ErrInvalidChosenOption          = errors.New("chosen option is not one of this decision's options")
)

// ErrConflict is returned when an update/delete is attempted against an older version (LWW).
type ErrConflict struct {
	ServerDecision Decision
}

func (e *ErrConflict) Error() string {
	return "conflict: server has newer version"
}

// Option represents a decision option.
type Option struct {
	ID         uuid.UUID `json:"id"`
	DecisionID uuid.UUID `json:"decision_id"`
	Title      string    `json:"title"`
	Pros       *string   `json:"pros,omitempty"`
	Cons       *string   `json:"cons,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// DecisionOutcome represents a chosen option and commitment note.
type DecisionOutcome struct {
	DecisionID     uuid.UUID `json:"decision_id"`
	ChosenOptionID uuid.UUID `json:"chosen_option_id"`
	CommitmentNote *string   `json:"commitment_note,omitempty"`
	DecidedAt      time.Time `json:"decided_at"`
}

// Evaluation represents hindsight rating & notes.
type Evaluation struct {
	ID                uuid.UUID `json:"id"`
	DecisionID        uuid.UUID `json:"decision_id"`
	SatisfactionAfter int       `json:"satisfaction_after"`
	ConfidenceAfter   int       `json:"confidence_after"`
	HindsightNotes    string    `json:"hindsight_notes"`
	CreatedAt         time.Time `json:"created_at"`
}

// Decision represents the aggregate core model.
type Decision struct {
	ID                  uuid.UUID        `json:"id"`
	UserID              uuid.UUID        `json:"user_id"`
	Title               string           `json:"title"`
	Context             *string          `json:"context,omitempty"`
	ConfidenceBefore    *int             `json:"confidence_before,omitempty"`
	ScheduledReviewDate *time.Time       `json:"scheduled_review_date,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	ServerUpdatedAt     time.Time        `json:"server_updated_at"`
	DeletedAt           *time.Time       `json:"deleted_at,omitempty"`
	IdempotencyKey      *string          `json:"idempotency_key,omitempty"`
	Options             []Option         `json:"options"`
	Outcome             *DecisionOutcome `json:"outcome,omitempty"`
	Evaluation          *Evaluation      `json:"evaluation,omitempty"`
	Tags                []Tag            `json:"tags"`
}

// SyncOption represents an option item in the aggregate sync payload.
type SyncOption struct {
	ID        *uuid.UUID
	ClientKey *string
	Title     string
	Pros      *string
	Cons      *string
}

// SyncOutcome represents an outcome item in the aggregate sync payload.
type SyncOutcome struct {
	ChosenOptionID  *uuid.UUID
	ChosenOptionKey *string
	CommitmentNote  *string
	DecidedAt       *time.Time
}

// SyncDecisionData represents the aggregate whole-decision sync input.
type SyncDecisionData struct {
	Title               string
	Context             *string
	ConfidenceBefore    *int
	ScheduledReviewDate *time.Time
	CreatedAt           *time.Time
	LocalUpdatedAt      time.Time
	TagIDs              []uuid.UUID
	Options             []SyncOption
	Outcome             *SyncOutcome
}

// DecisionRepository defines data operations on decisions and related entities.
type DecisionRepository interface {
	Create(ctx context.Context, d Decision, initialOptions []Option, initialTagIDs []uuid.UUID) (Decision, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (Decision, error)
	GetByIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) (Decision, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]Decision, error)
	ListUpdatedSince(ctx context.Context, userID uuid.UUID, updatedSince time.Time) ([]Decision, error)
	UpdateWithLWW(ctx context.Context, d Decision, localUpdatedAt time.Time) (Decision, error)
	SoftDeleteWithLWW(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error
	SyncDecision(ctx context.Context, id, userID uuid.UUID, data SyncDecisionData) (Decision, error)

	CreateOption(ctx context.Context, opt Option, userID uuid.UUID) (Option, error)
	DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error
	GetOptionByID(ctx context.Context, optionID, decisionID uuid.UUID) (Option, error)

	UpsertOutcome(ctx context.Context, outcome DecisionOutcome, userID uuid.UUID) (DecisionOutcome, error)
	UpsertEvaluation(ctx context.Context, eval Evaluation, userID uuid.UUID) (Evaluation, error)

	AddTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	RemoveTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
}

// DecisionService defines business logic operations.
type DecisionService interface {
	CreateDecision(ctx context.Context, userID uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []Option, initialTagIDs []uuid.UUID, createdAt *time.Time, idempotencyKey *string) (Decision, error)
	GetDecision(ctx context.Context, id, userID uuid.UUID) (Decision, error)
	ListDecisions(ctx context.Context, userID uuid.UUID, updatedSince *time.Time) ([]Decision, error)
	UpdateDecision(ctx context.Context, userID uuid.UUID, id uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, localUpdatedAt time.Time) (Decision, error)
	DeleteDecision(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error

	AddOption(ctx context.Context, decisionID, userID uuid.UUID, title string, pros, cons *string) (Option, error)
	DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error

	CommitDecision(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string, decidedAt *time.Time) (DecisionOutcome, error)
	CreateEvaluation(ctx context.Context, decisionID, userID uuid.UUID, satisfactionAfter, confidenceAfter int, hindsightNotes string, createdAt *time.Time) (Evaluation, error)

	AttachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	DetachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error

	SyncDecision(ctx context.Context, id, userID uuid.UUID, data SyncDecisionData) (Decision, error)
}
