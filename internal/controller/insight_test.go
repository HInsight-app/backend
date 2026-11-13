package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/controller"
	"backend/internal/domain"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
)

type mockInsightService struct {
	generateErr error
	listErr     error
}

func (m *mockInsightService) GenerateInsight(ctx context.Context, decisionID, userID uuid.UUID, trigger domain.InsightTrigger) (domain.Insight, error) {
	if m.generateErr != nil {
		return domain.Insight{}, m.generateErr
	}
	return domain.Insight{
		ID:              uuid.New(),
		DecisionID:      decisionID,
		ModelVersion:    "test-model",
		TriggerContext:  trigger,
		InsightAnalysis: domain.InsightAnalysis{InsightText: "ok"},
	}, nil
}

func (m *mockInsightService) ListInsights(ctx context.Context, decisionID, userID uuid.UUID) ([]domain.Insight, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []domain.Insight{{ID: uuid.New(), DecisionID: decisionID}}, nil
}

func TestInsightController_Generate(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{"created", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"pre_decision"}`, nil, http.StatusCreated},
		{"bad decision id", "/decisions/not-a-uuid/insights", `{"trigger_context":"pre_decision"}`, nil, http.StatusBadRequest},
		{"bad trigger", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"someday"}`, nil, http.StatusBadRequest},
		{"missing trigger", "/decisions/" + uuid.NewString() + "/insights", `{}`, nil, http.StatusBadRequest},
		{"not found", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"pre_decision"}`, domain.ErrDecisionNotFound, http.StatusNotFound},
		{"needs evaluation", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"post_evaluation"}`, domain.ErrInsightRequiresEvaluation, http.StatusBadRequest},
		{"rate limited", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"pre_decision"}`, domain.ErrInsightRateLimited, http.StatusTooManyRequests},
		{"disabled", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"pre_decision"}`, domain.ErrInsightsDisabled, http.StatusServiceUnavailable},
		{"model failed", "/decisions/" + uuid.NewString() + "/insights", `{"trigger_context":"pre_decision"}`, domain.ErrInsightGeneration, http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, g := setupEchoWithAuth(uuid.New())
			controller.NewInsightController(g, &mockInsightService{generateErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestInsightController_List(t *testing.T) {
	e, g := setupEchoWithAuth(uuid.New())
	controller.NewInsightController(g, &mockInsightService{})

	req := httptest.NewRequest(http.MethodGet, "/decisions/"+uuid.NewString()+"/insights", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res []response.InsightResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || len(res) != 1 {
		t.Fatalf("expected one insight in response, got %s", rec.Body.String())
	}
}

func TestInsightController_List_NotFound(t *testing.T) {
	e, g := setupEchoWithAuth(uuid.New())
	controller.NewInsightController(g, &mockInsightService{listErr: domain.ErrDecisionNotFound})

	req := httptest.NewRequest(http.MethodGet, "/decisions/"+uuid.NewString()+"/insights", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
