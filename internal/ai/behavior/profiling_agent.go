package behavior

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	appbehavior "github.com/mounis-bhat/starter/internal/app/behavior"
)

const maxProfilingTurns = 5

// profilingSummary is the structured output for the final turn.
type profilingSummary struct {
	Summary                   string   `json:"summary"`
	KeyInsights               []string `json:"key_insights"`
	RecommendedDeliveryMode   string   `json:"recommended_delivery_mode"`
	RecommendedEmotionalState string   `json:"recommended_emotional_state"`
}

// ProfilingAgent wraps a Genkit streaming flow for multi-turn onboarding profiling.
type ProfilingAgent struct {
	flow *core.Flow[*appbehavior.OnboardingChatRequest, *appbehavior.OnboardingChatResponse, string]
}

func NewProfilingAgent(g *genkit.Genkit) *ProfilingAgent {
	flow := genkit.DefineStreamingFlow(g, "onboardingProfilingFlow",
		func(ctx context.Context, input *appbehavior.OnboardingChatRequest, sendChunk core.StreamCallback[string]) (*appbehavior.OnboardingChatResponse, error) {
			// Count AI messages in history to determine turn number
			aiTurnCount := 0
			for _, msg := range input.History {
				if msg.Role == "assistant" {
					aiTurnCount++
				}
			}
			nextTurn := aiTurnCount + 1

			if nextTurn > maxProfilingTurns {
				return nil, fmt.Errorf("maximum profiling turns (%d) exceeded", maxProfilingTurns)
			}

			systemPrompt := buildProfilingSystemPrompt(input, nextTurn)

			// Build conversation messages
			var messages []*ai.Message
			messages = append(messages, ai.NewSystemMessage(ai.NewTextPart(systemPrompt)))

			for _, msg := range input.History {
				if msg.Role == "user" {
					messages = append(messages, ai.NewUserMessage(ai.NewTextPart(msg.Content)))
				} else {
					messages = append(messages, ai.NewModelMessage(ai.NewTextPart(msg.Content)))
				}
			}

			// Add the new user message (empty for first turn when AI initiates)
			if input.UserMessage != "" {
				messages = append(messages, ai.NewUserMessage(ai.NewTextPart(input.UserMessage)))
			}

			hasUserMessage := input.UserMessage != ""
			if !hasUserMessage {
				for _, msg := range input.History {
					if msg.Role == "user" && msg.Content != "" {
						hasUserMessage = true
						break
					}
				}
			}
			if !hasUserMessage {
				messages = append(messages, ai.NewUserMessage(ai.NewTextPart("Please start the conversation.")))
			}

			// Build updated history
			newHistory := make([]appbehavior.OnboardingChatMessage, len(input.History))
			copy(newHistory, input.History)
			if input.UserMessage != "" {
				newHistory = append(newHistory, appbehavior.OnboardingChatMessage{
					Role:    "user",
					Content: input.UserMessage,
				})
			}

			if nextTurn == maxProfilingTurns {
				// Final turn: generate structured summary (no streaming — structured JSON output)
				result, _, err := genkit.GenerateData[profilingSummary](ctx, g,
					ai.WithMessages(messages...),
				)
				if err != nil {
					return nil, fmt.Errorf("generate profiling summary: %w", err)
				}

				summaryJSON, err := json.Marshal(result)
				if err != nil {
					return nil, fmt.Errorf("marshal profiling summary: %w", err)
				}

				aiMessage := fmt.Sprintf("Thanks for sharing all of that with me. I have a good understanding of where you're at now. Based on our conversation, I'd recommend the \"%s\" delivery mode for your daily plans.", readableDeliveryMode(result.RecommendedDeliveryMode))

				newHistory = append(newHistory, appbehavior.OnboardingChatMessage{
					Role:    "assistant",
					Content: aiMessage,
				})

				return &appbehavior.OnboardingChatResponse{
					AIMessage:                 aiMessage,
					History:                   newHistory,
					TurnNumber:                nextTurn,
					IsComplete:                true,
					Summary:                   summaryJSON,
					RecommendedDeliveryMode:   result.RecommendedDeliveryMode,
					RecommendedEmotionalState: result.RecommendedEmotionalState,
				}, nil
			}

			// Non-final turn: stream free-text response chunk by chunk
			stream := genkit.GenerateStream(ctx, g, ai.WithMessages(messages...))

			var aiMessage string
			for result, err := range stream {
				if err != nil {
					return nil, fmt.Errorf("generate profiling response: %w", err)
				}
				if result.Done {
					aiMessage = result.Response.Text()
					break
				}
				if err := sendChunk(ctx, result.Chunk.Text()); err != nil {
					return nil, fmt.Errorf("send chunk: %w", err)
				}
			}

			newHistory = append(newHistory, appbehavior.OnboardingChatMessage{
				Role:    "assistant",
				Content: aiMessage,
			})

			return &appbehavior.OnboardingChatResponse{
				AIMessage:  aiMessage,
				History:    newHistory,
				TurnNumber: nextTurn,
				IsComplete: false,
			}, nil
		})

	return &ProfilingAgent{flow: flow}
}

