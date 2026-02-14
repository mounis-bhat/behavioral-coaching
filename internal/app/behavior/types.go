package behavior

import (
	"encoding/json"
	"fmt"
	"slices"
)

// CreateProfileInput represents profile creation input
// @Description Profile creation request
type CreateProfileInput struct {
	Goals              json.RawMessage `json:"goals" swaggertype:"object"`
	Constraints        json.RawMessage `json:"constraints" swaggertype:"object"`
	PsychologicalState json.RawMessage `json:"psychological_state" swaggertype:"object"`
	DifficultyLevel    *float64        `json:"difficulty_level,omitempty"`
	DeliveryMode       string          `json:"delivery_mode,omitempty"`
	EmotionalState     string          `json:"emotional_state,omitempty"`
	AIProfileSummary   json.RawMessage `json:"ai_profile_summary,omitempty" swaggertype:"object"`
}

// UpdateProfileInput represents profile update input
// @Description Profile update request
type UpdateProfileInput struct {
	Goals               json.RawMessage `json:"goals,omitempty" swaggertype:"object"`
	Constraints         json.RawMessage `json:"constraints,omitempty" swaggertype:"object"`
	PsychologicalState  json.RawMessage `json:"psychological_state,omitempty" swaggertype:"object"`
	DifficultyLevel     *float64        `json:"difficulty_level,omitempty"`
	OnboardingCompleted *bool           `json:"onboarding_completed,omitempty"`
	DeliveryMode        *string         `json:"delivery_mode,omitempty"`
	EmotionalState      *string         `json:"emotional_state,omitempty"`
	AIProfileSummary    json.RawMessage `json:"ai_profile_summary,omitempty" swaggertype:"object"`
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
	DeliveryMode        string          `json:"delivery_mode"`
	EmotionalState      string          `json:"emotional_state"`
	AIProfileSummary    json.RawMessage `json:"ai_profile_summary" swaggertype:"object"`
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
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Difficulty    float64 `json:"difficulty"`
	Position      int     `json:"position"`
	TaskType      string  `json:"task_type"`
	SuggestedTime string  `json:"suggested_time"`
	AnchorLabel   string  `json:"anchor_label"`
	CreatedAt     string  `json:"created_at"`
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

// AdaptationLogResponse represents an adaptation log entry
// @Description Adaptation log response
type AdaptationLogResponse struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	AdaptationReason   string          `json:"adaptation_reason"`
	DifficultyChange   float64         `json:"difficulty_change"`
	PreviousDifficulty float64         `json:"previous_difficulty"`
	NewDifficulty      float64         `json:"new_difficulty"`
	TriggerMetrics     json.RawMessage `json:"trigger_metrics" swaggertype:"object"`
	CreatedAt          string          `json:"created_at"`
}

// AnalyticsResponse contains weekly trends, category performance, and difficulty trajectory.
// @Description Analytics response with trends and performance data
type AnalyticsResponse struct {
	WeeklyTrends         []WeeklyTrend        `json:"weekly_trends"`
	CategoryPerformance  []CategoryPerformance `json:"category_performance"`
	DifficultyTrajectory []DifficultyPoint     `json:"difficulty_trajectory"`
}

// WeeklyTrend represents completion data for a single week.
// @Description Weekly completion trend
type WeeklyTrend struct {
	WeekStart      string  `json:"week_start"`
	PlanCount      int     `json:"plan_count"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

// CategoryPerformance represents completion data for a single task category.
// @Description Category performance breakdown
type CategoryPerformance struct {
	Category       string  `json:"category"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

// DifficultyPoint represents the difficulty score on a given date.
// @Description Difficulty trajectory point
type DifficultyPoint struct {
	Date       string  `json:"date"`
	Difficulty float64 `json:"difficulty"`
}

// --- AI Agent Request/Response Types ---

var validCategories = []string{"health", "mindset", "discipline", "relationships", "productivity"}

// PlanRequest is the input to the planning agent.
type PlanRequest struct {
	Goals              json.RawMessage `json:"goals"`
	Constraints        json.RawMessage `json:"constraints"`
	PsychologicalState json.RawMessage `json:"psychological_state"`
	DifficultyLevel    float64         `json:"difficulty_level"`
	CompletionRate     float64         `json:"completion_rate"`
	StreakCount        int             `json:"streak_count"`
	DeliveryMode       string          `json:"delivery_mode"`
	EmotionalState     string          `json:"emotional_state"`
	AIProfileSummary   json.RawMessage `json:"ai_profile_summary"`
}

// PlanResult is the structured output from the planning agent.
type PlanResult struct {
	Tasks []GeneratedTask `json:"tasks"`
}

// GeneratedTask represents a single AI-generated task.
type GeneratedTask struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Difficulty    float64 `json:"difficulty"`
	TaskType      string  `json:"task_type"`
	SuggestedTime string  `json:"suggested_time"`
	AnchorLabel   string  `json:"anchor_label"`
}

