package repository

import (
	"backend/internal/domain"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type decisionRepository struct {
	db *gorm.DB
}

// NewDecisionRepository injects the GORM database dependency
func NewDecisionRepository(db *gorm.DB) domain.DecisionRepository {
	return &decisionRepository{db}
}

func (r *decisionRepository) Create(ctx context.Context, title string, userID uuid.UUID) (uuid.UUID, error) {
	var idStr string // Scan into a string first

	query := "INSERT INTO decisions (title, user_id) VALUES (?, ?) RETURNING id"
	err := r.db.WithContext(ctx).Raw(query, title, userID).Scan(&idStr).Error
	if err != nil {
		return uuid.Nil, err
	}

	// Parse the string back into a UUID object
	return uuid.Parse(idStr)
}

func (r *decisionRepository) GetAll(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	decisions := make([]domain.Decision, 0)
	// Filter by UserID so users only see their own stuff
	query := "SELECT id, title, user_id FROM decisions WHERE user_id = ?"
	err := r.db.WithContext(ctx).Raw(query, userID).Scan(&decisions).Error
	return decisions, err
}
