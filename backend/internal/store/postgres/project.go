package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
)

// Проекты
func (s *Store) GetProjectsByOrg(ctx context.Context, orgID, userID int, isTeacher bool) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if isTeacher {

		query = `
			SELECT p.id_pk, p.org_id_fk, p.name, COALESCE(m.role, 'admin') as role
			FROM projects p
			LEFT JOIN project_member m ON p.id_pk = m.project_id_fk AND m.user_id_fk = $2
			WHERE p.org_id_fk = $1
			ORDER BY p.name ASC`
		args = []interface{}{orgID, userID}
	} else {

		query = `
			SELECT p.id_pk, p.org_id_fk, p.name, m.role
			FROM projects p
			JOIN project_member m ON p.id_pk = m.project_id_fk
			WHERE p.org_id_fk = $1 AND m.user_id_fk = $2
			ORDER BY p.name ASC`
		args = []interface{}{orgID, userID}
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []map[string]interface{}
	for rows.Next() {
		var id, oID int
		var name, role string
		if err := rows.Scan(&id, &oID, &name, &role); err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}{
			"id":     id,
			"org_id": oID,
			"name":   name,
			"role":   role,
		})
	}
	return res, nil
}

func (s *Store) CreateProject(ctx context.Context, orgID int, name string) (int, error) {
	var id int
	query := `INSERT INTO projects (org_id_fk, name) VALUES ($1, $2) RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query, orgID, name).Scan(&id)
	if err != nil {
		slog.Error("failed to create project", "error", err, "org_id", orgID)
		return 0, err
	}
	return id, nil
}

func (s *Store) GetProjectMembers(ctx context.Context, projectID int) ([]map[string]interface{}, error) {
	query := `
		SELECT u.id_pk, u.login, m.role 
		FROM project_member m
		JOIN "user" u ON m.user_id_fk = u.id_pk
		WHERE m.project_id_fk = $1
		ORDER BY m.role ASC, u.login ASC`

	rows, err := s.conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var id int
		var login, role string
		if err := rows.Scan(&id, &login, &role); err != nil {
			return nil, err
		}
		members = append(members, map[string]interface{}{
			"user_id": id,
			"login":   login,
			"role":    role,
		})
	}
	return members, nil
}

func (s *Store) GetUserRoleInProject(ctx context.Context, projectID, userID int) (string, error) {
	var role string
	err := s.conn.QueryRow(ctx, `SELECT role FROM project_member WHERE project_id_fk = $1 AND user_id_fk = $2`, projectID, userID).Scan(&role)
	return role, err
}

func (s *Store) AddProjectMember(ctx context.Context, projectID, userID int, role string) error {
	query := `INSERT INTO project_member (project_id_fk, user_id_fk, role) VALUES ($1, $2, $3)`
	_, err := s.conn.Exec(ctx, query, projectID, userID, role)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("already_member")
		}
		return err
	}
	return nil
}

func (s *Store) GetProjectOrgID(ctx context.Context, projectID int) (int, error) {
	var orgID int
	err := s.conn.QueryRow(ctx, `SELECT org_id_fk FROM projects WHERE id_pk = $1`, projectID).Scan(&orgID)
	return orgID, err
}

func (s *Store) UpdateProjectMemberRole(ctx context.Context, projectID, userID int, role string) error {
	query := `UPDATE project_member SET role = $1 WHERE project_id_fk = $2 AND user_id_fk = $3`

	result, err := s.conn.Exec(ctx, query, role, projectID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member_not_found")
	}
	return nil
}

func (s *Store) RemoveProjectMember(ctx context.Context, projectID, userID int) error {
	query := `DELETE FROM project_member WHERE project_id_fk = $1 AND user_id_fk = $2`

	result, err := s.conn.Exec(ctx, query, projectID, userID)
	if err != nil {
		slog.Error("failed to remove project member", "error", err, "project_id", projectID, "user_id", userID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member_not_found")
	}
	return nil
}
