package service

import (
	"context"

	"backend/internal/domain"

	"github.com/google/uuid"
)

type decisionService struct {
	repo domain.DecisionRepository
}

func NewDecisionService(repo domain.DecisionRepository) domain.DecisionService {
	return &decisionService{repo}
}

func (s *decisionService) CreateDecision(ctx context.Context, title string, userID uuid.UUID) (uuid.UUID, error) {
	return s.repo.Create(ctx, title, userID)
}

func (s *decisionService) GetDecisions(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	return s.repo.GetAll(ctx, userID)
}
