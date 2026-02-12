package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mounis-bhat/starter/internal/app/behavior"
)

type BehaviorHandler struct {
	service *behavior.Service
}

func NewBehaviorHandler(service *behavior.Service) *BehaviorHandler {
	return &BehaviorHandler{service: service}
}

// HandleCreateProfile creates a behavior profile for the authenticated user
// @Summary      Create behavior profile
// @Description  Creates a behavior profile with goals, constraints, and psychological state
// @Tags         behavior
// @Accept       json
// @Produce      json
// @Param        request body behavior.CreateProfileInput true "Profile creation request"
// @Success      201  {object}  behavior.ProfileResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/profile [post]
func (h *BehaviorHandler) HandleCreateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var input behavior.CreateProfileInput
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	profile, err := h.service.CreateProfile(r.Context(), user.ID, input)
	if err != nil {
		if errors.Is(err, behavior.ErrProfileExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "profile already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create profile"})
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

// HandleGetProfile returns the behavior profile for the authenticated user
// @Summary      Get behavior profile
// @Description  Returns the behavior profile for the authenticated user
// @Tags         behavior
// @Produce      json
// @Success      200  {object}  behavior.ProfileResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/profile [get]
func (h *BehaviorHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	profile, err := h.service.GetProfile(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, behavior.ErrProfileNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// HandleUpdateProfile updates the behavior profile for the authenticated user
// @Summary      Update behavior profile
// @Description  Updates the behavior profile for the authenticated user
// @Tags         behavior
// @Accept       json
// @Produce      json
// @Param        request body behavior.UpdateProfileInput true "Profile update request"
// @Success      200  {object}  behavior.ProfileResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/profile [put]
func (h *BehaviorHandler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var input behavior.UpdateProfileInput
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), user.ID, input)
	if err != nil {
		if errors.Is(err, behavior.ErrProfileNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// HandleGeneratePlan generates a new daily plan for the authenticated user
// @Summary      Generate daily plan
// @Description  Generates a new daily plan based on the user's behavior profile
// @Tags         behavior
// @Produce      json
// @Success      200  {object}  behavior.PlanResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/plan/generate [post]
func (h *BehaviorHandler) HandleGeneratePlan(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	plan, err := h.service.GenerateDailyPlan(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate plan"})
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

// HandleGetTodaysPlan returns the active daily plan for today
// @Summary      Get today's plan
// @Description  Returns the active daily plan for today with all tasks
// @Tags         behavior
// @Produce      json
// @Success      200  {object}  behavior.PlanResponse
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/plan/today [get]
func (h *BehaviorHandler) HandleGetTodaysPlan(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	plan, err := h.service.GetTodaysPlan(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, behavior.ErrPlanNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no plan for today"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get plan"})
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

// HandleLogExecution records execution status for a plan task
// @Summary      Log task execution
// @Description  Records execution status for a specific plan task
// @Tags         behavior
// @Accept       json
// @Produce      json
// @Param        taskID path string true "Plan task ID" format(uuid)
// @Param        request body behavior.LogExecutionInput true "Execution log request"
// @Success      200  {object}  behavior.ExecutionLogResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/tasks/{taskID}/log [post]
func (h *BehaviorHandler) HandleLogExecution(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task ID is required"})
		return
	}

	var input behavior.LogExecutionInput
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	log, err := h.service.LogExecution(r.Context(), user.ID, taskID, input)
	if err != nil {
		if errors.Is(err, behavior.ErrTaskNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to log execution"})
		return
	}

	writeJSON(w, http.StatusOK, log)
}

// HandleGetAdherence returns adherence metrics for the authenticated user
// @Summary      Get adherence metrics
// @Description  Returns the current adherence metrics for the authenticated user
// @Tags         behavior
// @Produce      json
// @Success      200  {object}  behavior.AdherenceResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/adherence [get]
func (h *BehaviorHandler) HandleGetAdherence(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	adherence, err := h.service.RecomputeAdherence(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get adherence"})
		return
	}

	writeJSON(w, http.StatusOK, adherence)
}

// HandleGetAdaptations returns adaptation logs for the authenticated user
// @Summary      Get adaptation logs
// @Description  Returns recent adaptation logs showing plan difficulty adjustments
// @Tags         behavior
// @Produce      json
// @Param        limit query int false "Number of logs to return" default(10)
// @Success      200  {array}   behavior.AdaptationLogResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/adaptations [get]
func (h *BehaviorHandler) HandleGetAdaptations(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.service.GetAdaptationLogs(r.Context(), user.ID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get adaptations"})
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// HandleGetTodaysExecutionLogs returns today's execution logs for the authenticated user
// @Summary      Get today's execution logs
// @Description  Returns execution logs for today's active plan
// @Tags         behavior
// @Produce      json
// @Success      200  {array}   behavior.ExecutionLogResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /behavior/plan/today/logs [get]
func (h *BehaviorHandler) HandleGetTodaysExecutionLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	logs, err := h.service.GetTodaysExecutionLogs(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get execution logs"})
		return
	}

	writeJSON(w, http.StatusOK, logs)
}
