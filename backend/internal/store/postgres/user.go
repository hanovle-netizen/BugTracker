package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Аутентификация

func (s *Store) CreateUser(ctx context.Context, email, hashedPwd, role string) (int, error) {
	var id int
	query := `INSERT INTO "user" (login, password, role) VALUES ($1, $2, $3) RETURNING id_pk`

	slog.Info("attempting to create user", "login", email, "role", role)

	err := s.conn.QueryRow(ctx, query, email, hashedPwd, role).Scan(&id)

	if err != nil {

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			slog.Warn("user registration failed: login already exists", "login", email)
			return 0, fmt.Errorf("user_exists")
		}

		slog.Error("database error during user creation",
			"error", err,
			"login", email,
		)
		return 0, err
	}

	slog.Info("user created successfully", "id", id, "login", email)
	return id, nil
}

func (s *Store) GetUserByLogin(ctx context.Context, login string) (int, string, string, int, error) {
	var id int
	var hashedPwd, role string
	var ver int

	query := `SELECT id_pk, password, role, ver FROM "user" WHERE login = $1`

	slog.Info("fetching user by login", "login", login)

	err := s.conn.QueryRow(ctx, query, login).Scan(&id, &hashedPwd, &role, &ver)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Warn("user not found", "login", login)
			return 0, "", "", 0, fmt.Errorf("user_not_found")
		}

		slog.Error("database error while fetching user", "error", err, "login", login)
		return 0, "", "", 0, err
	}

	return id, hashedPwd, role, ver, nil
}

// Пользователь (текущий)
func (s *Store) GetUserByID(ctx context.Context, id int) (id_pk int, login, hashedPwd, role string, ver int, err error) {
	query := `SELECT id_pk, login, password, role, ver FROM "user" WHERE id_pk = $1`

	err = s.conn.QueryRow(ctx, query, id).Scan(&id_pk, &login, &hashedPwd, &role, &ver)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, "", "", "", 0, fmt.Errorf("user_not_found")
		}
		slog.Error("database error fetching user", "error", err, "id", id)
		return 0, "", "", "", 0, err
	}

	return
}

func (s *Store) UpdateUserLogin(ctx context.Context, userID int, newLogin string) error {
	query := `UPDATE "user" SET login = $1 WHERE id_pk = $2`

	_, err := s.conn.Exec(ctx, query, newLogin, userID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("login_exists")
		}
		return err
	}
	return nil
}

func (s *Store) UpdateUserPassword(ctx context.Context, userID int, newHashedPwd string) error {
	query := `UPDATE "user" SET password = $1 WHERE id_pk = $2`

	_, err := s.conn.Exec(ctx, query, newHashedPwd, userID)
	if err != nil {
		slog.Error("database error updating password", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (s *Store) IncrementUserVersion(ctx context.Context, userID int) error {
	query := `UPDATE "user" SET ver = ver + 1 WHERE id_pk = $1`

	_, err := s.conn.Exec(ctx, query, userID)
	if err != nil {
		slog.Error("database error incrementing user version", "error", err, "user_id", userID)
		return err
	}
	return nil
}

func (s *Store) ReplaceTaskTags(ctx context.Context, taskID int, tags []string) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM bug_tag WHERE bug_id_fk = $1`, taskID)
	if err != nil {
		return err
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			_, err = tx.Exec(ctx, `INSERT INTO bug_tag (bug_id_fk, tag) VALUES ($1, $2)`, taskID, tag)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
