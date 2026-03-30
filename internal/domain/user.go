package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type LocalCredential struct {
	UserID       uuid.UUID `json:"user_id"`
	PasswordHash string    `json:"-"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"-"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserRepository interface {
	CreateUserWithPassword(ctx context.Context, email, displayName, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error)
	CreateSession(ctx context.Context, session Session) error
	GetSessionByToken(ctx context.Context, tokenHash string) (Session, error)
}
