package behavior

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	appchat "github.com/mounis-bhat/starter/internal/app/chat"
)

// CoachingAgent wraps a Genkit streaming flow for persistent AI coaching chat.
type CoachingAgent struct {
	flow *core.Flow[*appchat.CoachingChatRequest, *appchat.SendMessageResponse, string]
}

// NewCoachingAgent creates a CoachingAgent backed by a Genkit streaming flow.
func NewCoachingAgent(g *genkit.Genkit) *CoachingAgent {
	flow := genkit.DefineStreamingFlow(g, "coachingChatFlow",
		func(ctx context.Context, input *appchat.CoachingChatRequest, sendChunk core.StreamCallback[string]) (*appchat.SendMessageResponse, error) {
			systemPrompt := buildCoachingSystemPrompt(input)

			var messages []*ai.Message
			messages = append(messages, ai.NewSystemMessage(ai.NewTextPart(systemPrompt)))

			for _, msg := range input.History {
				if msg.Role == "user" {
					messages = append(messages, ai.NewUserMessage(ai.NewTextPart(msg.Content)))
				} else {
					messages = append(messages, ai.NewModelMessage(ai.NewTextPart(msg.Content)))
				}
			}

			if input.UserMessage != "" {
				messages = append(messages, ai.NewUserMessage(ai.NewTextPart(input.UserMessage)))
			}

			stream := genkit.GenerateStream(ctx, g, ai.WithMessages(messages...))

			var aiMessage string
			for result, err := range stream {
				if err != nil {
					return nil, fmt.Errorf("generate coaching response: %w", err)
				}
				if result.Done {
					aiMessage = result.Response.Text()
					break
				}
				if err := sendChunk(ctx, result.Chunk.Text()); err != nil {
					return nil, fmt.Errorf("send chunk: %w", err)
				}
			}

			return &appchat.SendMessageResponse{
				AssistantMessage: appchat.ChatMessageResponse{Content: aiMessage},
			}, nil
		})

	return &CoachingAgent{flow: flow}
}

// Chat runs the coaching flow without streaming.
func (a *CoachingAgent) Chat(ctx context.Context, req appchat.CoachingChatRequest) (*appchat.SendMessageResponse, error) {
	return a.flow.Run(ctx, &req)
}

// StreamChat runs the coaching flow and streams chunk-by-chunk.
func (a *CoachingAgent) StreamChat(ctx context.Context, req appchat.CoachingChatRequest) (<-chan appchat.ChatChunk, error) {
	out := make(chan appchat.ChatChunk, 16)
	go func() {
		defer close(out)
		for result, err := range a.flow.Stream(ctx, &req) {
			if err != nil {
				out <- appchat.ChatChunk{Err: err}
				return
			}
			if result.Done {
				out <- appchat.ChatChunk{Done: true}
			} else {
				out <- appchat.ChatChunk{Content: result.Stream}
			}
		}
	}()
	return out, nil
}

func buildCoachingSystemPrompt(input *appchat.CoachingChatRequest) string {
	base := `You are a warm, supportive behavioral coach. Your role is to help the user stay on track with their behavioral goals, reflect on their progress, and navigate challenges. You have full access to their profile, daily plan, journal data, and adherence metrics.

STRICT SCOPE — Only discuss:
- The user's behavioral goals, habits, and progress
- Their wellbeing, motivation, stress, or emotional state
- Their daily plans, tasks, schedule, and behavioral patterns
- Coaching strategies and techniques tied to their specific profile
- Reflections on their journal entries or mood trends
- Adherence data — streaks, completion rates, what's working or not

If the user asks about anything outside this scope (weather, news, recipes, coding help, general trivia, etc.), gently but firmly redirect them back to their coaching journey. Say something like: "That's outside what I can help with here — I'm here to support your behavioral goals. Is there something about your progress or wellbeing you'd like to explore?"

COMMUNICATION STYLE:
- Warm and empathetic, never clinical or robotic
- Concise: 3-5 sentences unless the user needs more depth
- Ask follow-up questions to deepen reflection
- Celebrate progress, normalise setbacks
- Avoid generic advice — always tie responses to the user's specific context below
- Always respond using Markdown formatting: use **bold** for emphasis, bullet lists for multiple points, and headers (##) for structured responses when appropriate

USER CONTEXT:
`

	// Append profile context
	if len(input.UserContext.Profile) > 0 && string(input.UserContext.Profile) != "null" {
		profileStr := prettyJSON(input.UserContext.Profile)
		base += fmt.Sprintf("\n## Profile\n%s\n", profileStr)
	}

	// Append today's plan
	if len(input.UserContext.TodayPlan) > 0 && string(input.UserContext.TodayPlan) != "null" {
		planStr := prettyJSON(input.UserContext.TodayPlan)
		base += fmt.Sprintf("\n## Today's Plan\n%s\n", planStr)
	}

	// Append recent journal
	if len(input.UserContext.RecentJournal) > 0 {
		journalBytes, _ := json.Marshal(input.UserContext.RecentJournal)
		base += fmt.Sprintf("\n## Recent Journal (last 7 entries)\n%s\n", string(journalBytes))
	}

	// Append adherence
	if len(input.UserContext.Adherence) > 0 && string(input.UserContext.Adherence) != "null" {
		adherenceStr := prettyJSON(input.UserContext.Adherence)
		base += fmt.Sprintf("\n## Adherence\n%s\n", adherenceStr)
	}

	return base
}

func prettyJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}
