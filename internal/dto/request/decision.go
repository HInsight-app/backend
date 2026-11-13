package request

import (
	"time"

	"github.com/google/uuid"
)

type CreateOptionRequest struct {
	Title string  `json:"title" validate:"required"`
	Pros  *string `json:"pros"`
	Cons  *string `json:"cons"`
}

type CreateDecisionRequest struct {
	Title               string                `json:"title" validate:"required"`
	Context             *string               `json:"context"`
	ConfidenceBefore    *int                  `json:"confidence_before" validate:"omitempty,min=1,max=100"`
	ScheduledReviewDate *string               `json:"scheduled_review_date"` // "yyyy-MM-dd" or RFC3339
	Options             []CreateOptionRequest `json:"options"`
	TagIDs              []uuid.UUID           `json:"tag_ids"`
}

type UpdateDecisionRequest struct {
	Title               string    `json:"title" validate:"required"`
	Context             *string   `json:"context"`
	ConfidenceBefore    *int      `json:"confidence_before" validate:"omitempty,min=1,max=100"`
	ScheduledReviewDate *string   `json:"scheduled_review_date"`
	LocalUpdatedAt      time.Time `json:"local_updated_at" validate:"required"`
}

type DeleteDecisionRequest struct {
	LocalUpdatedAt *time.Time `json:"local_updated_at"`
}

type CommitDecisionRequest struct {
	ChosenOptionID uuid.UUID `json:"chosen_option_id" validate:"required"`
	CommitmentNote *string   `json:"commitment_note"`
}

type CreateEvaluationRequest struct {
	SatisfactionAfter int    `json:"satisfaction_after" validate:"required,min=1,max=100"`
	ConfidenceAfter   int    `json:"confidence_after" validate:"required,min=1,max=100"`
	HindsightNotes    string `json:"hindsight_notes" validate:"required"`
}

type AttachTagRequest struct {
	TagID uuid.UUID `json:"tag_id" validate:"required"`
}

type CreateTagRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=100"`
	ColorHex string `json:"color_hex" validate:"required,hexcolor"`
}
