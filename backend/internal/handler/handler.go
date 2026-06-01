package handler

import (
	"TaskTracker/internal/service"
	"encoding/json"
	"net/http"
)

type Handler struct {
	svc *service.Service
}

func NewUserHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
