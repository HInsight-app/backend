package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/controller"
	"backend/internal/domain"
	response "backend/internal/dto/response"
	"backend/internal/helpers"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type mockDecisionService struct {
	createDecisionFunc   func(ctx context.Context, userID uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []domain.Option, initialTagIDs []uuid.UUID, createdAt *time.Time, idempotencyKey *string) (domain.Decision, error)
	getDecisionFunc      func(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error)
	listDecisionsFunc    func(ctx context.Context, userID uuid.UUID, updatedSince *time.Time) ([]domain.Decision, error)
	updateDecisionFunc   func(ctx context.Context, userID uuid.UUID, id uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, localUpdatedAt time.Time) (domain.Decision, error)
	deleteDecisionFunc   func(ctx context.Context, id, userID uuid.UUID, localUpdatedAt *time.Time) error
	addOptionFunc        func(ctx context.Context, decisionID, userID uuid.UUID, title string, pros, cons *string) (domain.Option, error)
	deleteOptionFunc     func(ctx context.Context, optionID, decisionID, userID uuid.UUID) error
	commitDecisionFunc   func(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string, decidedAt *time.Time) (domain.DecisionOutcome, error)
	createEvaluationFunc func(ctx context.Context, decisionID, userID uuid.UUID, satisfactionAfter, confidenceAfter int, hindsightNotes string, createdAt *time.Time) (domain.Evaluation, error)
	attachTagFunc        func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	detachTagFunc        func(ctx context.Context, decisionID, tagID, userID uuid.UUID) error
	syncDecisionFunc     func(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error)
}

func (m *mockDecisionService) CreateDecision(ctx context.Context, userID uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []domain.Option, initialTagIDs []uuid.UUID, createdAt *time.Time, idempotencyKey *string) (domain.Decision, error) {
	if m.createDecisionFunc != nil {
		return m.createDecisionFunc(ctx, userID, title, contextStr, confidenceBefore, reviewDate, initialOptions, initialTagIDs, createdAt, idempotencyKey)
	}
	return domain.Decision{}, nil
}

func (m *mockDecisionService) GetDecision(ctx context.Context, id, userID uuid.UUID) (domain.Decision, error) {
	if m.getDecisionFunc != nil {
		return m.getDecisionFunc(ctx, id, userID)
	}
	return domain.Decision{}, nil
}

