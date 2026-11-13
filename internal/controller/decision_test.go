package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/controller"
	"backend/internal/domain"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type mockDecisionService struct {
	createDecisionFunc   func(ctx context.Context, userID uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error)
	getDecisionFunc      func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error)
	listDecisionsFunc    func(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error)
	updateDecisionFunc   func(ctx context.Context, userID uuid.UUID, id uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, localUpdatedAt time.Time) (domain.Decision, error)
	deleteDecisionFunc   func(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error
	addOptionFunc        func(ctx context.Context, decisionID, userID uuid.UUID, title string, pros, cons *string) (domain.Option, error)
	deleteOptionFunc     func(ctx context.Context, optionID, decisionID, userID uuid.UUID) error
	commitDecisionFunc   func(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string) (domain.DecisionOutcome, error)
	createEvaluationFunc func(ctx context.Context, decisionID, userID uuid.UUID, satisfactionAfter, confidenceAfter int, hindsightNotes string) (domain.Evaluation, error)
	attachTagFunc        func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	detachTagFunc        func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
}

func (m *mockDecisionService) CreateDecision(ctx context.Context, userID uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []domain.Option, initialTagIDs []uuid.UUID) (domain.Decision, error) {
	if m.createDecisionFunc != nil {
		return m.createDecisionFunc(ctx, userID, title, contextStr, confidenceBefore, reviewDate, initialOptions, initialTagIDs)
	}
	return domain.Decision{}, nil
}

func (m *mockDecisionService) GetDecision(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	if m.getDecisionFunc != nil {
		return m.getDecisionFunc(ctx, id, userID)
	}
	return domain.Decision{}, nil
}

func (m *mockDecisionService) ListDecisions(ctx context.Context, userID uuid.UUID) ([]domain.Decision, error) {
	if m.listDecisionsFunc != nil {
		return m.listDecisionsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockDecisionService) UpdateDecision(ctx context.Context, userID uuid.UUID, id uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, localUpdatedAt time.Time) (domain.Decision, error) {
	if m.updateDecisionFunc != nil {
		return m.updateDecisionFunc(ctx, userID, id, title, contextStr, confidenceBefore, reviewDate, localUpdatedAt)
	}
	return domain.Decision{}, nil
}

func (m *mockDecisionService) DeleteDecision(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error {
	if m.deleteDecisionFunc != nil {
		return m.deleteDecisionFunc(ctx, id, userID, localUpdatedAt)
	}
	return nil
}

func (m *mockDecisionService) AddOption(ctx context.Context, decisionID, userID uuid.UUID, title string, pros, cons *string) (domain.Option, error) {
	if m.addOptionFunc != nil {
		return m.addOptionFunc(ctx, decisionID, userID, title, pros, cons)
	}
	return domain.Option{}, nil
}

func (m *mockDecisionService) DeleteOption(ctx context.Context, optionID, decisionID, userID uuid.UUID) error {
	if m.deleteOptionFunc != nil {
		return m.deleteOptionFunc(ctx, optionID, decisionID, userID)
	}
	return nil
}

func (m *mockDecisionService) CommitDecision(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string) (domain.DecisionOutcome, error) {
	if m.commitDecisionFunc != nil {
		return m.commitDecisionFunc(ctx, decisionID, userID, chosenOptionID, commitmentNote)
	}
	return domain.DecisionOutcome{}, nil
}

func (m *mockDecisionService) CreateEvaluation(ctx context.Context, decisionID, userID uuid.UUID, satisfactionAfter, confidenceAfter int, hindsightNotes string) (domain.Evaluation, error) {
	if m.createEvaluationFunc != nil {
		return m.createEvaluationFunc(ctx, decisionID, userID, satisfactionAfter, confidenceAfter, hindsightNotes)
	}
	return domain.Evaluation{}, nil
}

func (m *mockDecisionService) AttachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	if m.attachTagFunc != nil {
		return m.attachTagFunc(ctx, decisionID, tagID, userID)
	}
	return nil
}

func (m *mockDecisionService) DetachTag(ctx context.Context, decisionID, tagID, userID uuid.UUID) error {
	if m.detachTagFunc != nil {
		return m.detachTagFunc(ctx, decisionID, tagID, userID)
	}
	return nil
}

func setupEchoWithAuth(userID uuid.UUID) (*echo.Echo, *echo.Group) {
	e := echo.New()
	g := e.Group("")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", userID)
			return next(c)
		}
	})
	return e, g
}

func TestDecisionController_InvalidDateFormat(t *testing.T) {
	userID := uuid.New()
	svc := &mockDecisionService{}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{"title": "Valid Title", "scheduled_review_date": "not-a-valid-date"}`
	req := httptest.NewRequest(http.MethodPost, "/decisions", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on invalid date, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDecisionController_DeleteChosenOption_ReturnsBadRequest(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()
	optionID := uuid.New()

	svc := &mockDecisionService{
		deleteOptionFunc: func(ctx context.Context, optID, decID, uid uuid.UUID) error {
			return errors.New("cannot delete option chosen for this decision's commitment")
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/options/"+optionID.String()+"/delete", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDecisionController_CreateEvaluation_RequiresCommitment(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()

	svc := &mockDecisionService{
		createEvaluationFunc: func(ctx context.Context, decID, uid uuid.UUID, sat, conf int, notes string) (domain.Evaluation, error) {
			return domain.Evaluation{}, errors.New("decision must be committed before it can be evaluated")
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{"satisfaction_after": 80, "confidence_after": 75, "hindsight_notes": "Valid notes"}`
	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/evaluations", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDecisionController_ComputedStatusInResponse(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()

	svc := &mockDecisionService{
		getDecisionFunc: func(ctx context.Context, id, uid uuid.UUID) (domain.Decision, error) {
			return domain.Decision{
				ID:        decisionID,
				UserID:    userID,
				Title:     "Decided Decision",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Outcome: &domain.DecisionOutcome{
					DecisionID:     decisionID,
					ChosenOptionID: uuid.New(),
				},
				Evaluation: nil,
			}, nil
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	req := httptest.NewRequest(http.MethodGet, "/decisions/"+decisionID.String(), nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res response.DecisionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Status != "decided" {
		t.Fatalf("expected status 'decided', got '%s'", res.Status)
	}
}
