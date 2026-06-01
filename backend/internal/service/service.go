package service

import (
	"TaskTracker/internal/store/postgres"
	"context"
	"errors"
)

type Service struct {
	store     *postgres.Store
	jwtSecret string
}

func NewService(store *postgres.Store, jwtSecret string) *Service {
	return &Service{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) GetTaskAudit(ctx context.Context, taskID, userID int) ([]map[string]interface{}, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTaskAudit(ctx, taskID)
}

type StatResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}
