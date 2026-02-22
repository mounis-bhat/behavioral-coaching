package journal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"

	"github.com/google/uuid"
)

var ErrEntryNotFound = errors.New("journal entry not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) UpsertEntry(ctx context.Context, userID string, input UpsertEntryInput) (*JournalEntryResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	if input.MoodScore < 1 || input.MoodScore > 10 {
		return nil, fmt.Errorf("mood_score must be between 1 and 10")
	}

	entry, err := s.queries.UpsertJournalEntry(ctx, db.UpsertJournalEntryParams{
		UserID:     uid,
		Content:    input.Content,
		MoodScore:  int32(input.MoodScore),
		PromptUsed: input.PromptUsed,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert journal entry: %w", err)
	}

	// Best-effort: update mood signal in behavior profile
	if err := s.updateMoodSignal(ctx, uid); err != nil {
		log.Printf("warning: failed to update mood signal for user %s: %v", userID, err)
	}

	return entryToResponse(entry), nil
}

func (s *Service) GetTodaysEntry(ctx context.Context, userID string) (*JournalEntryResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	entry, err := s.queries.GetJournalEntryByDate(ctx, db.GetJournalEntryByDateParams{
		UserID:    uid,
		EntryDate: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}
		return nil, fmt.Errorf("get today's journal entry: %w", err)
	}

	return entryToResponse(entry), nil
}

func (s *Service) ListEntries(ctx context.Context, userID string, limit, offset int) ([]JournalEntryResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	entries, err := s.queries.GetJournalEntriesByUser(ctx, db.GetJournalEntriesByUserParams{
		UserID: uid,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	resp := make([]JournalEntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = *entryToResponse(e)
	}
	return resp, nil
}

func (s *Service) GetMoodTrend(ctx context.Context, userID string) (*MoodTrendResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	scores, err := s.queries.GetRecentMoodScores(ctx, db.GetRecentMoodScoresParams{
		UserID: uid,
		Limit:  14,
	})
	if err != nil {
		return nil, fmt.Errorf("get mood scores: %w", err)
	}

	dataPoints := make([]MoodDataPoint, len(scores))
	for i, s := range scores {
		dataPoints[i] = MoodDataPoint{
			Date:  s.EntryDate.Time.Format("2006-01-02"),
			Score: int(s.MoodScore),
		}
	}

	avg, trend := computeTrend(scores)

	return &MoodTrendResponse{
		AvgScore:   avg,
		Trend:      trend,
		DataPoints: dataPoints,
	}, nil
}

// updateMoodSignal fetches the last 7 mood scores, computes the trend,
// and merges the "mood_trend" key into behavior_profiles.psychological_state.
func (s *Service) updateMoodSignal(ctx context.Context, uid pgtype.UUID) error {
	scores, err := s.queries.GetRecentMoodScores(ctx, db.GetRecentMoodScoresParams{
		UserID: uid,
		Limit:  7,
	})
	if err != nil || len(scores) == 0 {
		return nil
	}

	avg, trend := computeTrend(scores)

	// Read current psychological_state
	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err != nil {
		// No profile yet – skip silently
		return nil
	}

	current := make(map[string]any)
	if len(profile.PsychologicalState) > 0 {
		_ = json.Unmarshal(profile.PsychologicalState, &current)
	}

	current["mood_trend"] = map[string]any{
		"avg_score": avg,
		"trend":     trend,
	}

	merged, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("marshal psychological_state: %w", err)
	}

	_, err = s.queries.UpdateBehaviorProfile(ctx, db.UpdateBehaviorProfileParams{
		UserID:             uid,
		PsychologicalState: merged,
	})
	return err
}

// computeTrend calculates the average mood score and determines the trend direction.
// Compares the mean of the 3 most recent entries vs the prior entries.
func computeTrend(scores []db.GetRecentMoodScoresRow) (avg float64, trend string) {
	if len(scores) == 0 {
		return 0, "stable"
	}

	var sum float64
	for _, s := range scores {
		sum += float64(s.MoodScore)
	}
	avg = sum / float64(len(scores))

	if len(scores) < 4 {
		return avg, "stable"
	}

	// Recent 3 vs prior
	var recentSum float64
	for i := 0; i < 3; i++ {
		recentSum += float64(scores[i].MoodScore)
	}
	recentAvg := recentSum / 3

	var priorSum float64
	for i := 3; i < len(scores); i++ {
		priorSum += float64(scores[i].MoodScore)
	}
	priorAvg := priorSum / float64(len(scores)-3)

	diff := recentAvg - priorAvg
	switch {
	case diff > 0.5:
		trend = "improving"
	case diff < -0.5:
		trend = "declining"
	default:
		trend = "stable"
	}
	return avg, trend
}

// --- Helpers ---

func parseUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func entryToResponse(e db.JournalEntry) *JournalEntryResponse {
	return &JournalEntryResponse{
		ID:         uuidToString(e.ID),
		UserID:     uuidToString(e.UserID),
		Content:    e.Content,
		MoodScore:  int(e.MoodScore),
		PromptUsed: e.PromptUsed,
		EntryDate:  e.EntryDate.Time.Format("2006-01-02"),
		CreatedAt:  e.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  e.UpdatedAt.Time.Format(time.RFC3339),
	}
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
