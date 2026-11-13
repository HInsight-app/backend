package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"backend/internal/repository/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type tagRepository struct {
	db *pgxpool.Pool
	q  *database.Queries
}

func NewTagRepository(db *pgxpool.Pool) domain.TagRepository {
	return &tagRepository{
		db: db,
		q:  database.New(db),
	}
}

func (r *tagRepository) Create(ctx context.Context, tag domain.Tag) (domain.Tag, error) {
	row, err := r.q.CreateTag(ctx, database.CreateTagParams{
		UserID:   tag.UserID,
		Name:     tag.Name,
		ColorHex: pgtype.Text{String: tag.ColorHex, Valid: tag.ColorHex != ""},
	})
	if err != nil {
		return domain.Tag{}, err
	}

	return domain.Tag{
		ID:       row.ID,
		UserID:   row.UserID,
		Name:     row.Name,
		ColorHex: row.ColorHex.String,
	}, nil
}

func (r *tagRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (domain.Tag, error) {
	row, err := r.q.GetTagByID(ctx, database.GetTagByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Tag{}, errors.New("tag not found")
		}
		return domain.Tag{}, err
	}

	return domain.Tag{
		ID:       row.ID,
		UserID:   row.UserID,
		Name:     row.Name,
		ColorHex: row.ColorHex.String,
	}, nil
}

func (r *tagRepository) GetByName(ctx context.Context, userID uuid.UUID, name string) (domain.Tag, error) {
	row, err := r.q.GetTagByName(ctx, database.GetTagByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Tag{}, errors.New("tag not found")
		}
		return domain.Tag{}, err
	}

	return domain.Tag{
		ID:       row.ID,
		UserID:   row.UserID,
		Name:     row.Name,
		ColorHex: row.ColorHex.String,
	}, nil
}

func (r *tagRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.q.GetTagsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	tags := make([]domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = domain.Tag{
			ID:       row.ID,
			UserID:   row.UserID,
			Name:     row.Name,
			ColorHex: row.ColorHex.String,
		}
	}
	return tags, nil
}

func (r *tagRepository) DeleteByID(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteTagByID(ctx, database.DeleteTagByIDParams{
		ID:     id,
		UserID: userID,
	})
}
