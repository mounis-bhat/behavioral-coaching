package behavior

import "encoding/json"

// CreateProfileInput represents profile creation input
// @Description Profile creation request
type CreateProfileInput struct {
	Goals              json.RawMessage `json:"goals" swaggertype:"object"`
	Constraints        json.RawMessage `json:"constraints" swaggertype:"object"`
	PsychologicalState json.RawMessage `json:"psychological_state" swaggertype:"object"`
	DifficultyLevel    *float64        `json:"difficulty_level,omitempty"`
}

// UpdateProfileInput represents profile update input
// @Description Profile update request
type UpdateProfileInput struct {
	Goals               json.RawMessage `json:"goals,omitempty" swaggertype:"object"`
	Constraints         json.RawMessage `json:"constraints,omitempty" swaggertype:"object"`
	PsychologicalState  json.RawMessage `json:"psychological_state,omitempty" swaggertype:"object"`
	DifficultyLevel     *float64        `json:"difficulty_level,omitempty"`
	OnboardingCompleted *bool           `json:"onboarding_completed,omitempty"`
}

// LogExecutionInput represents task execution log input
// @Description Task execution log request
type LogExecutionInput struct {
	Completed bool   `json:"completed"`
	Notes     string `json:"notes"`
}

// ProfileResponse represents a behavior profile
// @Description Behavior profile response
type ProfileResponse struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	Goals               json.RawMessage `json:"goals" swaggertype:"object"`
	Constraints         json.RawMessage `json:"constraints" swaggertype:"object"`
	PsychologicalState  json.RawMessage `json:"psychological_state" swaggertype:"object"`
	DifficultyLevel     float64         `json:"difficulty_level"`
	OnboardingCompleted bool            `json:"onboarding_completed"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

// PlanResponse represents a daily plan
// @Description Daily plan response
type PlanResponse struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	PlanDate        string         `json:"plan_date"`
	DifficultyScore float64        `json:"difficulty_score"`
	Status          string         `json:"status"`
	Tasks           []TaskResponse `json:"tasks"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

// TaskResponse represents a plan task
// @Description Plan task response
type TaskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Difficulty  float64 `json:"difficulty"`
	Position    int     `json:"position"`
	CreatedAt   string  `json:"created_at"`
}

// ExecutionLogResponse represents a task execution log entry
// @Description Task execution log response
type ExecutionLogResponse struct {
	ID          string  `json:"id"`
	PlanTaskID  string  `json:"plan_task_id"`
	UserID      string  `json:"user_id"`
	Completed   bool    `json:"completed"`
	CompletedAt *string `json:"completed_at,omitempty"`
	Notes       string  `json:"notes"`
	CreatedAt   string  `json:"created_at"`
}

// AdherenceResponse represents adherence metrics
// @Description Adherence metrics response
type AdherenceResponse struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"user_id"`
	CompletionRate     float64 `json:"completion_rate"`
	StreakCount        int     `json:"streak_count"`
	TotalTasks         int     `json:"total_tasks"`
	CompletedTasks     int     `json:"completed_tasks"`
	DifficultyMismatch bool    `json:"difficulty_mismatch"`
	LastComputedAt     string  `json:"last_computed_at"`
}
