package controller

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type HealthController struct {
	db *pgxpool.Pool
}

func NewHealthController(e *echo.Echo, db *pgxpool.Pool) {
	handler := &HealthController{db: db}
	e.GET("/health", handler.Check)
}

func (ctrl *HealthController) Check(c echo.Context) error {
	if err := ctrl.db.Ping(context.Background()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"message": "All systems operational",
	})
}
