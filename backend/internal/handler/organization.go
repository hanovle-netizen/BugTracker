package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Группы (Организации)
func (h *Handler) GetMyOrganizations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	orgs, err := h.svc.GetMyOrganizations(r.Context(), userID)
	if err != nil {
		h.sendError(w, "failed to get organizations", "internal_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orgs)
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	res, err := h.svc.CreateOrg(r.Context(), req.Name, userID)
	if err != nil {
		h.sendError(w, err.Error(), "internal_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetOrgMembers(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	vars := mux.Vars(r)
	orgID, _ := strconv.Atoi(vars["id"])

	members, err := h.svc.GetOrgMembers(r.Context(), orgID, userID)
	if err != nil {
		if err.Error() == "access_denied" {
			h.sendError(w, "Forbidden", "access_denied", http.StatusForbidden)
		} else {
			h.sendError(w, "Error", "internal_error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *Handler) AddOrgMember(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	orgID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var req struct {
		Login string `json:"login"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AddMemberToOrg(r.Context(), orgID, requesterID, req.Login, req.Role); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "user_not_found":
			status = http.StatusNotFound
		case "already_member":
			status = http.StatusConflict
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) UpdateOrgMemberRole(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	orgID, _ := strconv.Atoi(vars["id"])
	targetID, _ := strconv.Atoi(vars["userId"])

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateMemberRole(r.Context(), orgID, requesterID, targetID, req.Role); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied", "insufficient_permissions":
			status = http.StatusForbidden
		case "member_not_found":
			status = http.StatusNotFound
		case "invalid_role":
			status = http.StatusBadRequest
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) RemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	orgID, _ := strconv.Atoi(vars["id"])
	targetID, _ := strconv.Atoi(vars["userId"])

	if err := h.svc.RemoveMember(r.Context(), orgID, requesterID, targetID); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied", "insufficient_permissions":
			status = http.StatusForbidden
		case "member_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
