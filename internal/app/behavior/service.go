package behavior

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	queries  *db.Queries
	planner  PlanGenerator
	advisor  AdaptationAdvisor
	profiler ProfileConversationAgent
}

func NewService(queries *db.Queries, planner PlanGenerator, advisor AdaptationAdvisor, profiler ProfileConversationAgent) *Service {
	return &Service{queries: queries, planner: planner, advisor: advisor, profiler: profiler}
}

func (s *Service) CreateProfile(ctx context.Context, userID string, input CreateProfileInput) (*ProfileResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	goals := defaultJSON(input.Goals, []byte("[]"))
	constraints := defaultJSON(input.Constraints, []byte("{}"))
	psychState := defaultJSON(input.PsychologicalState, []byte("{}"))
	aiProfileSummary := defaultJSON(input.AIProfileSummary, []byte("{}"))
	difficulty := 5.0
	if input.DifficultyLevel != nil {
		difficulty = *input.DifficultyLevel
	}
	deliveryMode := "flexible_list"
	if input.DeliveryMode != "" {
		deliveryMode = input.DeliveryMode
	}

	profile, err := s.queries.CreateBehaviorProfile(ctx, db.CreateBehaviorProfileParams{
		UserID:              uid,
		Goals:               goals,
		Constraints:         constraints,
		PsychologicalState:  psychState,
		DifficultyLevel:     numericFromFloat(difficulty),
		OnboardingCompleted: false,
		DeliveryMode:        deliveryMode,
		EmotionalState:      input.EmotionalState,
		AiProfileSummary:    aiProfileSummary,
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
	if input.DeliveryMode != nil {
		params.DeliveryMode = pgtype.Text{String: *input.DeliveryMode, Valid: true}
	}
	if input.EmotionalState != nil {
		params.EmotionalState = pgtype.Text{String: *input.EmotionalState, Valid: true}
	}
	if input.AIProfileSummary != nil {
		params.AiProfileSummary = input.AIProfileSummary
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

	// Get profile and adherence data for the AI planner
	difficulty := 5.0
	var completionRate float64
	var streakCount int
	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err == nil {
		difficulty = numericToFloat(profile.DifficultyLevel)
	}

	adherence, err := s.queries.GetAdherenceStateByUserID(ctx, uid)
	if err == nil {
		completionRate = numericToFloat(adherence.CompletionRate)
		streakCount = int(adherence.StreakCount)
	}

	// Build the AI request from profile data
	planReq := PlanRequest{
		Goals:              defaultJSON(profile.Goals, []byte("[]")),
		Constraints:        defaultJSON(profile.Constraints, []byte("{}")),
		PsychologicalState: defaultJSON(profile.PsychologicalState, []byte("{}")),
		DifficultyLevel:    difficulty,
		CompletionRate:     completionRate,
		StreakCount:        streakCount,
		DeliveryMode:       profile.DeliveryMode,
		EmotionalState:     profile.EmotionalState,
		AIProfileSummary:   defaultJSON(profile.AiProfileSummary, []byte("{}")),
	}

	aiResult, err := s.planner.GeneratePlan(ctx, planReq)
	if err != nil {
		return nil, fmt.Errorf("ai plan generation: %w", err)
	}

	plan, err := s.queries.CreateDailyPlan(ctx, db.CreateDailyPlanParams{
		UserID:          uid,
		PlanDate:        pgtype.Date{Time: time.Now(), Valid: true},
		DifficultyScore: numericFromFloat(difficulty),
		Status:          "active",
		PromptVersion:   s.planner.PromptVersion(),
	})
	if err != nil {
		return nil, fmt.Errorf("create daily plan: %w", err)
	}

	var tasks []db.PlanTask
	for i, gt := range aiResult.Tasks {
		task, err := s.queries.CreatePlanTask(ctx, db.CreatePlanTaskParams{
			DailyPlanID:   plan.ID,
			Title:         gt.Title,
			Description:   gt.Description,
			Category:      gt.Category,
			Difficulty:    numericFromFloat(gt.Difficulty),
			Position:      int32(i + 1),
			TaskType:      gt.TaskType,
			SuggestedTime: gt.SuggestedTime,
			AnchorLabel:   gt.AnchorLabel,
		})
		if err != nil {
			return nil, fmt.Errorf("create plan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return planToResponse(plan, tasks), nil
}

// OnboardingChat delegates to the profiling conversation agent.
func (s *Service) OnboardingChat(ctx context.Context, req OnboardingChatRequest) (*OnboardingChatResponse, error) {
	return s.profiler.Chat(ctx, req)
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

	execLog, err := s.queries.UpsertExecutionLog(ctx, db.UpsertExecutionLogParams{
		PlanTaskID:  tid,
		UserID:      uid,
		Completed:   input.Completed,
		CompletedAt: completedAt,
		Notes:       input.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert execution log: %w", err)
	}

	// Best-effort: recompute adherence and trigger adaptation if needed
	adherence, err := s.RecomputeAdherence(ctx, userID)
	if err != nil {
		log.Printf("warning: failed to recompute adherence after execution log: %v", err)
	} else if adherence.DifficultyMismatch {
		if adaptErr := s.AdaptUserPlan(ctx, userID); adaptErr != nil {
			log.Printf("warning: failed to adapt user plan after mismatch: %v", adaptErr)
		}
	}

	return executionLogToResponse(execLog), nil
}

// RecomputeAdherence computes real adherence metrics from today's plan and execution logs.
func (s *Service) RecomputeAdherence(ctx context.Context, userID string) (*AdherenceResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	// Get today's active plan
	plan, err := s.queries.GetActiveDailyPlan(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No active plan today → upsert zeroed state
			state, uErr := s.queries.UpsertAdherenceState(ctx, db.UpsertAdherenceStateParams{
				UserID:             uid,
				CompletionRate:     numericFromFloat(0),
				StreakCount:        0,
				TotalTasks:         0,
				CompletedTasks:     0,
				DifficultyMismatch: false,
			})
			if uErr != nil {
				return nil, fmt.Errorf("upsert zeroed adherence state: %w", uErr)
			}
			return adherenceToResponse(state), nil
		}
		return nil, fmt.Errorf("get active daily plan: %w", err)
	}

	// Get tasks and execution logs for today's plan
	tasks, err := s.queries.GetPlanTasksByDailyPlan(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("get plan tasks for adherence: %w", err)
	}

	logs, err := s.queries.GetExecutionLogsByDailyPlan(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("get execution logs for adherence: %w", err)
	}

	// Compute completion metrics
	totalTasks := len(tasks)
	var completedTasks int
	for _, l := range logs {
		if l.Completed {
			completedTasks++
		}
	}

	var completionRate float64
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks)
	}

	// Compute streak: consecutive days with ≥70% completion
	streakCount := s.computeStreak(ctx, uid)

	// Detect difficulty mismatch: completion rate below 40%
	difficultyMismatch := completionRate < 0.4 && totalTasks > 0

	state, err := s.queries.UpsertAdherenceState(ctx, db.UpsertAdherenceStateParams{
		UserID:             uid,
		CompletionRate:     numericFromFloat(completionRate),
		StreakCount:        int32(streakCount),
		TotalTasks:         int32(totalTasks),
		CompletedTasks:     int32(completedTasks),
		DifficultyMismatch: difficultyMismatch,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert adherence state: %w", err)
	}

	return adherenceToResponse(state), nil
}

// computeStreak counts consecutive recent days where the plan had ≥70% completion.
// Uses a single query instead of N+1.
func (s *Service) computeStreak(ctx context.Context, uid pgtype.UUID) int {
	rates, err := s.queries.GetDailyCompletionRates(ctx, db.GetDailyCompletionRatesParams{
		UserID: uid,
		Limit:  30,
	})
	if err != nil || len(rates) == 0 {
		return 0
	}

	var streak int
	for _, r := range rates {
		if r.TotalTasks == 0 {
			break
		}
		if float64(r.CompletedTasks)/float64(r.TotalTasks) < 0.7 {
			break
		}
		streak++
	}
	return streak
}

// AdaptUserPlan queries adherence metrics, asks the AI advisor for adaptation advice,
// and applies changes (updates profile difficulty, creates adaptation log) when needed.
func (s *Service) AdaptUserPlan(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}

	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err != nil {
		return fmt.Errorf("get profile for adaptation: %w", err)
	}

	adherence, err := s.queries.GetAdherenceStateByUserID(ctx, uid)
	if err != nil {
		// No adherence data yet — nothing to adapt
		return nil
	}

	currentDifficulty := numericToFloat(profile.DifficultyLevel)
	completionRate := numericToFloat(adherence.CompletionRate)
	totalTasks := int(adherence.TotalTasks)
	completedTasks := int(adherence.CompletedTasks)
	streakCount := int(adherence.StreakCount)

	req := AdaptationRequest{
		Goals:              profile.Goals,
		CurrentDifficulty:  currentDifficulty,
		CompletionRate:     completionRate,
		StreakCount:        streakCount,
		TotalTasks:         totalTasks,
		CompletedTasks:     completedTasks,
		DifficultyMismatch: adherence.DifficultyMismatch,
	}

	result, err := s.advisor.Advise(ctx, req)
	if err != nil {
		return fmt.Errorf("ai adaptation advice: %w", err)
	}

	if !result.ShouldAdapt {
		log.Printf("adaptation: no change needed for user %s (reason=%q)", userID, result.Reason)
		return nil
	}

	// Update profile difficulty
	_, err = s.queries.UpdateBehaviorProfile(ctx, db.UpdateBehaviorProfileParams{
		UserID:          uid,
		DifficultyLevel: numericFromFloat(result.NewDifficulty),
	})
	if err != nil {
		return fmt.Errorf("update profile difficulty: %w", err)
	}

	// Marshal trigger metrics
	triggerMetrics, err := json.Marshal(map[string]any{
		"completion_rate":     completionRate,
		"streak_count":        streakCount,
		"total_tasks":         totalTasks,
		"completed_tasks":     completedTasks,
		"difficulty_mismatch": adherence.DifficultyMismatch,
	})
	if err != nil {
		return fmt.Errorf("marshal trigger metrics: %w", err)
	}

	// Persist adaptation log
	difficultyChange := result.NewDifficulty - currentDifficulty
	_, err = s.queries.CreateAdaptationLog(ctx, db.CreateAdaptationLogParams{
		UserID:             uid,
		AdaptationReason:   result.Reason,
		DifficultyChange:   numericFromFloat(difficultyChange),
		PreviousDifficulty: numericFromFloat(currentDifficulty),
		NewDifficulty:      numericFromFloat(result.NewDifficulty),
		TriggerMetrics:     triggerMetrics,
	})
	if err != nil {
		return fmt.Errorf("create adaptation log: %w", err)
	}

	log.Printf("adaptation applied for user %s: difficulty %.1f → %.1f (reason=%q)",
		userID, currentDifficulty, result.NewDifficulty, result.Reason)

	return nil
}

func (s *Service) GetAdaptationLogs(ctx context.Context, userID string, limit int) ([]AdaptationLogResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	logs, err := s.queries.GetAdaptationLogsByUser(ctx, db.GetAdaptationLogsByUserParams{
		UserID: uid,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get adaptation logs: %w", err)
	}

	resp := make([]AdaptationLogResponse, len(logs))
	for i, l := range logs {
		resp[i] = adaptationLogToResponse(l)
	}
	return resp, nil
}

func (s *Service) GetTodaysExecutionLogs(ctx context.Context, userID string) ([]ExecutionLogResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	plan, err := s.queries.GetActiveDailyPlan(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []ExecutionLogResponse{}, nil
		}
		return nil, fmt.Errorf("get active daily plan: %w", err)
	}

	logs, err := s.queries.GetExecutionLogsByDailyPlan(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("get execution logs: %w", err)
	}

	resp := make([]ExecutionLogResponse, len(logs))
	for i, l := range logs {
		resp[i] = *executionLogToResponse(l)
	}
	return resp, nil
}

// GetAnalytics returns weekly trends, category performance, and difficulty trajectory.
func (s *Service) GetAnalytics(ctx context.Context, userID string) (*AnalyticsResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	trends, err := s.queries.GetWeeklyCompletionTrends(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get weekly trends: %w", err)
	}

	categories, err := s.queries.GetCategoryPerformance(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get category performance: %w", err)
	}

	trajectory, err := s.queries.GetDifficultyTrajectory(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get difficulty trajectory: %w", err)
	}

	resp := &AnalyticsResponse{
		WeeklyTrends:         make([]WeeklyTrend, len(trends)),
		CategoryPerformance:  make([]CategoryPerformance, len(categories)),
		DifficultyTrajectory: make([]DifficultyPoint, len(trajectory)),
	}

	for i, t := range trends {
		var rate float64
		if t.TotalTasks > 0 {
			rate = float64(t.CompletedTasks) / float64(t.TotalTasks)
		}
		resp.WeeklyTrends[i] = WeeklyTrend{
			WeekStart:      t.WeekStart.Time.Format("2006-01-02"),
			PlanCount:      int(t.PlanCount),
			TotalTasks:     int(t.TotalTasks),
			CompletedTasks: int(t.CompletedTasks),
			CompletionRate: rate,
		}
	}

	for i, c := range categories {
		var rate float64
		if c.TotalTasks > 0 {
			rate = float64(c.CompletedTasks) / float64(c.TotalTasks)
		}
		resp.CategoryPerformance[i] = CategoryPerformance{
			Category:       c.Category,
			TotalTasks:     int(c.TotalTasks),
			CompletedTasks: int(c.CompletedTasks),
			CompletionRate: rate,
		}
	}

	for i, d := range trajectory {
		resp.DifficultyTrajectory[i] = DifficultyPoint{
			Date:       d.PlanDate.Time.Format("2006-01-02"),
			Difficulty: numericToFloat(d.DifficultyScore),
		}
	}

	return resp, nil
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
		DeliveryMode:        p.DeliveryMode,
		EmotionalState:      p.EmotionalState,
		AIProfileSummary:    defaultJSON(p.AiProfileSummary, []byte("{}")),
		CreatedAt:           p.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:           p.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func planToResponse(p db.DailyPlan, tasks []db.PlanTask) *PlanResponse {
	taskResponses := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		taskResponses[i] = TaskResponse{
			ID:            uuidToString(t.ID),
			Title:         t.Title,
			Description:   t.Description,
			Category:      t.Category,
			Difficulty:    numericToFloat(t.Difficulty),
			Position:      int(t.Position),
			TaskType:      t.TaskType,
			SuggestedTime: t.SuggestedTime,
			AnchorLabel:   t.AnchorLabel,
			CreatedAt:     t.CreatedAt.Time.Format(time.RFC3339),
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

func adaptationLogToResponse(l db.AdaptationLog) AdaptationLogResponse {
	return AdaptationLogResponse{
		ID:                 uuidToString(l.ID),
		UserID:             uuidToString(l.UserID),
		AdaptationReason:   l.AdaptationReason,
		DifficultyChange:   numericToFloat(l.DifficultyChange),
		PreviousDifficulty: numericToFloat(l.PreviousDifficulty),
		NewDifficulty:      numericToFloat(l.NewDifficulty),
		TriggerMetrics:     json.RawMessage(l.TriggerMetrics),
		CreatedAt:          l.CreatedAt.Time.Format(time.RFC3339),
	}
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
