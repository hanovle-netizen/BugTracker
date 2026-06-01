package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Группы (Организации)
func (s *Store) GetUserOrganizations(ctx context.Context, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT o.id_pk, o.name, m.role 
		FROM organizations o
		JOIN org_member m ON o.id_pk = m.org_id_fk
		WHERE m.user_id_fk = $1
		ORDER BY o.name ASC`

	rows, err := s.conn.Query(ctx, query, userID)
	if err != nil {
		slog.Error("failed to fetch user organizations", "error", err, "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var orgs []map[string]interface{}
	for rows.Next() {
		var id int
		var name, role string
		if err := rows.Scan(&id, &name, &role); err != nil {
			return nil, err
		}
		orgs = append(orgs, map[string]interface{}{
			"id":   id,
			"name": name,
			"role": role,
		})
	}
	return orgs, nil
}

func (s *Store) CreateOrganization(ctx context.Context, name string, ownerID int) (int, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var orgID int
	err = tx.QueryRow(ctx, `INSERT INTO organizations (name) VALUES ($1) RETURNING id_pk`, name).Scan(&orgID)
	if err != nil {
		slog.Error("failed to insert organization", "error", err)
		return 0, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO org_member (org_id_fk, user_id_fk, role) VALUES ($1, $2, $3)`,
		orgID, ownerID, "owner",
	)
	if err != nil {
		slog.Error("failed to insert org member", "error", err)
		return 0, err
	}

	return orgID, tx.Commit(ctx)
}

func (s *Store) GetUserRoleInOrg(ctx context.Context, orgID, userID int) (string, error) {
	var role string
	query := `SELECT role FROM org_member WHERE org_id_fk = $1 AND user_id_fk = $2`

	err := s.conn.QueryRow(ctx, query, orgID, userID).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("membership_not_found")
		}
		slog.Error("database error fetching user role in org", "error", err, "org_id", orgID, "user_id", userID)
		return "", err
	}

	return role, nil
}

func (s *Store) GetOrganizationMembers(ctx context.Context, orgID int) ([]map[string]interface{}, error) {
	query := `
		SELECT u.id_pk, u.login, m.role 
		FROM org_member m
		JOIN "user" u ON m.user_id_fk = u.id_pk
		WHERE m.org_id_fk = $1
		ORDER BY m.role DESC, u.login ASC`

	rows, err := s.conn.Query(ctx, query, orgID)
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

func (s *Store) GetUserIDByLogin(ctx context.Context, login string) (int, error) {
	var id int
	err := s.conn.QueryRow(ctx, `SELECT id_pk FROM "user" WHERE login = $1`, login).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("user_not_found")
		}
		return 0, err
	}
	return id, nil
}

func (s *Store) AddOrgMember(ctx context.Context, orgID, userID int, role string) error {
	query := `INSERT INTO org_member (org_id_fk, user_id_fk, role) VALUES ($1, $2, $3)`
	_, err := s.conn.Exec(ctx, query, orgID, userID, role)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("already_member")
		}
		return err
	}
	return nil
}

func (s *Store) UpdateOrgMemberRole(ctx context.Context, orgID, userID int, newRole string) error {
	query := `UPDATE org_member SET role = $1 WHERE org_id_fk = $2 AND user_id_fk = $3`

	result, err := s.conn.Exec(ctx, query, newRole, orgID, userID)
	if err != nil {
		slog.Error("failed to update org member role", "error", err, "org_id", orgID, "user_id", userID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member_not_found")
	}
	return nil
}

func (s *Store) RemoveOrgMember(ctx context.Context, orgID, userID int) error {
	query := `DELETE FROM org_member WHERE org_id_fk = $1 AND user_id_fk = $2`

	result, err := s.conn.Exec(ctx, query, orgID, userID)
	if err != nil {
		slog.Error("failed to remove org member", "error", err, "org_id", orgID, "user_id", userID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member_not_found")
	}
	return nil
}
