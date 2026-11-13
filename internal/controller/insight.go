package controller

import (
	"errors"
	"net/http"

	"backend/internal/domain"
	request "backend/internal/dto/request"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type InsightController struct {
	service domain.InsightService
}

func NewInsightController(e *echo.Group, svc domain.InsightService) {
	ctrl := &InsightController{service: svc}

	e.POST("/decisions/:id/insights", ctrl.Generate)
	e.GET("/decisions/:id/insights", ctrl.List)
}

func (ctrl *InsightController) Generate(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "Invalid decision ID")
	}

	var req request.GenerateInsightRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, domain.ErrInvalidInsightTrigger.Error())
	}

	insight, err := ctrl.service.GenerateInsight(c.Request().Context(), decisionID, userID, domain.InsightTrigger(req.TriggerContext))
	if err != nil {
		return insightErrorResponse(c, err)
	}

	return c.JSON(http.StatusCreated, toInsightResponse(insight))
}

func (ctrl *InsightController) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "Invalid decision ID")
	}

	insights, err := ctrl.service.ListInsights(c.Request().Context(), decisionID, userID)
	if err != nil {
		return insightErrorResponse(c, err)
	}

	res := make([]response.InsightResponse, len(insights))
	for i, in := range insights {
		res[i] = toInsightResponse(in)
	}
	return c.JSON(http.StatusOK, res)
}

func insightErrorResponse(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrDecisionNotFound):
		return errorResponse(c, http.StatusNotFound, "Decision not found")
	case errors.Is(err, domain.ErrInvalidInsightTrigger):
		return errorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInsightRequiresEvaluation):
		return errorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInsightRateLimited):
		return errorResponse(c, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, domain.ErrInsightsDisabled):
		return errorResponse(c, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, domain.ErrInsightGeneration):
		return errorResponse(c, http.StatusBadGateway, err.Error())
	default:
		return errorResponse(c, http.StatusInternalServerError, "Failed to process insight request")
	}
}

func toInsightResponse(in domain.Insight) response.InsightResponse {
	return response.InsightResponse{
		ID:                   in.ID,
		DecisionID:           in.DecisionID,
		ModelVersion:         in.ModelVersion,
		TriggerContext:       string(in.TriggerContext),
		InsightText:          in.InsightText,
		PrimaryBias:          in.PrimaryBias,
		EmotionalTone:        in.EmotionalTone,
		RiskLevel:            in.RiskLevel,
		ActionabilityScore:   in.ActionabilityScore,
		SuggestedAlternative: in.SuggestedAlternative,
		ModelConfidence:      in.ModelConfidence,
		GeneratedAt:          in.GeneratedAt,
	}
}
