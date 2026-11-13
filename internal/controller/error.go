package controller

import (
	"github.com/labstack/echo/v4"
)

const (
	errInvalidPayload = "Invalid request payload"
	errUserContextLost = "User context lost"
)

func errorResponse(c echo.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}
