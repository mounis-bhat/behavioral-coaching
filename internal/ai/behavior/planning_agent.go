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
const PlanningPromptVersion = "v3"

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

		// Build optional context sections
		var goalContext string
		if goals.FreeText != "" {
			goalContext = fmt.Sprintf("\n- Specific goals: %s", goals.FreeText)
		}

		var limitationContext string
		if constraints.Limitations != "" {
			limitationContext = fmt.Sprintf("\n- Physical/schedule limitations: %s", constraints.Limitations)
		}

		// Build AI profile context if available
		var profileContext string
		if len(input.AIProfileSummary) > 0 && string(input.AIProfileSummary) != "{}" {
			profileContext = fmt.Sprintf("\n\nAI PROFILE INSIGHTS (from onboarding conversation):\n%s", string(input.AIProfileSummary))
		}

		// Build emotional state context
		var emotionalContext string
		if input.EmotionalState != "" {
			emotionalContext = fmt.Sprintf("\n- Emotional state: %s", input.EmotionalState)
		}

		var prompt string

		switch input.DeliveryMode {
		case "gentle_structure":
			prompt = buildGentleStructurePrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext, constraints, psych, input)
		case "strict_schedule":
			prompt = buildStrictSchedulePrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext, constraints, psych, input)
		default: // "flexible_list" or empty
			prompt = buildFlexibleListPrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext, constraints, psych, input)
		}

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

func buildFlexibleListPrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext string, constraints constraintsData, psych psychData, input *appbehavior.PlanRequest) string {
	minTasks, maxTasks := taskCountRange(constraints.TimeAvailability, len(strings.Split(categoriesList, ",")))

	return fmt.Sprintf(`You are a behavioral coaching AI generating a realistic daily plan.

User profile:
- Focus areas: %s%s
- Time available today: %s%s%s
- Self-assessment: motivation %d/10, stress %d/10, energy %d/10
- Difficulty level: %.1f/10
- Recent completion rate: %.0f%%
- Current streak: %d days%s

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
- difficulty: a number from 1.0 to 10.0 matching the task's actual challenge level
- task_type: always "regular"
- suggested_time: always ""
- anchor_label: always ""`,
		categoriesList,
		goalContext,
		constraints.TimeAvailability,
		limitationContext,
		emotionalContext,
		psych.Motivation, psych.Stress, psych.Energy,
		input.DifficultyLevel,
		input.CompletionRate*100,
		input.StreakCount,
		profileContext,
		minTasks, maxTasks,
		categoriesList,
		input.DifficultyLevel,
		constraints.TimeAvailability,
		input.CompletionRate*100,
		categoriesList,
	)
}

func buildGentleStructurePrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext string, constraints constraintsData, psych psychData, input *appbehavior.PlanRequest) string {
	return fmt.Sprintf(`You are a warm, supportive behavioral coaching AI generating a gentle daily plan for someone who needs low-pressure structure. This person may be burned out, anxious, or stuck.

User profile:
- Focus areas: %s%s
- Time available today: %s%s%s
- Self-assessment: motivation %d/10, stress %d/10, energy %d/10
- Difficulty level: %.1f/10
- Recent completion rate: %.0f%%
- Current streak: %d days%s

DELIVERY MODE: GENTLE STRUCTURE (behavioral activation style)

Generate EXACTLY 3 anchor tasks plus 2-4 tiny regular tasks (5-7 total).

ANCHOR TASKS (task_type: "anchor"):
You MUST generate exactly 3 anchor tasks with these specific anchor_labels:
1. anchor_label: "wake_window" — A gentle morning routine task (e.g., "Open the curtains and drink a glass of water", "Step outside for 2 minutes of fresh air")
2. anchor_label: "activation_activity" — One meaningful but low-effort activity in the middle of the day (e.g., "Take a 10-minute walk around the block", "Tidy one small area for 5 minutes")
3. anchor_label: "sleep_winddown" — An evening wind-down task (e.g., "Put your phone in another room 30 minutes before bed", "Do 5 minutes of gentle stretching")

TINY REGULAR TASKS (task_type: "regular", anchor_label: ""):
Generate 2-4 tiny tasks that are:
- Very low difficulty (1.0-2.5)
- Under 10 minutes each
- Focused on nervous system regulation, NOT productivity
- Warm, non-demanding titles (e.g., "Drink a warm cup of tea mindfully" not "Execute hydration protocol")

IMPORTANT RULES:
- All difficulty values for this mode should be between 1.0 and 2.5
- Titles should feel gentle and inviting, not commanding
- Descriptions should be encouraging and warm
- Focus on basic self-care, not achievement
- suggested_time: always ""

CATEGORIES: ONLY use categories from this list: [%s]. Do NOT generate tasks outside these categories.

Each task must have:
- title: warm, inviting, non-demanding title
- description: 1-2 encouraging sentences
- category: exactly one of [%s]
- difficulty: 1.0 to 2.5
- task_type: "anchor" or "regular"
- suggested_time: ""
- anchor_label: one of "wake_window", "activation_activity", "sleep_winddown" for anchors, or "" for regular tasks`,
		categoriesList,
		goalContext,
		constraints.TimeAvailability,
		limitationContext,
		emotionalContext,
		psych.Motivation, psych.Stress, psych.Energy,
		input.DifficultyLevel,
		input.CompletionRate*100,
		input.StreakCount,
		profileContext,
		categoriesList,
		categoriesList,
	)
}