// Chat runs the profiling flow without streaming and returns the complete response.
func (a *ProfilingAgent) Chat(ctx context.Context, req appbehavior.OnboardingChatRequest) (*appbehavior.OnboardingChatResponse, error) {
	return a.flow.Run(ctx, &req)
}

// StreamChat runs the profiling flow and returns a channel of streaming chunks.
// The channel is closed after the final chunk (Done == true) or on error.
func (a *ProfilingAgent) StreamChat(ctx context.Context, req appbehavior.OnboardingChatRequest) (<-chan appbehavior.OnboardingChatChunk, error) {
	out := make(chan appbehavior.OnboardingChatChunk, 16)
	go func() {
		defer close(out)
		for result, err := range a.flow.Stream(ctx, &req) {
			if err != nil {
				out <- appbehavior.OnboardingChatChunk{Err: err}
				return
			}
			if result.Done {
				out <- appbehavior.OnboardingChatChunk{Done: true, Response: result.Output}
			} else {
				out <- appbehavior.OnboardingChatChunk{Content: result.Stream}
			}
		}
	}()

	return out, nil
}

func buildProfilingSystemPrompt(input *appbehavior.OnboardingChatRequest, turn int) string {
	base := `You are a warm, empathetic behavioral coach having a brief getting-to-know-you conversation during onboarding. Your goal is to understand the user's life situation, emotional state, and what kind of support they need.

Context from earlier onboarding steps:
- Goals: %s
- Constraints: %s
- Self-assessment: %s

Guidelines:
- Be warm, conversational, and non-judgmental
- Ask focused, open-ended questions (one at a time)
- Listen for signs of burnout, anxiety, or feeling stuck
- Keep responses concise (2-3 sentences max, then ask your question)
- Do NOT give advice yet — just understand their situation
`

	prompt := fmt.Sprintf(base,
		string(input.Goals),
		string(input.Constraints),
		string(input.PsychologicalState),
	)

	switch turn {
	case 1:
		prompt += `
This is the FIRST message. The user hasn't said anything yet. Greet them warmly and ask an opening question about what's going on in their life right now that led them to seek behavioral coaching. Keep it casual and inviting.`
	case 2:
		prompt += `
This is turn 2 of 5. Based on their response, dig deeper into their emotional state. Are they feeling overwhelmed? Burned out? Stuck in a rut? Ask a follow-up that helps you understand their energy and motivation levels.`
	case 3:
		prompt += `
This is turn 3 of 5. Now explore their relationship with structure and routine. Do they thrive with strict schedules, or does too much structure feel suffocating? Have they tried planning tools before? What worked or didn't?`
	case 4:
		prompt += `
This is turn 4 of 5. Ask about what a realistic "good day" looks like for them. What are the small wins that would make them feel like they're making progress? This is the last question before you wrap up.`
	case 5:
		prompt += `
This is the FINAL turn (5 of 5). Based on the entire conversation, generate a structured summary.

You must output valid JSON with these fields:
- summary: A 2-3 sentence summary of the user's situation
- key_insights: An array of 3-5 key insights about the user
- recommended_delivery_mode: One of "strict_schedule", "flexible_list", or "gentle_structure"
  - Use "strict_schedule" if the user thrives with clear time blocks and accountability
  - Use "flexible_list" if the user wants guidance but values flexibility
  - Use "gentle_structure" if the user shows signs of burnout, low energy, or overwhelm
- recommended_emotional_state: One of "burned_out", "anxious", "stuck", or "" (empty if none clearly applies)
  - "burned_out": depleted energy, cynicism, feeling of meaninglessness
  - "anxious": worry, restlessness, difficulty relaxing
  - "stuck": lack of direction, procrastination, feeling trapped`
	}

	return prompt
}

func readableDeliveryMode(mode string) string {
	switch mode {
	case "strict_schedule":
		return "Strict Schedule"
	case "flexible_list":
		return "Flexible List"
	case "gentle_structure":
		return "Gentle Structure"
	default:
		return mode
	}
}
