package handlers

import (
	"encoding/json"
	"net/http"

	"main/di"
)

type HealthHandler struct {
	deps *di.DI
}

func NewHealthHandler(deps *di.DI) *HealthHandler {
	return &HealthHandler{deps: deps}
}

func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   h.deps.Clock.Now().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
