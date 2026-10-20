package handler

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// Health checks API availability.
// @Summary Health check
// @Description Проверка, что сервис жив.
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		HealthResponse{Status: "ok"},
	)
}
