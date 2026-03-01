package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mounis-bhat/starter/internal/app/chat"
)

type ChatHandler struct {
	service *chat.Service
}

func NewChatHandler(service *chat.Service) *ChatHandler {
	return &ChatHandler{service: service}
}

// HandleListMessages returns chat history for the authenticated user.
// @Summary      List chat messages
// @Description  Returns the chat history (oldest first)
// @Tags         chat
// @Produce      json
// @Param        limit query int false "Max messages to return" default(50)
// @Success      200  {array}   chat.ChatMessageResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /chat/messages [get]
func (h *ChatHandler) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	msgs, err := h.service.ListMessages(r.Context(), user.ID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load messages"})
		return
	}

	writeJSON(w, http.StatusOK, msgs)
}

// HandleSendMessage sends a user message and streams the AI response via SSE.
// Falls back to JSON if the client does not request SSE.
// @Summary      Send a chat message
// @Description  Sends a message and returns the AI response (SSE if requested)
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        request body chat.SendMessageRequest true "User message"
// @Success      200  {object}  chat.SendMessageResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /chat/messages [post]
func (h *ChatHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req chat.SendMessageRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	if r.Header.Get("Accept") == "text/event-stream" {
		h.streamMessage(w, r, user.ID, req.Content)
		return
	}

	// Non-streaming: collect all chunks then return the final response
	chunks, err := h.service.StreamMessage(r.Context(), user.ID, req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to process message"})
		return
	}

	var finalResp *chat.SendMessageResponse
	for chunk := range chunks {
		if chunk.Err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI generation failed"})
			return
		}
		if chunk.Done {
			finalResp = chunk.Response
		}
	}

	if finalResp == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no response generated"})
		return
	}

	writeJSON(w, http.StatusOK, finalResp)
}

// HandleClearHistory deletes all chat messages for the authenticated user.
// @Summary      Clear chat history
// @Description  Deletes all chat messages for the user
// @Tags         chat
// @Success      204
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /chat/messages [delete]
func (h *ChatHandler) HandleClearHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.service.ClearHistory(r.Context(), user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to clear history"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) streamMessage(w http.ResponseWriter, r *http.Request, userID, content string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	writeEvent := func(v any) {
		data, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	chunks, err := h.service.StreamMessage(r.Context(), userID, content)
	if err != nil {
		writeEvent(map[string]string{"error": "failed to start chat stream"})
		return
	}

	for chunk := range chunks {
		if chunk.Err != nil {
			writeEvent(map[string]string{"error": chunk.Err.Error()})
			return
		}
		writeEvent(chunk)
	}
}
