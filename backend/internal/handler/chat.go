package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Чат
func (h *Handler) CreateChatThread(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req struct {
		Scope     string `json:"scope"`
		OrgID     int    `json:"org_id"`
		ProjectID int    `json:"project_id"`
		Login     string `json:"login"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	var targetID int
	if req.Scope == "org" {
		targetID = req.OrgID
	} else if req.Scope == "project" {
		targetID = req.ProjectID
	}

	id, err := h.svc.GetOrCreateThread(r.Context(), req.Scope, req.Login, targetID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "user_not_found":
			status = http.StatusNotFound
		case "unsupported_scope", "cannot_chat_with_self":
			status = http.StatusBadRequest
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) GetThreads(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	query := r.URL.Query()
	scope := query.Get("scope")

	var targetID int
	if scope == "org" {
		targetID, _ = strconv.Atoi(query.Get("org_id"))
	} else if scope == "project" {
		targetID, _ = strconv.Atoi(query.Get("project_id"))
	}

	threads, err := h.svc.GetThreadsByScope(r.Context(), scope, targetID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "unsupported_scope" {
			status = http.StatusBadRequest
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threads)
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	threadID, _ := strconv.Atoi(vars["id"])

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	before, _ := strconv.Atoi(query.Get("before_id"))
	after, _ := strconv.Atoi(query.Get("after_id"))

	messages, err := h.svc.GetMessages(r.Context(), threadID, userID, limit, before, after)
	if err != nil {
		h.sendError(w, err.Error(), "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	vars := mux.Vars(r)
	threadID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	id, err := h.svc.SendMessage(r.Context(), threadID, userID, req.Body)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "message_body_required" {
			status = http.StatusBadRequest
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) EditChatMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	messageID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.EditMessage(r.Context(), messageID, userID, req.Body); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		if code == "message_not_found_or_access_denied" {
			status = http.StatusForbidden
		} else if code == "message_body_required" {
			status = http.StatusBadRequest
		}

		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) DeleteChatMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	messageID, _ := strconv.Atoi(vars["id"])

	if err := h.svc.DeleteMessage(r.Context(), messageID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "message_not_found_or_access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) MarkChatRead(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	threadID, _ := strconv.Atoi(vars["id"])

	if err := h.svc.ReadThread(r.Context(), threadID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetTyping(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	threadID, _ := strconv.Atoi(mux.Vars(r)["id"])

	logins, err := h.svc.GetTyping(r.Context(), threadID, userID)
	if err != nil {
		h.sendError(w, err.Error(), "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logins)
}

func (h *Handler) ReportTyping(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	threadID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var req struct {
		IsTyping bool `json:"is_typing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.ReportTyping(r.Context(), threadID, userID, req.IsTyping); err != nil {
		h.sendError(w, err.Error(), "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
