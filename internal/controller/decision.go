package controller

import (
	"net/http"

	"backend/internal/domain"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type DecisionController struct {
	service domain.DecisionService
}

func NewDecisionController(e *echo.Group, svc domain.DecisionService) {
	handler := &DecisionController{service: svc}

	e.POST("/decisions", handler.Create)
	e.GET("/decisions", handler.GetAll)
}

func (ctrl *DecisionController) Create(c echo.Context) error {
	// 1. Extract and cast the User ID from Middleware
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User context lost"})
	}

	var req struct {
		Title string `json:"title" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// 2. PASS THE userID TO THE SERVICE
	newID, err := ctrl.service.CreateDecision(c.Request().Context(), req.Title, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save to database"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Decision saved!",
		"id":      newID.String(),
	})
}

func (ctrl *DecisionController) GetAll(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User context lost"})
	}

	// PASS THE CONTEXT
	decisions, err := ctrl.service.GetDecisions(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch from database"})
	}

	return c.JSON(http.StatusOK, decisions)
}