var validTaskTypes = []string{"regular", "anchor"}
var validAnchorLabels = []string{"", "wake_window", "activation_activity", "sleep_winddown"}

// Validate checks that the PlanResult contains valid tasks.
func (r *PlanResult) Validate() error {
	if len(r.Tasks) == 0 {
		return fmt.Errorf("plan must contain at least 1 task")
	}
	for i, t := range r.Tasks {
		if !slices.Contains(validCategories, t.Category) {
			return fmt.Errorf("task %d: invalid category %q", i, t.Category)
		}
		if t.Difficulty < 1.0 || t.Difficulty > 10.0 {
			return fmt.Errorf("task %d: difficulty %.1f out of range [1.0, 10.0]", i, t.Difficulty)
		}
		// Default task_type to "regular" if empty
		if t.TaskType == "" {
			r.Tasks[i].TaskType = "regular"
		} else if !slices.Contains(validTaskTypes, t.TaskType) {
			return fmt.Errorf("task %d: invalid task_type %q", i, t.TaskType)
		}
		if !slices.Contains(validAnchorLabels, t.AnchorLabel) {
			return fmt.Errorf("task %d: invalid anchor_label %q", i, t.AnchorLabel)
		}
	}
	return nil
}

// AdaptationRequest is the input to the adaptation agent.
type AdaptationRequest struct {
	Goals              json.RawMessage `json:"goals"`
	CurrentDifficulty  float64         `json:"current_difficulty"`
	CompletionRate     float64         `json:"completion_rate"`
	StreakCount        int             `json:"streak_count"`
	TotalTasks         int             `json:"total_tasks"`
	CompletedTasks     int             `json:"completed_tasks"`
	DifficultyMismatch bool            `json:"difficulty_mismatch"`
}

// AdaptationResult is the structured output from the adaptation agent.
type AdaptationResult struct {
	ShouldAdapt   bool    `json:"should_adapt"`
	NewDifficulty float64 `json:"new_difficulty"`
	Reason        string  `json:"reason"`
}

// Validate checks that the AdaptationResult is well-formed.
func (r *AdaptationResult) Validate() error {
	if r.NewDifficulty < 0.0 || r.NewDifficulty > 10.0 {
		return fmt.Errorf("new_difficulty %.1f out of range [0.0, 10.0]", r.NewDifficulty)
	}
	if r.ShouldAdapt && r.Reason == "" {
		return fmt.Errorf("reason must be non-empty when should_adapt is true")
	}
	return nil
}

// --- Onboarding Chat Types ---

// OnboardingChatMessage represents a single message in the profiling conversation.
type OnboardingChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OnboardingChatRequest is the input to the profiling conversation agent.
type OnboardingChatRequest struct {
	UserMessage        string                  `json:"user_message"`
	History            []OnboardingChatMessage `json:"history"`
	Goals              json.RawMessage         `json:"goals"`
	Constraints        json.RawMessage         `json:"constraints"`
	PsychologicalState json.RawMessage         `json:"psychological_state"`
}

// OnboardingChatResponse is the output from the profiling conversation agent.
type OnboardingChatResponse struct {
	AIMessage                 string                  `json:"ai_message"`
	History                   []OnboardingChatMessage `json:"history"`
	TurnNumber                int                     `json:"turn_number"`
	IsComplete                bool                    `json:"is_complete"`
	Summary                   json.RawMessage         `json:"summary,omitempty" swaggertype:"object"`
	RecommendedDeliveryMode   string                  `json:"recommended_delivery_mode,omitempty"`
	RecommendedEmotionalState string                  `json:"recommended_emotional_state,omitempty"`
}
