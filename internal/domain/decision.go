package domain

import (
	"context"

	"github.com/google/uuid"
)

// Decision represents the core data model.
type Decision struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Title  string    `gorm:"type:varchar(255);not null" json:"title"`
}

// DecisionRepository defines how to interact with the database.
type DecisionRepository interface {
	Create(ctx context.Context, title string, userID uuid.UUID) (uuid.UUID, error)
	GetAll(ctx context.Context, userID uuid.UUID) ([]Decision, error)
}

// DecisionService defines the business logic operations.
type DecisionService interface {
	CreateDecision(ctx context.Context, title string, userID uuid.UUID) (uuid.UUID, error)
	GetDecisions(ctx context.Context, userID uuid.UUID) ([]Decision, error)
}
