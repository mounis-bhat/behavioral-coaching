package api

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents the health check response
// @Description Health check response
type HealthResponse struct {
	Status string `json:"status" example:"ok" validate:"required"`
}

// handleHealth returns the health status of the API
// @Summary      Health check
// @Description  Returns the health status of the API
// @Tags         system
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}
