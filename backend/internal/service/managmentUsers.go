package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Управление пользователями (только admin)
type AdminUserResponse struct {
	Id        int       `json:"id"`
	Login     string    `json:"login"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) GetAllUsers(ctx context.Context) ([]AdminUserResponse, error) {
	usersData, err := s.store.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	var users []AdminUserResponse
	for _, u := range usersData {
		users = append(users, AdminUserResponse{
			Id:        u["id"].(int),
			Login:     u["login"].(string),
			Role:      u["role"].(string),
			CreatedAt: u["created_at"].(time.Time),
		})
	}

	return users, nil
}

func (s *Service) AdminUpdateRole(ctx context.Context, userID int, role string) error {
	if role != "admin" && role != "developer" && role != "qa" {
		return errors.New("invalid_role")
	}

	slog.Info("admin is changing user role", "target_user_id", userID, "new_role", role)

	return s.store.AdminUpdateUserRole(ctx, userID, role)
}

func (s *Service) AdminDeleteUser(ctx context.Context, adminID, targetID int) error {
	if adminID == targetID {
		slog.Warn("admin tried to delete themselves", "admin_id", adminID)
		return errors.New("cannot_delete_self")
	}

	slog.Info("admin is deleting user", "admin_id", adminID, "target_user_id", targetID)

	return s.store.AdminDeleteUser(ctx, targetID)
}
