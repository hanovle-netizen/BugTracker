package service

import (
	"TaskTracker/internal/store/postgres"
	"context"
	"errors"
	"strings"
)

func (s *Service) checkTaskProjectAccess(ctx context.Context, taskID, userID int) error {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		if _, err := s.store.GetUserRoleInProject(ctx, projectID, userID); err != nil {
			return errors.New("access_denied")
		}
	}
	return nil
}

func (s *Service) checkBugAccess(ctx context.Context, bugID, userID int) error {
	taskID, err := s.store.GetBugTaskID(ctx, bugID)
	if err != nil {
		if err.Error() == "bug_not_found" {
			return errors.New("bug_not_found")
		}
		return err
	}
	return s.checkTaskProjectAccess(ctx, taskID, userID)
}

func (s *Service) normalizeBugFields(ctx context.Context, req map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range req {
		out[k] = v
	}

	if login, ok := out["assigned_to_email"].(string); ok && strings.TrimSpace(login) != "" {
		if userID, err := s.store.GetUserIDByLogin(ctx, strings.TrimSpace(login)); err == nil {
			out["assigned_to_fk"] = userID
		}
		delete(out, "assigned_to_email")
	}
	if v, ok := out["assigned_to"]; ok {
		out["assigned_to_fk"] = v
		delete(out, "assigned_to")
	}
	if v, ok := out["passed_by"]; ok {
		out["passed_by_fk"] = v
		delete(out, "passed_by")
	}
	if v, ok := out["accepted_by"]; ok {
		out["accepted_by_fk"] = v
		delete(out, "accepted_by")
	}

	delete(out, "id")
	delete(out, "id_pk")
	delete(out, "task_id")
	delete(out, "created_by")
	delete(out, "created_at")
	delete(out, "updated_at")
	return out
}

func formatBugsAPI(bugs []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(bugs))
	for _, b := range bugs {
		out = append(out, postgres.FormatBugAPI(b))
	}
	return out
}

func (s *Service) GetBugsByTask(ctx context.Context, taskID, userID int) ([]map[string]interface{}, error) {
	if err := s.checkTaskProjectAccess(ctx, taskID, userID); err != nil {
		return nil, err
	}
	bugs, err := s.store.GetBugsByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return formatBugsAPI(bugs), nil
}

func (s *Service) CreateBug(ctx context.Context, taskID, userID int, req map[string]interface{}) (int, error) {
	if err := s.checkTaskProjectAccess(ctx, taskID, userID); err != nil {
		return 0, err
	}

	fields := s.normalizeBugFields(ctx, req)
	return s.store.CreateBug(ctx, taskID, userID, fields)
}

func (s *Service) UpdateBug(ctx context.Context, bugID, userID int, req map[string]interface{}) (map[string]interface{}, error) {
	if err := s.checkBugAccess(ctx, bugID, userID); err != nil {
		return nil, err
	}
	fields := s.normalizeBugFields(ctx, req)
	if err := s.store.UpdateBug(ctx, bugID, fields); err != nil {
		return nil, err
	}
	bug, err := s.store.GetBugByID(ctx, bugID)
	if err != nil {
		return nil, err
	}
	return postgres.FormatBugAPI(bug), nil
}

func (s *Service) DeleteBug(ctx context.Context, bugID, userID int) error {
	if err := s.checkBugAccess(ctx, bugID, userID); err != nil {
		return err
	}
	return s.store.DeleteBug(ctx, bugID)
}

func (s *Service) GetBugTaskID(ctx context.Context, bugID int) (int, error) {
	return s.store.GetBugTaskID(ctx, bugID)
}

func (s *Service) GetBugComments(ctx context.Context, bugID, userID int) ([]map[string]interface{}, error) {
	if err := s.checkBugAccess(ctx, bugID, userID); err != nil {
		return nil, err
	}
	return s.store.GetBugComments(ctx, bugID)
}

func (s *Service) AddBugComment(ctx context.Context, bugID, userID int, body string) (int, error) {
	if err := s.checkBugAccess(ctx, bugID, userID); err != nil {
		return 0, err
	}
	if strings.TrimSpace(body) == "" {
		return 0, errors.New("body_required")
	}
	return s.store.CreateBugComment(ctx, bugID, userID, body)
}
