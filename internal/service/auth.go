package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"backend/internal/domain"
	request "backend/internal/dto/request"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// dummyHash is used for constant-time comparison when a user is not found,
// preventing email enumeration via timing attacks.
const dummyHash = "$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

type AuthService interface {
	Register(ctx context.Context, req request.RegisterRequest, userAgent, ipAddress string, remember bool) (response.AuthResponse, error)
	Login(ctx context.Context, req request.LoginRequest, userAgent, ipAddress string, remember bool) (response.AuthResponse, error)
	ValidateSession(ctx context.Context, rawToken string) (domain.Session, error)
}

type authService struct {
	repo domain.UserRepository
}

func NewAuthService(repo domain.UserRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Register(ctx context.Context, req request.RegisterRequest, userAgent, ipAddress string, remember bool) (response.AuthResponse, error) {
	// 1. Hash the password before opening any DB connection
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.AuthResponse{}, errors.New("failed to process password")
	}

	// 2. Create the user via repository
	user, err := s.repo.CreateUserWithPassword(ctx, req.Email, req.DisplayName, string(hashedBytes))
	if err != nil {
		// FIX: wrap the real error so callers can inspect it (e.g. check for
		// pq error 23505 unique violation and return 409 instead of 500),
		// while still surfacing a generic message up the call chain.
		return response.AuthResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	// 3. Generate session token and persist
	rawToken, err := s.createSession(ctx, user.ID, userAgent, ipAddress, remember)
	if err != nil {
		return response.AuthResponse{}, err
	}

	return response.AuthResponse{
		User: response.UserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
		},
		Token: rawToken, 
	}, nil
}

func (s *authService) Login(ctx context.Context, req request.LoginRequest, userAgent, ipAddress string, remember bool) (response.AuthResponse, error) {
	// 1. Fetch user — keep error generic to avoid exposing whether the email exists
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// run a dummy bcrypt comparison so the response time is identical
		// whether the email exists or not, preventing timing-based enumeration.
		bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(req.Password))
		return response.AuthResponse{}, errors.New("invalid credentials")
	}

	// 2. Fetch stored password hash
	hash, err := s.repo.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return response.AuthResponse{}, errors.New("invalid credentials")
	}

	// 3. Compare — bcrypt handles timing safety internally
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		return response.AuthResponse{}, errors.New("invalid credentials")
	}

	// 4. Generate session token and persist
	rawToken, err := s.createSession(ctx, user.ID, userAgent, ipAddress, remember)
	if err != nil {
		return response.AuthResponse{}, err
	}

	return response.AuthResponse{
		User: response.UserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
		},
		Token: rawToken,
	}, nil
}

// ValidateSession hashes the raw token internally before querying, so callers
// never need to know about the hashing scheme and can't accidentally pass the
// wrong value.
func (s *authService) ValidateSession(ctx context.Context, rawToken string) (domain.Session, error) {
	session, err := s.repo.GetSessionByToken(ctx, hashToken(rawToken))
	if err != nil {
		return domain.Session{}, errors.New("invalid or expired session")
	}
	return session, nil
}

// createSession generates a secure token, persists its hash, and returns the
// raw token to send to the client. Extracted to avoid duplication between
// Register and Login.
func (s *authService) createSession(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string, remember bool) (string, error) {
	rawToken, err := generateSecureToken()
	if err != nil {
		return "", err
	}

	duration := 24 * time.Hour
	if remember {
		duration = 7 * 24 * time.Hour
	}
	session := domain.Session{
		UserID:    userID,
		Token:     hashToken(rawToken),
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: time.Now().UTC().Add(duration),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return rawToken, nil
}

// --- Private Security Helpers ---

// generateSecureToken creates 32 bytes of cryptographically secure randomness.
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("failed to generate secure token")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashToken converts the raw token into a SHA-256 hash for database storage.
// The raw token is sent to the client; only this hash is ever persisted.
func hashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
