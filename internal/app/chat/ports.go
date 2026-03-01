package chat

import "context"

// CoachingAgent is the interface the chat service uses to talk to the AI.
type CoachingAgent interface {
	Chat(ctx context.Context, req CoachingChatRequest) (*SendMessageResponse, error)
	StreamChat(ctx context.Context, req CoachingChatRequest) (<-chan ChatChunk, error)
}
