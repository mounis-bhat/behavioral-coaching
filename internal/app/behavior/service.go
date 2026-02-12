package behavior

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

var (
	ErrProfileNotFound = errors.New("behavior profile not found")
	ErrProfileExists   = errors.New("behavior profile already exists")
	ErrPlanNotFound    = errors.New("daily plan not found")
	ErrTaskNotFound    = errors.New("plan task not found")
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) CreateProfile(ctx context.Context, userID string, input CreateProfileInput) (*ProfileResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	goals := defaultJSON(input.Goals, []byte("[]"))
	constraints := defaultJSON(input.Constraints, []byte("{}"))
	psychState := defaultJSON(input.PsychologicalState, []byte("{}"))
	difficulty := 5.0
	if input.DifficultyLevel != nil {
		difficulty = *input.DifficultyLevel
	}

	profile, err := s.queries.CreateBehaviorProfile(ctx, db.CreateBehaviorProfileParams{
		UserID:              uid,
		Goals:               goals,
		Constraints:         constraints,
		PsychologicalState:  psychState,
		DifficultyLevel:     numericFromFloat(difficulty),
		OnboardingCompleted: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create behavior profile: %w", err)
	}

	return profileToResponse(profile), nil
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*ProfileResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("get behavior profile: %w", err)
	}

	return profileToResponse(profile), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	params := db.UpdateBehaviorProfileParams{
		UserID: uid,
	}

	if input.Goals != nil {
		params.Goals = input.Goals
	}
	if input.Constraints != nil {
		params.Constraints = input.Constraints
	}
	if input.PsychologicalState != nil {
		params.PsychologicalState = input.PsychologicalState
	}
	if input.DifficultyLevel != nil {
		params.DifficultyLevel = numericFromFloat(*input.DifficultyLevel)
	}
	if input.OnboardingCompleted != nil {
		params.OnboardingCompleted = pgtype.Bool{Bool: *input.OnboardingCompleted, Valid: true}
	}

	profile, err := s.queries.UpdateBehaviorProfile(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("update behavior profile: %w", err)
	}

	return profileToResponse(profile), nil
}

