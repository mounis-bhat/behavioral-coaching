package behavior

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	appbehavior "github.com/mounis-bhat/starter/internal/app/behavior"
)

// PlanningPromptVersion tracks the current version of the planning prompt.
const PlanningPromptVersion = "v2"

// PlanningAgent wraps a Genkit flow for daily plan generation.
type PlanningAgent struct {
	flow *core.Flow[*appbehavior.PlanRequest, *appbehavior.PlanResult, struct{}]
}

// goalsData extracts structured fields from the goals JSON.
type goalsData struct {
	Categories []string `json:"categories"`
	FreeText   string   `json:"free_text"`
}

// constraintsData extracts structured fields from the constraints JSON.
type constraintsData struct {
	TimeAvailability string `json:"time_availability"`
	Limitations      string `json:"limitations"`
}

// psychData extracts structured fields from the psychological state JSON.
type psychData struct {
	Motivation int `json:"motivation"`
	Stress     int `json:"stress"`
	Energy     int `json:"energy"`
}

func taskCountRange(timeAvail string, numCategories int) (int, int) {
	// Base range on time availability
	var minT, maxT int
	switch timeAvail {
	case "30min":
		minT, maxT = 2, 3
	case "1hr":
		minT, maxT = 3, 5
	case "2hr":
		minT, maxT = 4, 6
	default: // 3hr+
		minT, maxT = 5, 7
	}
	// Don't generate more tasks than makes sense for the selected categories
	if maxT > numCategories*2 {
		maxT = numCategories * 2
	}
	if maxT < minT {
		maxT = minT
	}
	return minT, maxT
}

func NewPlanningAgent(g *genkit.Genkit) *PlanningAgent {
	flow := genkit.DefineFlow(g, "behaviorPlanningFlow", func(ctx context.Context, input *appbehavior.PlanRequest) (*appbehavior.PlanResult, error) {
		// Parse structured data from JSON fields
		var goals goalsData
		_ = json.Unmarshal(input.Goals, &goals)

		var constraints constraintsData
		_ = json.Unmarshal(input.Constraints, &constraints)

		var psych psychData
		_ = json.Unmarshal(input.PsychologicalState, &psych)

		// Build the allowed categories list
		allowedCategories := goals.Categories
		if len(allowedCategories) == 0 {
			allowedCategories = []string{"health", "mindset", "discipline", "relationships", "productivity"}
		}
		categoriesList := `"` + strings.Join(allowedCategories, `", "`) + `"`

		// Determine task count range
		minTasks, maxTasks := taskCountRange(constraints.TimeAvailability, len(allowedCategories))

		// Build optional context sections
		var goalContext string
		if goals.FreeText != "" {
			goalContext = fmt.Sprintf("\n- Specific goals: %s", goals.FreeText)
		}

		var limitationContext string
		if constraints.Limitations != "" {
			limitationContext = fmt.Sprintf("\n- Physical/schedule limitations: %s", constraints.Limitations)
		}

		prompt := fmt.Sprintf(`You are a behavioral coaching AI generating a realistic daily plan.

User profile:
- Focus areas: %s%s
- Time available today: %s%s
- Self-assessment: motivation %d/10, stress %d/10, energy %d/10
- Difficulty level: %.1f/10
- Recent completion rate: %.0f%%
- Current streak: %d days

Generate %d to %d tasks following these rules:

CATEGORIES: ONLY use categories from this list: [%s]. Do NOT generate tasks outside these categories.

DIFFICULTY CALIBRATION (%.1f/10):
- 1-2: Tiny habits (drink a glass of water, 5-minute walk, write one sentence)
- 3-4: Light tasks (20-minute walk, read for 15 minutes, journal 3 things)
- 5-6: Moderate tasks (30-minute workout, 1-hour focused work block, meditate 15 minutes)
- 7-8: Challenging tasks (45-minute run, 2-hour deep work session, difficult conversation)
- 9-10: Intense tasks (1-hour intense workout, 3-hour deep work, major project milestone)

Tasks should be concrete, time-bound where appropriate, and completable today. Scale task duration and effort to match the difficulty level above.

TIME CONSTRAINT: The user has %s available. All tasks combined must fit within this window.

COMPLETION RATE ADJUSTMENT: The user's recent completion rate is %.0f%%. If below 50%%, make tasks easier than the difficulty level suggests to rebuild momentum. If above 80%%, you may push slightly harder.

Each task must have:
- title: concise action-oriented title (e.g. "Walk for 20 minutes", not "Implement a comprehensive walking routine")
- description: 1-2 sentences explaining what to do
- category: exactly one of [%s]
- difficulty: a number from 1.0 to 10.0 matching the task's actual challenge level`,
			categoriesList,
			goalContext,
			constraints.TimeAvailability,
			limitationContext,
			psych.Motivation, psych.Stress, psych.Energy,
			input.DifficultyLevel,
			input.CompletionRate*100,
			input.StreakCount,
			minTasks, maxTasks,
			categoriesList,
			input.DifficultyLevel,
			constraints.TimeAvailability,
			input.CompletionRate*100,
			categoriesList,
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

func (a *PlanningAgent) PromptVersion() string { return PlanningPromptVersion }