func (m *mockDecisionService) ListDecisions(ctx context.Context, userID uuid.UUID, updatedSince *time.Time) ([]domain.Decision, error) {
	if m.listDecisionsFunc != nil {
		return m.listDecisionsFunc(ctx, userID, updatedSince)
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

func (m *mockDecisionService) CommitDecision(ctx context.Context, decisionID, userID, chosenOptionID uuid.UUID, commitmentNote *string, decidedAt *time.Time) (domain.DecisionOutcome, error) {
	if m.commitDecisionFunc != nil {
		return m.commitDecisionFunc(ctx, decisionID, userID, chosenOptionID, commitmentNote, decidedAt)
	}
	return domain.DecisionOutcome{}, nil
}

func (m *mockDecisionService) CreateEvaluation(ctx context.Context, decisionID, userID uuid.UUID, satisfactionAfter, confidenceAfter int, hindsightNotes string, createdAt *time.Time) (domain.Evaluation, error) {
	if m.createEvaluationFunc != nil {
		return m.createEvaluationFunc(ctx, decisionID, userID, satisfactionAfter, confidenceAfter, hindsightNotes, createdAt)
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

func (m *mockDecisionService) SyncDecision(ctx context.Context, id, userID uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
	if m.syncDecisionFunc != nil {
		return m.syncDecisionFunc(ctx, id, userID, data)
	}
	return domain.Decision{ID: id, UserID: userID, Title: data.Title}, nil
}

func setupEchoWithAuth(userID uuid.UUID) (*echo.Echo, *echo.Group) {
	e := echo.New()
	e.Validator = helpers.NewValidator()
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

func TestDecisionController_DeleteChosenOption_ReturnsConflict(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()
	optionID := uuid.New()

	svc := &mockDecisionService{
		deleteOptionFunc: func(ctx context.Context, optID, decID, uid uuid.UUID) error {
			return domain.ErrCannotDeleteChosenOption
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/options/"+optionID.String()+"/delete", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDecisionController_UnknownID_ReturnsNotFound(t *testing.T) {
	userID := uuid.New()
	unknownID := uuid.New()

	svc := &mockDecisionService{
		getDecisionFunc: func(ctx context.Context, id, uid uuid.UUID) (domain.Decision, error) {
			return domain.Decision{}, domain.ErrDecisionNotFound
		},
		updateDecisionFunc: func(ctx context.Context, uid, id uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, localUpdatedAt time.Time) (domain.Decision, error) {
			return domain.Decision{}, domain.ErrDecisionNotFound
		},
		deleteDecisionFunc: func(ctx context.Context, id, uid uuid.UUID, localUpdatedAt *time.Time) error {
			return domain.ErrDecisionNotFound
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	// GET unknown ID
	req := httptest.NewRequest(http.MethodGet, "/decisions/"+unknownID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on GET unknown ID, got %d", rec.Code)
	}

	// POST update unknown ID
	payload := `{"title": "Some Title", "local_updated_at": "2026-11-13T10:00:00Z"}`
	req = httptest.NewRequest(http.MethodPost, "/decisions/"+unknownID.String()+"/update", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on update unknown ID, got %d", rec.Code)
	}

	// POST delete unknown ID
	req = httptest.NewRequest(http.MethodPost, "/decisions/"+unknownID.String()+"/delete", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on delete unknown ID, got %d", rec.Code)
	}
}

func TestDecisionController_Create_WithIdempotencyKey(t *testing.T) {
	userID := uuid.New()
	calledWithKey := false

	svc := &mockDecisionService{
		createDecisionFunc: func(ctx context.Context, uid uuid.UUID, title string, contextStr *string, confidenceBefore *int, reviewDate *time.Time, initialOptions []domain.Option, initialTagIDs []uuid.UUID, createdAt *time.Time, idempotencyKey *string) (domain.Decision, error) {
			if idempotencyKey != nil && *idempotencyKey == "my-idemp-123" {
				calledWithKey = true
			}
			return domain.Decision{ID: uuid.New(), UserID: uid, Title: title}, nil
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{"title": "Idempotent Decision"}`
	req := httptest.NewRequest(http.MethodPost, "/decisions", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Idempotency-Key", "my-idemp-123")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	if !calledWithKey {
		t.Fatal("expected Idempotency-Key header to be passed to service")
	}
}

func TestDecisionController_Sync_Success(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()

	svc := &mockDecisionService{
		syncDecisionFunc: func(ctx context.Context, id, uid uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
			return domain.Decision{
				ID:        id,
				UserID:    uid,
				Title:     data.Title,
				CreatedAt: time.Now(),
				UpdatedAt: data.LocalUpdatedAt,
			}, nil
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{
		"title": "Synced Decision",
		"local_updated_at": "2026-11-13T12:00:00Z",
		"options": [
			{"title": "Option 1", "client_key": "local-opt-1"}
		],
		"outcome": {
			"chosen_option_key": "local-opt-1",
			"decided_at": "2026-11-13T12:00:00Z"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/sync", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res response.DecisionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Title != "Synced Decision" {
		t.Fatalf("expected title 'Synced Decision', got '%s'", res.Title)
	}
}

func TestDecisionController_CreateEvaluation_RequiresCommitment(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()

	svc := &mockDecisionService{
		createEvaluationFunc: func(ctx context.Context, decID, uid uuid.UUID, sat, conf int, notes string, createdAt *time.Time) (domain.Evaluation, error) {
			return domain.Evaluation{}, domain.ErrEvaluationRequiresCommitment
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

func TestDecisionController_Sync_InvalidChosenOption_ReturnsBadRequest(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()

	svc := &mockDecisionService{
		syncDecisionFunc: func(ctx context.Context, id, uid uuid.UUID, data domain.SyncDecisionData) (domain.Decision, error) {
			return domain.Decision{}, domain.ErrInvalidChosenOption
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{
		"title": "Synced Decision",
		"local_updated_at": "2026-11-13T12:00:00Z",
		"options": [{"title": "Option 1", "client_key": "local-opt-1"}],
		"outcome": {"chosen_option_key": "no-such-key"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/sync", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDecisionController_PullIncludesServerUpdatedAtAndDeletedAt(t *testing.T) {
	userID := uuid.New()
	serverUpdatedAt := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 11, 13, 13, 0, 0, 0, time.UTC)
	var gotSince *time.Time

	svc := &mockDecisionService{
		listDecisionsFunc: func(ctx context.Context, uid uuid.UUID, updatedSince *time.Time) ([]domain.Decision, error) {
			gotSince = updatedSince
			return []domain.Decision{{
				ID:              uuid.New(),
				UserID:          uid,
				Title:           "Deleted elsewhere",
				UpdatedAt:       deletedAt,
				ServerUpdatedAt: serverUpdatedAt,
				DeletedAt:       &deletedAt,
			}}, nil
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	req := httptest.NewRequest(http.MethodGet, "/decisions?updated_since=2026-11-13T12:00:00.000Z", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotSince == nil || !gotSince.Equal(time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected updated_since 2026-11-13T12:00:00Z (with milliseconds accepted), got %v", gotSince)
	}

	var res []response.DecisionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res) != 1 || !res[0].ServerUpdatedAt.Equal(serverUpdatedAt) {
		t.Fatalf("expected server_updated_at %v, got %+v", serverUpdatedAt, res)
	}
	if res[0].DeletedAt == nil || !res[0].DeletedAt.Equal(deletedAt) {
		t.Fatalf("expected deleted_at %v, got %v", deletedAt, res[0].DeletedAt)
	}
}

func TestDecisionController_CreateEvaluation_PassesClientCreatedAt(t *testing.T) {
	userID := uuid.New()
	decisionID := uuid.New()
	var gotCreatedAt *time.Time

	svc := &mockDecisionService{
		createEvaluationFunc: func(ctx context.Context, decID, uid uuid.UUID, sat, conf int, notes string, createdAt *time.Time) (domain.Evaluation, error) {
			gotCreatedAt = createdAt
			return domain.Evaluation{ID: uuid.New(), DecisionID: decID}, nil
		},
	}
	e, g := setupEchoWithAuth(userID)
	controller.NewDecisionController(g, svc)

	payload := `{"satisfaction_after": 80, "confidence_after": 75, "hindsight_notes": "Valid notes", "created_at": "2026-11-10T08:30:00.000Z"}`
	req := httptest.NewRequest(http.MethodPost, "/decisions/"+decisionID.String()+"/evaluations", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotCreatedAt == nil || !gotCreatedAt.Equal(time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)) {
		t.Fatalf("expected created_at 2026-11-10T08:30:00Z, got %v", gotCreatedAt)
	}
}
