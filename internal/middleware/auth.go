package middleware

import (
	"net/http"
	"strings"

	"backend/internal/service"

	"github.com/labstack/echo/v4"
)

func AuthMiddleware(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Extract the Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			// 2. Check for "Bearer <token>" format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			// 3. Validate — hashing is handled inside ValidateSession, not here
			session, err := authService.ValidateSession(c.Request().Context(), parts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
			}

			// 4. Inject user_id into context for downstream controllers
			c.Set("user_id", session.UserID)

			return next(c)
		}
	}
}
