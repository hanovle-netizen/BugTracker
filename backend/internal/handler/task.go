package handler

import (
	"TaskTracker/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func fileURL(r *http.Request, objectKey string) string {
	if strings.EqualFold(os.Getenv("PHOTO_STORAGE"), "minio") {
		publicURL := strings.TrimRight(os.Getenv("MINIO_PUBLIC_URL"), "/")
		bucket := os.Getenv("MINIO_BUCKET")
		if bucket == "" {
			bucket = "tasktracker-photos"
		}
		if publicURL != "" {
			return fmt.Sprintf("%s/%s/%s", publicURL, bucket, path.Clean("/"+objectKey)[1:])
		}
		endpoint := strings.TrimRight(os.Getenv("MINIO_ENDPOINT"), "/")
		useSSL := strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true")
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		if endpoint != "" {
			return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, path.Clean("/"+objectKey)[1:])
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/uploads/%s", scheme, r.Host, objectKey)
}

func parseEntityID(vars map[string]string) int {
	for _, k := range []string{"id", "task_id", "bug_id"} {
		if v, ok := vars[k]; ok {
			if id, err := strconv.Atoi(v); err == nil {
				return id
			}
		}
	}
	return 0
}

// Задачи
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	projectID, err := strconv.Atoi(r.URL.Query().Get("project_id"))
	if err != nil {
		h.sendError(w, "project_id_required", "bad_request", http.StatusBadRequest)
		return
	}

	tasks, err := h.svc.GetProjectTasks(r.Context(), projectID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	var req service.CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	id, err := h.svc.CreateTask(r.Context(), userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "title_and_project_id_required" {
			status = http.StatusBadRequest
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateTask(r.Context(), taskID, userID, req); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		if strings.Contains(code, "violates check constraint") {
			status, code = http.StatusBadRequest, "invalid_enum_value"
		} else if code == "access_denied" {
			status = http.StatusForbidden
		} else if code == "task_not_found" {
			status = http.StatusNotFound
		}

		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))

	if err := h.svc.DeleteTask(r.Context(), taskID, userID); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["id"])

	r.ParseMultipartForm(16 << 20)

	file, header, err := r.FormFile("photo")
	if err != nil {
		h.sendError(w, "photo_field_required", "bad_request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, "read_error", "internal_error", http.StatusInternalServerError)
		return
	}

	err = h.svc.UploadTaskPhoto(r.Context(), taskID, userID, header.Filename, header.Size, data)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()
		switch code {
		case "file_too_large", "invalid_format":
			status = http.StatusBadRequest
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetTaskPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["id"])

	photoData, err := h.svc.GetTaskPhoto(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "no_photo" || err.Error() == "task_not_found" {
			status = http.StatusNotFound
		} else if err.Error() == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	contentType := http.DetectContentType(photoData)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(photoData)
}

func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))

	comments, err := h.svc.GetComments(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	id, err := h.svc.AddComment(r.Context(), taskID, userID, req.Body)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		case "comment_body_required":
			status = http.StatusBadRequest
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) GetTaskAudit(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["id"])

	audit, err := h.svc.GetTaskAudit(r.Context(), taskID, userID)
	if err != nil {

		h.sendError(w, err.Error(), "error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(audit)
}

func (h *Handler) GetTaskRelations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	taskID, _ := strconv.Atoi(vars["id"])

	relations, err := h.svc.GetTaskRelations(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()
		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relations)
}

