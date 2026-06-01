package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func FormatBugAPI(bug map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(bug))
	for k, v := range bug {
		out[k] = v
	}
	textFields := []string{
		"description", "playback_description", "expected_result", "actual_result",
		"version_product", "os",
	}
	for _, f := range textFields {
		out[f] = stringOrEmpty(out[f])
	}
	if out["status"] == nil {
		out["status"] = "Open"
	} else if s, ok := out["status"].(*string); ok {
		if s == nil {
			out["status"] = "Open"
		} else {
			out["status"] = *s
		}
	}
	if out["severity"] != nil {
		out["severity"] = derefString(out["severity"])
	}
	if out["priority"] != nil {
		out["priority"] = derefString(out["priority"])
	}
	return out
}

func stringOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(*string); ok {
		if s == nil {
			return ""
		}
		return *s
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func derefString(v interface{}) interface{} {
	if s, ok := v.(*string); ok && s != nil {
		return *s
	}
	return v
}

func scanBugRow(
	id, taskID int,
	createdBy, assignedTo, passedBy, acceptedBy *int,
	severity, priority, status *string,
	description, playbackDescription, expectedResult, actualResult *string,
	versionProduct, osVal *string,
	createdAt, updatedAt time.Time,
	assignedTime, passedTime, acceptedTime *time.Time,
) map[string]interface{} {
	return map[string]interface{}{
		"id":                   id,
		"task_id":              taskID,
		"created_by":           createdBy,
		"assigned_to":          assignedTo,
		"passed_by":            passedBy,
		"accepted_by":          acceptedBy,
		"severity":             severity,
		"priority":             priority,
		"status":               status,
		"description":          description,
		"playback_description": playbackDescription,
		"expected_result":      expectedResult,
		"actual_result":        actualResult,
		"version_product":      versionProduct,
		"os":                   osVal,
		"created_at":           createdAt,
		"updated_at":           updatedAt,
		"assigned_time":        assignedTime,
		"passed_time":          passedTime,
		"accepted_time":        acceptedTime,
	}
}

const bugSelectColumns = `
		id_pk, task_id_fk, created_by_fk, assigned_to_fk, passed_by_fk, accepted_by_fk,
		severity, priority, status, description, playback_description, expected_result, actual_result,
		version_product, os, created_at, updated_at, assigned_time, passed_time, accepted_time`

func (s *Store) scanBugFromRow(row pgx.Row) (map[string]interface{}, error) {
	var (
		id, taskID             int
		createdBy, assignedTo  *int
		passedBy, acceptedBy   *int
		severity, priority     *string
		status                 *string
		description            *string
		playbackDescription    *string
		expectedResult         *string
		actualResult           *string
		versionProduct, osVal  *string
		createdAt, updatedAt   time.Time
		assignedTime           *time.Time
		passedTime             *time.Time
		acceptedTime           *time.Time
	)
	err := row.Scan(
		&id, &taskID, &createdBy, &assignedTo, &passedBy, &acceptedBy,
		&severity, &priority, &status, &description, &playbackDescription, &expectedResult, &actualResult,
		&versionProduct, &osVal, &createdAt, &updatedAt, &assignedTime, &passedTime, &acceptedTime,
	)
	if err != nil {
		return nil, err
	}
	return scanBugRow(
		id, taskID, createdBy, assignedTo, passedBy, acceptedBy,
		severity, priority, status, description, playbackDescription, expectedResult, actualResult,
		versionProduct, osVal, createdAt, updatedAt, assignedTime, passedTime, acceptedTime,
	), nil
}

func (s *Store) GetBugsByTaskID(ctx context.Context, taskID int) ([]map[string]interface{}, error) {
	query := `SELECT` + bugSelectColumns + ` FROM bugs WHERE task_id_fk = $1 ORDER BY created_at ASC`
	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bugs := []map[string]interface{}{}
	for rows.Next() {
		bug, err := s.scanBugFromRow(rows)
		if err != nil {
			return nil, err
		}
		bugs = append(bugs, bug)
	}
	return bugs, rows.Err()
}

func (s *Store) GetBugByID(ctx context.Context, bugID int) (map[string]interface{}, error) {
	query := `SELECT` + bugSelectColumns + ` FROM bugs WHERE id_pk = $1`
	row := s.conn.QueryRow(ctx, query, bugID)
	bug, err := s.scanBugFromRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("bug_not_found")
		}
		return nil, err
	}
	return bug, nil
}

func (s *Store) GetBugTaskID(ctx context.Context, bugID int) (int, error) {
	var taskID int
	err := s.conn.QueryRow(ctx, `SELECT task_id_fk FROM bugs WHERE id_pk = $1`, bugID).Scan(&taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("bug_not_found")
		}
		return 0, err
	}
	return taskID, nil
}

func (s *Store) CreateBug(ctx context.Context, taskID, createdBy int, fields map[string]interface{}) (int, error) {
	status := "Open"
	if v, ok := fields["status"].(string); ok && strings.TrimSpace(v) != "" {
		status = v
	}

	query := `
		INSERT INTO bugs (
			task_id_fk, created_by_fk, assigned_to_fk, passed_by_fk, accepted_by_fk,
			severity, priority, status, description, playback_description,
			expected_result, actual_result, version_product, os,
			assigned_time, passed_time, accepted_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id_pk`

	var id int
	err := s.conn.QueryRow(ctx, query,
		taskID, createdBy,
		fields["assigned_to_fk"], fields["passed_by_fk"], fields["accepted_by_fk"],
		fields["severity"], fields["priority"], status,
		fields["description"], fields["playback_description"],
		fields["expected_result"], fields["actual_result"],
		fields["version_product"], fields["os"],
		fields["assigned_time"], fields["passed_time"], fields["accepted_time"],
	).Scan(&id)
	if err != nil {
		slog.Error("failed to create bug", "error", err, "task_id", taskID)
		return 0, err
	}
	return id, nil
}

var allowedBugUpdateColumns = map[string]bool{
	"assigned_to_fk":       true,
	"passed_by_fk":         true,
	"accepted_by_fk":       true,
	"severity":             true,
	"priority":             true,
	"status":               true,
	"description":          true,
	"playback_description": true,
	"expected_result":      true,
	"actual_result":        true,
	"version_product":      true,
	"os":                   true,
	"assigned_time":        true,
	"passed_time":          true,
	"accepted_time":        true,
}

func (s *Store) UpdateBug(ctx context.Context, bugID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := `UPDATE bugs SET `
	args := []interface{}{}
	i := 1
	for field, value := range updates {
		if !allowedBugUpdateColumns[field] {
			continue
		}
		query += fmt.Sprintf("%s = $%d, ", field, i)
		args = append(args, value)
		i++
	}
	if i == 1 {
		return nil
	}

	query = strings.TrimSuffix(query, ", ")
	query += fmt.Sprintf(", updated_at = NOW() WHERE id_pk = $%d", i)
	args = append(args, bugID)

	result, err := s.conn.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("bug_not_found")
	}
	return nil
}

func (s *Store) DeleteBug(ctx context.Context, bugID int) error {
	result, err := s.conn.Exec(ctx, `DELETE FROM bugs WHERE id_pk = $1`, bugID)
	if err != nil {
		slog.Error("failed to delete bug", "error", err, "bug_id", bugID)
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("bug_not_found")
	}
	return nil
}
