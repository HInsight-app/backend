package main

import (
	"backend/internal/controller"
	"backend/internal/helpers"
	appMiddleware "backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Setup Infrastructure
	pool, dsn := helpers.InitDB()
	defer pool.Close()

	if err := helpers.RunMigrations(dsn); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	e := echo.New()
	e.Validator = helpers.NewValidator()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// 2. Dependency Injection: Repositories
	// decisionRepo := repository.NewDecisionRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	// 3. Dependency Injection: Services
	// decisionService := service.NewDecisionService(decisionRepo)
	authService := service.NewAuthService(userRepo)

	// 4. Mount Routes
	controller.NewAuthController(e, authService)
	controller.NewHealthController(e, pool)

	protected := e.Group("")
	protected.Use(appMiddleware.AuthMiddleware(authService))

	// controller.NewDecisionController(protected, decisionService)

	// 5. Start Server
	e.Logger.Fatal(e.Start("0.0.0.0:8080"))
}
