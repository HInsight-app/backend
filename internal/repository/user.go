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

type userRepository struct {
	db *pgxpool.Pool
	q  *database.Queries
}

func NewUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &userRepository{
		db: db,
		q:  database.New(db),
	}
}

func (r *userRepository) CreateUserWithPassword(ctx context.Context, email, displayName, passwordHash string) (domain.User, error) {
	var domainUser domain.User

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domainUser, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	dbUser, err := qtx.CreateUser(ctx, database.CreateUserParams{
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return domainUser, err
	}

	err = qtx.CreateLocalCredential(ctx, database.CreateLocalCredentialParams{
		UserID:       dbUser.ID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return domainUser, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domainUser, err
	}

	return domain.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		DisplayName: dbUser.DisplayName,
		CreatedAt:   dbUser.CreatedAt.Time,
	}, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	dbUser, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, errors.New("user not found")
		}
		return domain.User{}, err
	}

	return domain.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		DisplayName: dbUser.DisplayName,
		CreatedAt:   dbUser.CreatedAt.Time,
	}, nil
}

func (r *userRepository) GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	hash, err := r.q.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("credentials not found")
		}
		return "", err
	}

	return hash, nil
}

func (r *userRepository) CreateSession(ctx context.Context, session domain.Session) error {
	return r.q.CreateSession(ctx, database.CreateSessionParams{
		UserID:    session.UserID,
		Token:     session.Token,
		UserAgent: pgtype.Text{String: session.UserAgent, Valid: session.UserAgent != ""},
		IpAddress: pgtype.Text{String: session.IPAddress, Valid: session.IPAddress != ""},
		ExpiresAt: pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true},
	})
}

func (r *userRepository) GetSessionByToken(ctx context.Context, tokenHash string) (domain.Session, error) {
	row, err := r.q.GetSessionByToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, errors.New("session not found or expired")
		}
		return domain.Session{}, err
	}

	return domain.Session{
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}
