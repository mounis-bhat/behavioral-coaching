package behavior

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	appbehavior "github.com/mounis-bhat/starter/internal/app/behavior"
)

// PlanningAgent wraps a Genkit flow for daily plan generation.
type PlanningAgent struct {
	flow *core.Flow[*appbehavior.PlanRequest, *appbehavior.PlanResult, struct{}]
}

func NewPlanningAgent(g *genkit.Genkit) *PlanningAgent {
	flow := genkit.DefineFlow(g, "behaviorPlanningFlow", func(ctx context.Context, input *appbehavior.PlanRequest) (*appbehavior.PlanResult, error) {
		prompt := fmt.Sprintf(`You are a behavioral coaching AI. Generate a personalized daily plan of tasks for someone working on self-improvement.

User profile:
- Goals: %s
- Constraints: %s
- Psychological state: %s
- Current difficulty level: %.1f/10
- Recent completion rate: %.0f%%
- Current streak: %d days

Generate 3-7 tasks that:
1. Are realistic and achievable for today
2. Cover a mix of categories: health, mindset, discipline, relationships, productivity
3. Match the difficulty level (%.1f/10) — scale task challenge accordingly
4. Account for the user's completion rate — if low, lean toward easier wins; if high, push slightly harder
5. Respect the user's constraints

Each task must have:
- title: concise action-oriented title
- description: 1-2 sentence explanation of what to do
- category: exactly one of "health", "mindset", "discipline", "relationships", "productivity"
- difficulty: a number from 1.0 to 10.0`,
			string(input.Goals),
			string(input.Constraints),
			string(input.PsychologicalState),
			input.DifficultyLevel,
			input.CompletionRate*100,
			input.StreakCount,
			input.DifficultyLevel,
		)

		result, _, err := genkit.GenerateData[appbehavior.PlanResult](ctx, g, ai.WithPrompt(prompt))
		if err != nil {
			return nil, fmt.Errorf("generate plan: %w", err)
		}

		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("invalid plan from AI: %w", err)
		}

		return result, nil
	})

	return &PlanningAgent{flow: flow}
}

func (a *PlanningAgent) GeneratePlan(ctx context.Context, req appbehavior.PlanRequest) (*appbehavior.PlanResult, error) {
	return a.flow.Run(ctx, &req)
}
