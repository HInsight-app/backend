package domain

import (
	"context"

	"github.com/google/uuid"
)

type Tag struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	ColorHex string    `json:"color_hex"`
}

type TagRepository interface {
	Create(ctx context.Context, tag Tag) (Tag, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (Tag, error)
	GetByName(ctx context.Context, userID uuid.UUID, name string) (Tag, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]Tag, error)
	DeleteByID(ctx context.Context, id, userID uuid.UUID) error
}

type TagService interface {
	CreateTag(ctx context.Context, userID uuid.UUID, name, colorHex string) (Tag, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]Tag, error)
	DeleteTag(ctx context.Context, id, userID uuid.UUID) error
}
