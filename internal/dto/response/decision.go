package response

import (
	"time"

	"github.com/google/uuid"
)

type OptionResponse struct {
	ID         uuid.UUID `json:"id"`
	DecisionID uuid.UUID `json:"decision_id"`
	Title      string    `json:"title"`
	Pros       *string   `json:"pros,omitempty"`
	Cons       *string   `json:"cons,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type OutcomeResponse struct {
	DecisionID     uuid.UUID `json:"decision_id"`
	ChosenOptionID uuid.UUID `json:"chosen_option_id"`
	CommitmentNote *string   `json:"commitment_note,omitempty"`
	DecidedAt      time.Time `json:"decided_at"`
}

type EvaluationResponse struct {
	ID                uuid.UUID `json:"id"`
	DecisionID        uuid.UUID `json:"decision_id"`
	SatisfactionAfter int       `json:"satisfaction_after"`
	ConfidenceAfter   int       `json:"confidence_after"`
	HindsightNotes    string    `json:"hindsight_notes"`
	CreatedAt         time.Time `json:"created_at"`
}

type TagResponse struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	ColorHex string    `json:"color_hex"`
}

type DecisionResponse struct {
	ID                  uuid.UUID           `json:"id"`
	UserID              uuid.UUID           `json:"user_id"`
	Title               string              `json:"title"`
	Status              string              `json:"status"`
	Context             *string             `json:"context,omitempty"`
	ConfidenceBefore    *int                `json:"confidence_before,omitempty"`
	ScheduledReviewDate *string             `json:"scheduled_review_date,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
	DeletedAt           *time.Time          `json:"deleted_at,omitempty"`
	Options             []OptionResponse    `json:"options"`
	Outcome             *OutcomeResponse    `json:"outcome,omitempty"`
	Evaluation          *EvaluationResponse `json:"evaluation,omitempty"`
	Tags                []TagResponse       `json:"tags"`
}

type ConflictResponse struct {
	Error    string           `json:"error"`
	Decision DecisionResponse `json:"decision"`
}
