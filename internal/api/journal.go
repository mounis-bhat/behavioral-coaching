package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mounis-bhat/starter/internal/app/journal"
)

type JournalHandler struct {
	service *journal.Service
}

func NewJournalHandler(service *journal.Service) *JournalHandler {
	return &JournalHandler{service: service}
}

// HandleUpsertEntry creates or updates today's journal entry
// @Summary      Upsert journal entry
// @Description  Creates or updates the journal entry for today
// @Tags         journal
// @Accept       json
// @Produce      json
// @Param        request body journal.UpsertEntryInput true "Journal entry input"
// @Success      200  {object}  journal.JournalEntryResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /journal/entries [post]
func (h *JournalHandler) HandleUpsertEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var input journal.UpsertEntryInput
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if input.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}
	if input.MoodScore < 1 || input.MoodScore > 10 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mood_score must be between 1 and 10"})
		return
	}

	entry, err := h.service.UpsertEntry(r.Context(), user.ID, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save journal entry"})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// HandleGetToday returns today's journal entry
// @Summary      Get today's journal entry
// @Description  Returns the journal entry for today, or 404 if none exists
// @Tags         journal
// @Produce      json
// @Success      200  {object}  journal.JournalEntryResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /journal/entries/today [get]
func (h *JournalHandler) HandleGetToday(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	entry, err := h.service.GetTodaysEntry(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, journal.ErrEntryNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no entry for today"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get today's entry"})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// HandleListEntries returns paginated journal entries
// @Summary      List journal entries
// @Description  Returns a paginated list of journal entries, newest first
// @Tags         journal
// @Produce      json
// @Param        limit  query int false "Number of entries to return" default(20)
// @Param        offset query int false "Number of entries to skip" default(0)
// @Success      200  {array}   journal.JournalEntryResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /journal/entries [get]
func (h *JournalHandler) HandleListEntries(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	entries, err := h.service.ListEntries(r.Context(), user.ID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list journal entries"})
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// HandleGetMoodTrend returns the 14-day mood trend
// @Summary      Get mood trend
// @Description  Returns the 14-day mood trend including average score and direction
// @Tags         journal
// @Produce      json
// @Success      200  {object}  journal.MoodTrendResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /journal/mood-trend [get]
func (h *JournalHandler) HandleGetMoodTrend(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	trend, err := h.service.GetMoodTrend(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get mood trend"})
		return
	}

	writeJSON(w, http.StatusOK, trend)
}
