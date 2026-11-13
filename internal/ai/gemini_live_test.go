//go:build live

// Live check against the real Gemini API. It is excluded from normal test runs
// and costs one API call. Run it with:
//
//	GEMINI_API_KEY=... go test -tags live -run Live -v ./internal/ai
package ai_test

import (
	"context"
	"os"
	"testing"

	"backend/internal/ai"
	"backend/internal/domain"

	"github.com/google/uuid"
)

func TestLiveGeminiAnalyze(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	gen, err := ai.NewGeminiGenerator(context.Background(), key, os.Getenv("GEMINI_MODEL"))
	if err != nil {
		t.Fatalf("create generator: %v", err)
	}

	confidence := 92
	d := domain.Decision{
		ID:               uuid.New(),
		Title:            "Should I keep funding my struggling side project?",
		Context:          strPtr("I have spent two years and most of my savings on it. Users are flat, but quitting now would waste everything I put in."),
		ConfidenceBefore: &confidence,
		Options: []domain.Option{
			{ID: uuid.New(), Title: "Keep going for another year", Pros: strPtr("All my work is not wasted"), Cons: strPtr("More savings gone")},
			{ID: uuid.New(), Title: "Stop and get a job", Pros: strPtr("Stable income"), Cons: strPtr("Feels like failure")},
		},
	}

	a, err := gen.Analyze(context.Background(), d, domain.TriggerPreDecision)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if a.InsightText == "" {
		t.Fatal("empty insight text")
	}

	deref := func(s *string) string {
		if s == nil {
			return "<nil>"
		}
		return *s
	}
	t.Logf("model=%s bias=%s tone=%s risk=%s actionability=%v confidence=%v",
		gen.ModelVersion(), deref(a.PrimaryBias), deref(a.EmotionalTone), deref(a.RiskLevel),
		a.ActionabilityScore != nil, a.ModelConfidence != nil)
	t.Logf("insight_text (%d chars): %.300s", len(a.InsightText), a.InsightText)
}
