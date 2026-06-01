package handler

import (
	"TaskTracker/internal/service"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Аутентификация
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("failed to decode register request", "error", err, "remote_addr", r.RemoteAddr)
		h.sendError(w, "invalid_payload", "bad_request", http.StatusBadRequest)
		return
	}

	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "user_exists":
			status = http.StatusConflict
		case "password_required", "password_too_short", "password_too_weak":
			status = http.StatusBadRequest
		default:

			slog.Error("unexpected error during user registration", "error", err, "login", req.Login)
		}

		slog.Warn("sending error response to client", "code", code, "status", status, "login", req.Login)

		h.sendError(w, err.Error(), code, status)
		return
	}

	slog.Info("user registration handler completed", "user_id", res.Id, "login", req.Login)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("failed to encode success response", "error", err, "user_id", res.Id)
	}
}

func (h *Handler) sendError(w http.ResponseWriter, msg, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: msg,
		Code:  code,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("failed to decode login request", "error", err, "remote_addr", r.RemoteAddr)
		h.sendError(w, "invalid_payload", "bad_request", http.StatusBadRequest)
		return
	}

	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		if code == "invalid_credentials" {
			status = http.StatusUnauthorized
		} else {
			slog.Error("unexpected error during login", "error", err, "login", req.Login)
		}

		slog.Warn("login attempt failed", "login", req.Login, "code", code, "status", status)
		h.sendError(w, err.Error(), code, status)
		return
	}

	slog.Info("login successful", "user_id", res.Id, "login", req.Login)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// Пользователь (текущий)
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		slog.Error("user_id not found in context")
		h.sendError(w, "unauthorized", "unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		slog.Warn("get me failed", "user_id", userID, "error", err)
		h.sendError(w, "user not found", "not_found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req service.UpdateLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_payload", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateLogin(r.Context(), userID, req); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "invalid_password" {
			status = http.StatusForbidden
		} else if err.Error() == "login_exists" {
			status = http.StatusConflict
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req service.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_payload", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdatePassword(r.Context(), userID, req); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "invalid_password":
			status = http.StatusForbidden
		case "password_too_short", "password_too_weak":
			status = http.StatusBadRequest
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendError(w, "invalid_id", "bad_request", http.StatusBadRequest)
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AdminResetPassword(r.Context(), targetID, req.NewPassword); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "password_too_short", "password_too_weak":
			status = http.StatusBadRequest
		case "user_not_found":
			status = http.StatusNotFound
		}

		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		h.sendError(w, "unauthorized", "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.svc.LogoutAll(r.Context(), userID); err != nil {
		h.sendError(w, "failed to logout all devices", "internal_error", http.StatusInternalServerError)
		return
	}

	slog.Info("all sessions invalidated", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, "invalid_user_id", "bad_request", http.StatusBadRequest)
		return
	}

	res, err := h.svc.GetUserByID(r.Context(), id)
	if err != nil {
		if err.Error() == "user_not_found" {
			h.sendError(w, "user_not_found", "not_found", http.StatusNotFound)
		} else {
			h.sendError(w, "internal_server_error", "internal_error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