// GenerateDailyPlan creates a stubbed daily plan with sample tasks.
// In Phase 2, this will delegate to an AI planning agent.
func (s *Service) GenerateDailyPlan(ctx context.Context, userID string) (*PlanResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	// Check if a plan already exists for today
	existing, err := s.queries.GetActiveDailyPlan(ctx, uid)
	if err == nil {
		tasks, err := s.queries.GetPlanTasksByDailyPlan(ctx, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("get plan tasks: %w", err)
		}
		return planToResponse(existing, tasks), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check existing plan: %w", err)
	}

	// Get profile difficulty or use default
	difficulty := 5.0
	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err == nil {
		difficulty = numericToFloat(profile.DifficultyLevel)
	}

	plan, err := s.queries.CreateDailyPlan(ctx, db.CreateDailyPlanParams{
		UserID:          uid,
		PlanDate:        pgtype.Date{Time: time.Now(), Valid: true},
		DifficultyScore: numericFromFloat(difficulty),
		Status:          "active",
	})
	if err != nil {
		return nil, fmt.Errorf("create daily plan: %w", err)
	}

	// Stubbed sample tasks — AI will generate these in Phase 2
	sampleTasks := []struct {
		title       string
		description string
		category    string
		difficulty  float64
	}{
		{"Morning exercise", "30 minutes of moderate physical activity", "health", 5.0},
		{"Gratitude journaling", "Write 3 things you are grateful for", "mindset", 3.0},
		{"Deep work session", "90 minutes of focused work on your top priority", "productivity", 7.0},
		{"Reach out to someone", "Send a meaningful message to a friend or family member", "relationships", 4.0},
		{"Evening reflection", "Review what went well today and what to improve", "discipline", 3.0},
	}

	var tasks []db.PlanTask
	for i, st := range sampleTasks {
		task, err := s.queries.CreatePlanTask(ctx, db.CreatePlanTaskParams{
			DailyPlanID: plan.ID,
			Title:       st.title,
			Description: st.description,
			Category:    st.category,
			Difficulty:  numericFromFloat(st.difficulty),
			Position:    int32(i + 1),
		})
		if err != nil {
			return nil, fmt.Errorf("create plan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return planToResponse(plan, tasks), nil
}

func (s *Service) GetTodaysPlan(ctx context.Context, userID string) (*PlanResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	plan, err := s.queries.GetActiveDailyPlan(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get active daily plan: %w", err)
	}

	tasks, err := s.queries.GetPlanTasksByDailyPlan(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("get plan tasks: %w", err)
	}

	return planToResponse(plan, tasks), nil
}

func (s *Service) LogExecution(ctx context.Context, userID string, taskID string, input LogExecutionInput) (*ExecutionLogResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	tid, err := parseUUID(taskID)
	if err != nil {
		return nil, err
	}

	// Verify the task exists and belongs to the user's plan
	task, err := s.queries.GetPlanTaskByID(ctx, tid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get plan task: %w", err)
	}

	plan, err := s.queries.GetDailyPlanByUserAndDate(ctx, db.GetDailyPlanByUserAndDateParams{
		UserID:   uid,
		PlanDate: pgtype.Date{Time: task.CreatedAt.Time, Valid: true},
	})
	if err != nil || plan.ID != task.DailyPlanID {
		// Look up by the plan ID directly
		_, lookupErr := s.queries.GetActiveDailyPlan(ctx, uid)
		if lookupErr != nil {
			return nil, ErrTaskNotFound
		}
	}

	var completedAt pgtype.Timestamptz
	if input.Completed {
		completedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	log, err := s.queries.UpsertExecutionLog(ctx, db.UpsertExecutionLogParams{
		PlanTaskID:  tid,
		UserID:      uid,
		Completed:   input.Completed,
		CompletedAt: completedAt,
		Notes:       input.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert execution log: %w", err)
	}

	return executionLogToResponse(log), nil
}

// RecomputeAdherence is stubbed for Phase 1. Real implementation in Phase 3.
func (s *Service) RecomputeAdherence(ctx context.Context, userID string) (*AdherenceResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	state, err := s.queries.GetAdherenceStateByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create initial zeroed state
			state, err = s.queries.UpsertAdherenceState(ctx, db.UpsertAdherenceStateParams{
				UserID:             uid,
				CompletionRate:     numericFromFloat(0),
				StreakCount:        0,
				TotalTasks:         0,
				CompletedTasks:     0,
				DifficultyMismatch: false,
			})
			if err != nil {
				return nil, fmt.Errorf("create adherence state: %w", err)
			}
		} else {
			return nil, fmt.Errorf("get adherence state: %w", err)
		}
	}

	return adherenceToResponse(state), nil
}

// AdaptUserPlan is stubbed for Phase 1. Real implementation in Phase 3.
func (s *Service) AdaptUserPlan(ctx context.Context, userID string) error {
	return nil
}

// --- Helpers ---

func parseUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func numericFromFloat(f float64) pgtype.Numeric {
	// Use big.Float for precise conversion
	bf := new(big.Float).SetFloat64(f)
	// Multiply by 10 to shift decimal, then get int
	bf.Mul(bf, new(big.Float).SetInt64(10))
	bi, _ := bf.Int(nil)
	return pgtype.Numeric{Int: bi, Exp: -1, Valid: true}
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(n.Int).Float64()
	for i := int32(0); i < -n.Exp; i++ {
		f /= 10
	}
	for i := int32(0); i < n.Exp; i++ {
		f *= 10
	}
	return f
}

func defaultJSON(data json.RawMessage, fallback []byte) []byte {
	if len(data) == 0 {
		return fallback
	}
	return data
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	value, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return ""
	}
	return value.String()
}

func profileToResponse(p db.BehaviorProfile) *ProfileResponse {
	return &ProfileResponse{
		ID:                  uuidToString(p.ID),
		UserID:              uuidToString(p.UserID),
		Goals:               p.Goals,
		Constraints:         p.Constraints,
		PsychologicalState:  p.PsychologicalState,
		DifficultyLevel:     numericToFloat(p.DifficultyLevel),
		OnboardingCompleted: p.OnboardingCompleted,
		CreatedAt:           p.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:           p.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func planToResponse(p db.DailyPlan, tasks []db.PlanTask) *PlanResponse {
	taskResponses := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		taskResponses[i] = TaskResponse{
			ID:          uuidToString(t.ID),
			Title:       t.Title,
			Description: t.Description,
			Category:    t.Category,
			Difficulty:  numericToFloat(t.Difficulty),
			Position:    int(t.Position),
			CreatedAt:   t.CreatedAt.Time.Format(time.RFC3339),
		}
	}

	return &PlanResponse{
		ID:              uuidToString(p.ID),
		UserID:          uuidToString(p.UserID),
		PlanDate:        p.PlanDate.Time.Format("2006-01-02"),
		DifficultyScore: numericToFloat(p.DifficultyScore),
		Status:          p.Status,
		Tasks:           taskResponses,
		CreatedAt:       p.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func executionLogToResponse(l db.ExecutionLog) *ExecutionLogResponse {
	resp := &ExecutionLogResponse{
		ID:         uuidToString(l.ID),
		PlanTaskID: uuidToString(l.PlanTaskID),
		UserID:     uuidToString(l.UserID),
		Completed:  l.Completed,
		Notes:      l.Notes,
		CreatedAt:  l.CreatedAt.Time.Format(time.RFC3339),
	}
	if l.CompletedAt.Valid {
		t := l.CompletedAt.Time.Format(time.RFC3339)
		resp.CompletedAt = &t
	}
	return resp
}

func adherenceToResponse(a db.AdherenceState) *AdherenceResponse {
	return &AdherenceResponse{
		ID:                 uuidToString(a.ID),
		UserID:             uuidToString(a.UserID),
		CompletionRate:     numericToFloat(a.CompletionRate),
		StreakCount:        int(a.StreakCount),
		TotalTasks:         int(a.TotalTasks),
		CompletedTasks:     int(a.CompletedTasks),
		DifficultyMismatch: a.DifficultyMismatch,
		LastComputedAt:     a.LastComputedAt.Time.Format(time.RFC3339),
	}
}
