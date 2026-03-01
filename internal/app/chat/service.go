package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

type Service struct {
	queries *db.Queries
	agent   CoachingAgent
}

func NewService(queries *db.Queries, agent CoachingAgent) *Service {
	return &Service{queries: queries, agent: agent}
}

// ListMessages returns the last `limit` chat messages for the user, oldest first.
func (s *Service) ListMessages(ctx context.Context, userID string, limit int) ([]ChatMessageResponse, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	msgs, err := s.queries.ListChatMessages(ctx, db.ListChatMessagesParams{
		UserID: uid,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}

	resp := make([]ChatMessageResponse, len(msgs))
	for i, m := range msgs {
		resp[i] = messageToResponse(m)
	}
	return resp, nil
}

// StreamMessage persists the user message, calls the AI with streaming, then
// persists the assistant reply. It returns a channel of ChatChunk values.
// The final chunk (Done==true) carries both persisted messages in Response.
func (s *Service) StreamMessage(ctx context.Context, userID string, content string) (<-chan ChatChunk, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	// Persist user message first
	userMsg, err := s.queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
		UserID:  uid,
		Role:    "user",
		Content: content,
	})
	if err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	// Load conversation history (last 20 messages)
	history, err := s.loadHistory(ctx, uid, 20)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}

	// Build user context for AI prompt
	userCtx, err := s.buildUserContext(ctx, uid)
	if err != nil {
		// Non-fatal: proceed with empty context
		userCtx = UserContext{}
	}

	req := CoachingChatRequest{
		UserMessage: content,
		History:     history,
		UserContext: userCtx,
	}

	aiChunks, err := s.agent.StreamChat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("start AI stream: %w", err)
	}

	out := make(chan ChatChunk, 16)
	userMsgResp := messageToResponse(userMsg)

	go func() {
		defer close(out)
		var fullContent string

		for chunk := range aiChunks {
			if chunk.Err != nil {
				out <- ChatChunk{Err: chunk.Err}
				return
			}
			if chunk.Done {
				// Persist the complete assistant message
				assistantMsg, persistErr := s.queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
					UserID:  uid,
					Role:    "assistant",
					Content: fullContent,
				})
				if persistErr != nil {
					out <- ChatChunk{Err: fmt.Errorf("persist assistant message: %w", persistErr)}
					return
				}
				out <- ChatChunk{
					Done: true,
					Response: &SendMessageResponse{
						UserMessage:      userMsgResp,
						AssistantMessage: messageToResponse(assistantMsg),
					},
				}
				return
			}
			fullContent += chunk.Content
			out <- ChatChunk{Content: chunk.Content}
		}
	}()

	return out, nil
}

// ClearHistory deletes all chat messages for the user.
func (s *Service) ClearHistory(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return s.queries.DeleteChatHistory(ctx, uid)
}

// buildUserContext gathers profile, plan, journal, and adherence data.
func (s *Service) buildUserContext(ctx context.Context, uid pgtype.UUID) (UserContext, error) {
	var uctx UserContext

	// Profile
	profile, err := s.queries.GetBehaviorProfileByUserID(ctx, uid)
	if err == nil {
		type profileSnippet struct {
			Goals              json.RawMessage `json:"goals"`
			PsychologicalState json.RawMessage `json:"psychological_state"`
			DeliveryMode       string          `json:"delivery_mode"`
			EmotionalState     string          `json:"emotional_state"`
			AiProfileSummary   json.RawMessage `json:"ai_profile_summary"`
			DifficultyLevel    float64         `json:"difficulty_level"`
		}
		diffFloat := 5.0
		if profile.DifficultyLevel.Valid {
			f, _ := profile.DifficultyLevel.Float64Value()
			diffFloat = f.Float64
		}
		snip := profileSnippet{
			Goals:              profile.Goals,
			PsychologicalState: profile.PsychologicalState,
			DeliveryMode:       profile.DeliveryMode,
			EmotionalState:     profile.EmotionalState,
			AiProfileSummary:   profile.AiProfileSummary,
			DifficultyLevel:    diffFloat,
		}
		uctx.Profile, _ = json.Marshal(snip)
	}

	// Today's plan (best effort)
	plan, err := s.queries.GetActiveDailyPlan(ctx, uid)
	if err == nil {
		tasks, err2 := s.queries.GetPlanTasksByDailyPlan(ctx, plan.ID)
		if err2 == nil {
			type taskSnippet struct {
				Title    string `json:"title"`
				Category string `json:"category"`
			}
			type planSnippet struct {
				Status string        `json:"status"`
				Tasks  []taskSnippet `json:"tasks"`
			}
			ts := make([]taskSnippet, len(tasks))
			for i, t := range tasks {
				ts[i] = taskSnippet{Title: t.Title, Category: t.Category}
			}
			ps := planSnippet{Status: plan.Status, Tasks: ts}
			uctx.TodayPlan, _ = json.Marshal(ps)
		}
	}

	// Recent journal (last 7 entries)
	journalEntries, err := s.queries.GetJournalEntriesByUser(ctx, db.GetJournalEntriesByUserParams{
		UserID: uid,
		Limit:  7,
		Offset: 0,
	})
	if err == nil {
		snippets := make([]JournalSnippet, 0, len(journalEntries))
		for _, e := range journalEntries {
			c := e.Content
			if len(c) > 300 {
				c = c[:300]
			}
			snippets = append(snippets, JournalSnippet{
				Date:      e.EntryDate.Time.Format("2006-01-02"),
				MoodScore: int(e.MoodScore),
				Content:   c,
			})
		}
		uctx.RecentJournal = snippets
	}

	// Adherence (best effort)
	adherence, err := s.queries.GetAdherenceStateByUserID(ctx, uid)
	if err == nil && !errors.Is(err, pgx.ErrNoRows) {
		type adherenceSnippet struct {
			CompletionRate float64 `json:"completion_rate"`
			StreakCount    int32   `json:"streak_count"`
		}
		cr := 0.0
		if adherence.CompletionRate.Valid {
			f, _ := adherence.CompletionRate.Float64Value()
			cr = f.Float64
		}
		snip := adherenceSnippet{
			CompletionRate: cr,
			StreakCount:    adherence.StreakCount,
		}
		uctx.Adherence, _ = json.Marshal(snip)
	}

	return uctx, nil
}

// loadHistory fetches recent messages and converts them to ChatHistoryMessage slice.
func (s *Service) loadHistory(ctx context.Context, uid pgtype.UUID, limit int) ([]ChatHistoryMessage, error) {
	msgs, err := s.queries.ListChatMessages(ctx, db.ListChatMessagesParams{
		UserID: uid,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	history := make([]ChatHistoryMessage, len(msgs))
	for i, m := range msgs {
		history[i] = ChatHistoryMessage{Role: m.Role, Content: m.Content}
	}
	return history, nil
}

// --- Helpers ---

func parseUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
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

func messageToResponse(m db.ChatMessage) ChatMessageResponse {
	return ChatMessageResponse{
		ID:        uuidToString(m.ID),
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.CreatedAt.Time.Format(time.RFC3339),
	}
}
