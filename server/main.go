package main

import (
	"backend/internal/ai"
	"backend/internal/controller"
	"backend/internal/domain"
	"backend/internal/helpers"
	appMiddleware "backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"
	"context"
	"log"
	"os"
	"strconv"

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
	decisionRepo := repository.NewDecisionRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	insightRepo := repository.NewInsightRepository(pool)

	// 3. Dependency Injection: Services
	decisionService := service.NewDecisionService(decisionRepo)
	tagService := service.NewTagService(tagRepo)
	authService := service.NewAuthService(userRepo)
	insightService := service.NewInsightService(decisionRepo, insightRepo, newInsightGenerator(), dailyInsightLimit())

	// 4. Mount Routes
	controller.NewAuthController(e, authService)
	controller.NewHealthController(e, pool)

	protected := e.Group("")
	protected.Use(appMiddleware.AuthMiddleware(authService))

	controller.NewDecisionController(protected, decisionService)
	controller.NewTagController(protected, tagService)
	controller.NewInsightController(protected, insightService)

	// 5. Start Server
	e.Logger.Fatal(e.Start("0.0.0.0:8080"))
}

// newInsightGenerator returns the Gemini client, or nil when GEMINI_API_KEY is
// not set so the server still starts and only the insight endpoint is disabled.
func newInsightGenerator() domain.InsightGenerator {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		log.Println("⚠️  GEMINI_API_KEY not set: AI insights are disabled")
		return nil
	}
	gen, err := ai.NewGeminiGenerator(context.Background(), key, os.Getenv("GEMINI_MODEL"))
	if err != nil {
		log.Printf("⚠️  AI insights are disabled: %v", err)
		return nil
	}
	log.Printf("✅ AI insights enabled (model %s)", gen.ModelVersion())
	return gen
}

func dailyInsightLimit() int {
	n, err := strconv.Atoi(os.Getenv("AI_INSIGHT_DAILY_LIMIT"))
	if err != nil || n <= 0 {
		return service.DefaultDailyInsightLimit
	}
	return n
}
