package behavior

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	appbehavior "github.com/mounis-bhat/starter/internal/app/behavior"
)

// AdaptationAdvisor wraps a Genkit flow for plan adaptation advice.
type AdaptationAdvisor struct {
	flow *core.Flow[*appbehavior.AdaptationRequest, *appbehavior.AdaptationResult, struct{}]
}

func NewAdaptationAdvisor(g *genkit.Genkit) *AdaptationAdvisor {
	flow := genkit.DefineFlow(g, "behaviorAdaptationFlow", func(ctx context.Context, input *appbehavior.AdaptationRequest) (*appbehavior.AdaptationResult, error) {
		mismatchStr := "no"
		if input.DifficultyMismatch {
			mismatchStr = "yes"
		}

		prompt := fmt.Sprintf(`You are a behavioral coaching AI. Analyze the user's adherence metrics and decide whether to adapt their plan difficulty.

User goals: %s

Current metrics:
- Current difficulty level: %.1f/10
- Completion rate: %.0f%%
- Streak count: %d days
- Total tasks assigned: %d
- Tasks completed: %d
- Difficulty mismatch detected: %s

Decide whether to adapt the difficulty level. Consider:
1. If completion rate is very high (>85%%) and streak is long, the user might need more challenge
2. If completion rate is very low (<40%%), the difficulty might be too high
3. If there's a difficulty mismatch, adjustment is likely needed
4. Small incremental changes (0.5-1.0) are better than large jumps
5. If metrics look balanced, no adaptation is needed

Respond with:
- should_adapt: true or false
- new_difficulty: the recommended difficulty level (0.0-10.0), or the current level if no change
- reason: a brief human-readable explanation of your reasoning (required if should_adapt is true)`,
			string(input.Goals),
			input.CurrentDifficulty,
			input.CompletionRate*100,
			input.StreakCount,
			input.TotalTasks,
			input.CompletedTasks,
			mismatchStr,
		)

		result, _, err := genkit.GenerateData[appbehavior.AdaptationResult](ctx, g, ai.WithPrompt(prompt))
		if err != nil {
			return nil, fmt.Errorf("generate adaptation advice: %w", err)
		}

		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("invalid adaptation advice from AI: %w", err)
		}

		return result, nil
	})

	return &AdaptationAdvisor{flow: flow}
}

func (a *AdaptationAdvisor) Advise(ctx context.Context, req appbehavior.AdaptationRequest) (*appbehavior.AdaptationResult, error) {
	return a.flow.Run(ctx, &req)
}
