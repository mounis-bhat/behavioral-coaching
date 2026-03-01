package chat

import "encoding/json"

// ChatHistoryMessage is a single message in the conversation history passed to the AI.
type ChatHistoryMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// JournalSnippet is a compact view of a recent journal entry for AI context.
type JournalSnippet struct {
	Date      string `json:"date"`
	MoodScore int    `json:"mood_score"`
	Content   string `json:"content"` // first 300 chars
}

// UserContext bundles all profile/plan/journal/adherence data for the AI prompt.
type UserContext struct {
	Profile       json.RawMessage  `json:"profile"`
	TodayPlan     json.RawMessage  `json:"today_plan,omitempty"`
	RecentJournal []JournalSnippet `json:"recent_journal"`
	Adherence     json.RawMessage  `json:"adherence,omitempty"`
}

// CoachingChatRequest is the input to the coaching AI flow.
type CoachingChatRequest struct {
	UserMessage string               `json:"user_message"`
	History     []ChatHistoryMessage `json:"history"`
	UserContext UserContext          `json:"user_context"`
}

// ChatMessageResponse is the serialized form of a persisted chat message.
type ChatMessageResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// SendMessageRequest is the HTTP request body for sending a message.
type SendMessageRequest struct {
	Content string `json:"content"`
}

// SendMessageResponse is the non-streaming HTTP response returned after both
// the user message and assistant reply have been persisted.
type SendMessageResponse struct {
	UserMessage      ChatMessageResponse `json:"user_message"`
	AssistantMessage ChatMessageResponse `json:"assistant_message"`
}

// ChatChunk is a single SSE chunk emitted during streaming.
type ChatChunk struct {
	Content  string               `json:"content,omitempty"`
	Done     bool                 `json:"done"`
	Response *SendMessageResponse `json:"response,omitempty"`
	Err      error                `json:"-"`
}
