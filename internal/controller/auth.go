package controller

import (
	"net/http"

	request "backend/internal/dto/request"
	"backend/internal/service"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("auth-controller")

type AuthController struct {
	service service.AuthService
}

// NewAuthController wires up the routes and injects the service
func NewAuthController(e *echo.Echo, svc service.AuthService) {
	ctrl := &AuthController{service: svc}

	authGroup := e.Group("/auth")
	authGroup.POST("/register", ctrl.Register)
	authGroup.POST("/login", ctrl.Login)
}

func (ctrl *AuthController) Register(c echo.Context) error {
	// 1. Tracing Setup
	tracerCtx, span := tracer.Start(c.Request().Context(), "Controller / Auth / Register")
	var err error
	defer func() {
		// If you have your traceparent helper, call it here inside the closure
		// traceparent.SetSpanWithParent(err, span, c.Request().Context())
		span.End()
	}()

	// 2. Bind Request
	var req request.RegisterRequest
	if err = c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// 3. Validate Request
	if err = c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	userAgent := c.Request().UserAgent()
	ipAddress := c.RealIP()
	// 4. Execute Service
	res, err := ctrl.service.Register(tracerCtx, req, userAgent, ipAddress, req.Remember)
	if err != nil {
		// In a production app, you would check errors.Is() here to return 409 Conflict if email exists
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 5. Return Response
	return c.JSON(http.StatusCreated, res)
}

func (ctrl *AuthController) Login(c echo.Context) error {
	// 1. Tracing Setup
	tracerCtx, span := tracer.Start(c.Request().Context(), "Controller / Auth / Login")
	var err error
	defer func() {
		span.End()
	}()

	// 2. Bind Request
	var req request.LoginRequest
	if err = c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// 3. Validate Request
	if err = c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// 4. Extract Client Metadata for the Session Table
	userAgent := c.Request().UserAgent()
	ipAddress := c.RealIP() // Echo handles X-Forwarded-For headers automatically here

	// 5. Execute Service
	res, err := ctrl.service.Login(tracerCtx, req, userAgent, ipAddress, req.Remember)
	if err != nil {
		// Always return 401 Unauthorized for login failures, never 500 or 404
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 6. Return Response
	return c.JSON(http.StatusOK, res)
}
