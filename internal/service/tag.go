package service

import (
	"context"
	"strings"

	"backend/internal/domain"

	"github.com/google/uuid"
)

type tagService struct {
	repo domain.TagRepository
}

func NewTagService(repo domain.TagRepository) domain.TagService {
	return &tagService{repo: repo}
}

func (s *tagService) CreateTag(ctx context.Context, userID uuid.UUID, name, colorHex string) (domain.Tag, error) {
	tag := domain.Tag{
		UserID:   userID,
		Name:     strings.TrimSpace(name),
		ColorHex: strings.TrimSpace(colorHex),
	}

	created, err := s.repo.Create(ctx, tag)
	if err != nil {
		// Idempotency: if already exists, return existing tag
		if existing, getErr := s.repo.GetByName(ctx, userID, tag.Name); getErr == nil {
			return existing, nil
		}
		return domain.Tag{}, err
	}
	return created, nil
}

func (s *tagService) ListTags(ctx context.Context, userID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *tagService) DeleteTag(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteByID(ctx, id, userID)
}
