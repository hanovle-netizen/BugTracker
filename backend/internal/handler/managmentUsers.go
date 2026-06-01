package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Управление пользователями (только admin)
func (h *Handler) AdminGetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.GetAllUsers(r.Context())
	if err != nil {
		h.sendError(w, "failed to fetch users", "internal_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) AdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendError(w, "invalid_id", "bad_request", http.StatusBadRequest)
		return
	}

	if requesterID == id {
		h.sendError(w, "not allowed", "not_allowed", http.StatusForbidden)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AdminUpdateRole(r.Context(), id, req.Role); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		if code == "user_not_found" {
			status = http.StatusNotFound
		} else if code == "invalid_role" {
			status = http.StatusBadRequest
		}

		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("user_id").(int)

	vars := mux.Vars(r)
	targetID, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendError(w, "invalid_id", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AdminDeleteUser(r.Context(), adminID, targetID); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "cannot_delete_self":
			status = http.StatusForbidden
		case "user_not_found":
			status = http.StatusNotFound
		}

		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
