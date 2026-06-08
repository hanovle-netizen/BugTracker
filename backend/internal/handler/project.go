package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Проекты
func (h *Handler) GetProjects(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	orgIDStr := r.URL.Query().Get("org_id")
	if orgIDStr == "" {
		h.sendError(w, "org_id_required", "bad_request", http.StatusBadRequest)
		return
	}

	orgID, _ := strconv.Atoi(orgIDStr)

	projects, err := h.svc.GetOrgProjects(r.Context(), orgID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req struct {
		OrgID int    `json:"org_id"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	res, err := h.svc.CreateProject(r.Context(), req.OrgID, userID, req.Name)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "name_required":
			status = http.StatusBadRequest
		default:
			code = "failed_to_create_project"
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetProjectMembers(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	projectID, _ := strconv.Atoi(vars["id"])

	members, err := h.svc.GetProjectMembers(r.Context(), projectID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *Handler) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	projectID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var req struct {
		Login    string `json:"login"`
		Role     string `json:"role"`
		Position string `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AddMemberToProject(r.Context(), projectID, requesterID, req.Login, req.Role, req.Position); err != nil {
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

func (h *Handler) UpdateProjectMember(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	projectID, _ := strconv.Atoi(vars["id"])
	targetID, _ := strconv.Atoi(vars["userId"])

	var req struct {
		Role     string `json:"role"`
		Position string `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateProjectMember(r.Context(), projectID, requesterID, targetID, req.Role, req.Position); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
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

func (h *Handler) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	requesterID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	projectID, _ := strconv.Atoi(vars["id"])
	targetID, _ := strconv.Atoi(vars["userId"])

	if err := h.svc.RemoveProjectMember(r.Context(), projectID, requesterID, targetID); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
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