func (h *Handler) CreateTaskRelation(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	taskID, _ := strconv.Atoi(vars["id"])

	var req struct {
		RelatedTaskID int    `json:"related_task_id"`
		Type          string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	id, err := h.svc.AddTaskRelation(r.Context(), taskID, userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied", "access_denied_to_related_project":
			status = http.StatusForbidden
		case "task_not_found", "related_task_not_found":
			status = http.StatusNotFound
		case "cannot_relate_to_self", "invalid_type":
			status = http.StatusBadRequest
		case "relation_exists":
			status = http.StatusConflict
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) DeleteRelation(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	relID, _ := strconv.Atoi(vars["rel_id"])

	if err := h.svc.DeleteRelation(r.Context(), relID, userID); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()
		if code == "relation_not_found" {
			status = http.StatusNotFound
		} else if code == "access_denied" {
			status = http.StatusForbidden
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetTaskTags(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	taskID, _ := strconv.Atoi(vars["id"])

	tags, err := h.svc.GetTaskTags(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		code := err.Error()

		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *Handler) ReplaceTaskTags(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var tags []string
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		h.sendError(w, "invalid_json_array", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.ReplaceTags(r.Context(), taskID, userID, tags); err != nil {
		status := http.StatusInternalServerError
		code := err.Error()
		switch code {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, code, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.GetTemplates(r.Context())
	if err != nil {
		h.sendError(w, "failed to get templates", "internal_error", http.StatusInternalServerError)
		return
	}

	if templates == nil {
		templates = []service.TemplateResponse{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}

	id, err := h.svc.CreateTemplate(r.Context(), userID, req.Name, req.Body)
	if err != nil {
		h.sendError(w, err.Error(), "internal_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendError(w, "invalid_id", "bad_request", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteTemplate(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "template_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	projectID, err := strconv.Atoi(r.URL.Query().Get("project_id"))
	if err != nil {
		h.sendError(w, "project_id_required", "bad_request", http.StatusBadRequest)
		return
	}

	stats, err := h.svc.GetProjectTasks(r.Context(), projectID, userID)
	if err != nil {
		h.sendError(w, "failed to fetch stats", "internal_error", http.StatusInternalServerError)
		return
	}

	agg := map[string]int{}
	for _, t := range stats {
		status, _ := t["status"].(*string)
		key := "No Status"
		if status != nil {
			key = *status
		}
		agg[key]++
	}
	resp := make([]map[string]interface{}, 0, len(agg))
	for k, v := range agg {
		resp = append(resp, map[string]interface{}{"status": k, "count": v})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetTaskStats(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	taskID, _ := strconv.Atoi(vars["task_id"])

	stats, err := h.svc.GetTaskAnalytics(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "task_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) GetTaskPhotos(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))
	photos, err := h.svc.GetEntityPhotos(r.Context(), "task", taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "task_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	out := make([]map[string]interface{}, 0, len(photos))
	for _, p := range photos {
		out = append(out, map[string]interface{}{"id": p["id"], "url": p["url"]})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *Handler) AddTaskPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID := parseEntityID(mux.Vars(r))
	r.ParseMultipartForm(16 << 20)
	file, header, err := r.FormFile("photo")
	if err != nil {
		h.sendError(w, "photo_field_required", "bad_request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, "read_error", "internal_error", http.StatusInternalServerError)
		return
	}
	id, url, err := h.svc.AddEntityPhoto(r.Context(), "task", taskID, userID, header.Filename, header.Size, data)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "file_too_large", "invalid_format":
			status = http.StatusBadRequest
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "url": url})
}

func (h *Handler) DeleteTaskPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	taskID := parseEntityID(vars)
	photoID, _ := strconv.Atoi(vars["photo_id"])
	if err := h.svc.DeleteEntityPhoto(r.Context(), "task", taskID, photoID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "photo_not_found" || err.Error() == "task_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetBug(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["task_id"])
	bugs, err := h.svc.GetBugsByTask(r.Context(), taskID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "task_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	if bugs == nil {
		bugs = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bugs)
}

func (h *Handler) UpdateBug(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID, _ := strconv.Atoi(mux.Vars(r)["bug_id"])
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}
	bug, err := h.svc.UpdateBug(r.Context(), bugID, userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bug)
}

func (h *Handler) DeleteBug(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID, _ := strconv.Atoi(mux.Vars(r)["bug_id"])
	if err := h.svc.DeleteBug(r.Context(), bugID, userID); err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *Handler) GetBugComments(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID := parseEntityID(mux.Vars(r))
	comments, err := h.svc.GetBugComments(r.Context(), bugID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	if comments == nil {
		comments = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (h *Handler) CreateBugComment(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID := parseEntityID(mux.Vars(r))
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}
	id, err := h.svc.AddBugComment(r.Context(), bugID, userID, req.Body)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "bug_not_found":
			status = http.StatusNotFound
		case "body_required":
			status = http.StatusBadRequest
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) CreateBug(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	taskID, _ := strconv.Atoi(mux.Vars(r)["task_id"])
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid_json", "bad_request", http.StatusBadRequest)
		return
	}
	id, err := h.svc.CreateBug(r.Context(), taskID, userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access_denied" {
			status = http.StatusForbidden
		} else if err.Error() == "task_not_found" {
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func (h *Handler) GetBugPhotos(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID := parseEntityID(mux.Vars(r))
	photos, err := h.svc.GetEntityPhotos(r.Context(), "bug", bugID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found", "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	out := make([]map[string]interface{}, 0, len(photos))
	for _, p := range photos {
		out = append(out, map[string]interface{}{"id": p["id"], "url": p["url"]})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *Handler) AddBugPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bugID := parseEntityID(mux.Vars(r))
	r.ParseMultipartForm(16 << 20)
	file, header, err := r.FormFile("photo")
	if err != nil {
		h.sendError(w, "photo_field_required", "bad_request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, "read_error", "internal_error", http.StatusInternalServerError)
		return
	}
	id, url, err := h.svc.AddEntityPhoto(r.Context(), "bug", bugID, userID, header.Filename, header.Size, data)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "file_too_large", "invalid_format":
			status = http.StatusBadRequest
		case "access_denied":
			status = http.StatusForbidden
		case "task_not_found", "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "url": url})
}

func (h *Handler) DeleteBugPhoto(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	bugID := parseEntityID(vars)
	photoID, _ := strconv.Atoi(vars["photo_id"])
	if err := h.svc.DeleteEntityPhoto(r.Context(), "bug", bugID, photoID, userID); err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "access_denied":
			status = http.StatusForbidden
		case "photo_not_found", "task_not_found", "bug_not_found":
			status = http.StatusNotFound
		}
		h.sendError(w, err.Error(), err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