func buildStrictSchedulePrompt(categoriesList, goalContext, limitationContext, profileContext, emotionalContext string, constraints constraintsData, psych psychData, input *appbehavior.PlanRequest) string {
	minTasks, maxTasks := taskCountRange(constraints.TimeAvailability, len(strings.Split(categoriesList, ",")))

	return fmt.Sprintf(`You are a behavioral coaching AI generating a structured daily plan with specific time slots.

User profile:
- Focus areas: %s%s
- Time available today: %s%s%s
- Self-assessment: motivation %d/10, stress %d/10, energy %d/10
- Difficulty level: %.1f/10
- Recent completion rate: %.0f%%
- Current streak: %d days%s

DELIVERY MODE: STRICT SCHEDULE

Generate %d to %d tasks with specific time windows.

SCHEDULING RULES:
- Each task MUST have a suggested_time in "HH:MM-HH:MM" 24-hour format (e.g., "07:00-07:30", "14:00-15:00")
- Time windows must NOT overlap
- Leave at least 15 minutes buffer between tasks
- Start no earlier than 06:00 and end no later than 23:00
- Schedule tasks in a logical daily flow (morning routine → work → evening)
- All tasks should have task_type: "regular" and anchor_label: ""

CATEGORIES: ONLY use categories from this list: [%s]. Do NOT generate tasks outside these categories.

DIFFICULTY CALIBRATION (%.1f/10):
- 1-2: Tiny habits (drink a glass of water, 5-minute walk, write one sentence)
- 3-4: Light tasks (20-minute walk, read for 15 minutes, journal 3 things)
- 5-6: Moderate tasks (30-minute workout, 1-hour focused work block, meditate 15 minutes)
- 7-8: Challenging tasks (45-minute run, 2-hour deep work session, difficult conversation)
- 9-10: Intense tasks (1-hour intense workout, 3-hour deep work, major project milestone)

TIME CONSTRAINT: The user has %s available. All tasks combined must fit within this window.

COMPLETION RATE ADJUSTMENT: The user's recent completion rate is %.0f%%. If below 50%%, make tasks easier. If above 80%%, push slightly harder.

Each task must have:
- title: concise action-oriented title
- description: 1-2 sentences explaining what to do
- category: exactly one of [%s]
- difficulty: a number from 1.0 to 10.0
- task_type: "regular"
- suggested_time: time window in "HH:MM-HH:MM" format
- anchor_label: ""`,
		categoriesList,
		goalContext,
		constraints.TimeAvailability,
		limitationContext,
		emotionalContext,
		psych.Motivation, psych.Stress, psych.Energy,
		input.DifficultyLevel,
		input.CompletionRate*100,
		input.StreakCount,
		profileContext,
		minTasks, maxTasks,
		categoriesList,
		input.DifficultyLevel,
		constraints.TimeAvailability,
		input.CompletionRate*100,
		categoriesList,
	)
}
