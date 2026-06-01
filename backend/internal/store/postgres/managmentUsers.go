package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Управление пользователями (только admin)
func (s *Store) GetAllUsers(ctx context.Context) ([]map[string]interface{}, error) {
	query := `SELECT id_pk, login, role, created_at FROM "user" ORDER BY id_pk DESC`

	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		slog.Error("admin get users query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var login, role string
		var createdAt time.Time
		if err := rows.Scan(&id, &login, &role, &createdAt); err != nil {
			return nil, err
		}
		users = append(users, map[string]interface{}{
			"id":         id,
			"login":      login,
			"role":       role,
			"created_at": createdAt,
		})
	}
	return users, nil
}

func (s *Store) AdminUpdateUserRole(ctx context.Context, userID int, newRole string) error {
	query := `UPDATE "user" SET role = $1 WHERE id_pk = $2`

	result, err := s.conn.Exec(ctx, query, newRole, userID)
	if err != nil {
		slog.Error("admin: failed to update user role", "error", err, "user_id", userID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user_not_found")
	}

	return nil
}

func (s *Store) AdminDeleteUser(ctx context.Context, userID int) error {
	query := `DELETE FROM "user" WHERE id_pk = $1`

	result, err := s.conn.Exec(ctx, query, userID)
	if err != nil {
		slog.Error("admin: failed to delete user", "error", err, "user_id", userID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user_not_found")
	}

	return nil
}
