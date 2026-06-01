package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Задачи
func (s *Store) GetTasksByProject(ctx context.Context, projectID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, title, description, project_id_fk, owner_id_fk, 
		       status, severity, priority, os, version_product, 
		       created_at, updated_at
		FROM task
		WHERE project_id_fk = $1
		ORDER BY created_at DESC`

	rows, err := s.conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []map[string]interface{}
	for rows.Next() {
		var id, pID, oID int
		var title, desc string
		var status, severity, priority, os, version *string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &title, &desc, &pID, &oID,
			&status, &severity, &priority, &os, &version,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, map[string]interface{}{
			"id":              id,
			"title":           title,
			"description":     desc,
			"project_id":      pID,
			"owner_id":        oID,
			"status":          status,
			"severity":        severity,
			"priority":        priority,
			"os":              os,
			"version_product": version,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		})
	}
	return tasks, nil
}

func (s *Store) CreateTask(ctx context.Context, ownerID int, t map[string]interface{}) (int, error) {
	var id int
	query := `
		INSERT INTO task (
			title, description, project_id_fk, owner_id_fk, 
			status, severity, priority, os, version_product, 
			playback_description, expected_result, actual_result
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query,
		t["title"], t["description"], t["project_id"], ownerID,
		t["status"], t["severity"], t["priority"], t["os"], t["version_product"],
		t["playback_description"], t["expected_result"], t["actual_result"],
	).Scan(&id)

	if err != nil {
		slog.Error("failed to create task", "error", err, "owner_id", ownerID)
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateTask(ctx context.Context, taskID, userID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_user_id = '%d'", userID))
	if err != nil {
		slog.Error("failed to set session user_id", "error", err)
		return err
	}

	query := `UPDATE task SET `
	args := []interface{}{}
	i := 1

	for field, value := range updates {
		query += fmt.Sprintf("%s = $%d, ", field, i)
		args = append(args, value)
		i++
	}

	query = strings.TrimSuffix(query, ", ")
	query += fmt.Sprintf(", updated_at = NOW() WHERE id_pk = $%d", i)
	args = append(args, taskID)

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task_not_found")
	}

	return tx.Commit(ctx)
}

func (s *Store) GetTaskProjectID(ctx context.Context, taskID int) (int, error) {
	var projectID int
	query := `SELECT project_id_fk FROM task WHERE id_pk = $1`
	err := s.conn.QueryRow(ctx, query, taskID).Scan(&projectID)
	return projectID, err
}

func (s *Store) GetTaskByID(ctx context.Context, taskID int) (map[string]interface{}, error) {
	query := `
		SELECT id_pk, title, description, project_id_fk, owner_id_fk,
		       status, severity, priority, os, version_product,
		       playback_description, expected_result, actual_result,
		       assigned_to_fk, assigned_time, passed_by_fk, passed_time, accepted_by_fk, accepted_time,
		       created_at, updated_at
		FROM task
		WHERE id_pk = $1`

	var (
		id, projectID, ownerID int
		title                  string
		description            *string
		status                 *string
		severity               *string
		priority               *string
		osVal                  *string
		versionProduct         *string
		playbackDescription    *string
		expectedResult         *string
		actualResult           *string
		assignedTo             *int
		assignedTime           *time.Time
		passedBy               *int
		passedTime             *time.Time
		acceptedBy             *int
		acceptedTime           *time.Time
		createdAt              time.Time
		updatedAt              time.Time
	)

	err := s.conn.QueryRow(ctx, query, taskID).Scan(
		&id, &title, &description, &projectID, &ownerID,
		&status, &severity, &priority, &osVal, &versionProduct,
		&playbackDescription, &expectedResult, &actualResult,
		&assignedTo, &assignedTime, &passedBy, &passedTime, &acceptedBy, &acceptedTime,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task_not_found")
		}
		return nil, err
	}

	return map[string]interface{}{
		"id":                   id,
		"task_id":              id,
		"title":                title,
		"description":          description,
		"project_id":           projectID,
		"owner_id":             ownerID,
		"status":               status,
		"severity":             severity,
		"priority":             priority,
		"os":                   osVal,
		"version_product":      versionProduct,
		"playback_description": playbackDescription,
		"expected_result":      expectedResult,
		"actual_result":        actualResult,
		"assigned_to":          assignedTo,
		"assigned_time":        assignedTime,
		"passed_by":            passedBy,
		"passed_time":          passedTime,
		"accepted_by":          acceptedBy,
		"accepted_time":        acceptedTime,
		"created_at":           createdAt,
		"updated_at":           updatedAt,
	}, nil
}

func (s *Store) DeleteTask(ctx context.Context, taskID int) error {
	query := `DELETE FROM task WHERE id_pk = $1`

	result, err := s.conn.Exec(ctx, query, taskID)
	if err != nil {
		slog.Error("failed to delete task", "error", err, "task_id", taskID)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task_not_found")
	}
	return nil
}

func (s *Store) UpdateTaskPhoto(ctx context.Context, taskID int, photoData []byte) error {
	query := `UPDATE task SET photo = $1, updated_at = NOW() WHERE id_pk = $2`
	result, err := s.conn.Exec(ctx, query, photoData, taskID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task_not_found")
	}
	return nil
}

func (s *Store) GetTaskPhoto(ctx context.Context, taskID int) ([]byte, error) {
	var photo []byte
	query := `SELECT photo FROM task WHERE id_pk = $1`

	err := s.conn.QueryRow(ctx, query, taskID).Scan(&photo)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task_not_found")
		}
		return nil, err
	}

	if photo == nil {
		return nil, fmt.Errorf("no_photo")
	}

	return photo, nil
}

func (s *Store) GetTaskComments(ctx context.Context, taskID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, task_id_fk, user_id_fk, body, created_at 
		FROM task_comment 
		WHERE task_id_fk = $1 
		ORDER BY created_at ASC`

	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []map[string]interface{}
	for rows.Next() {
		var id, tID, uID int
		var body string
		var createdAt time.Time
		if err := rows.Scan(&id, &tID, &uID, &body, &createdAt); err != nil {
			return nil, err
		}
		comments = append(comments, map[string]interface{}{
			"id":         id,
			"task_id":    tID,
			"user_id":    uID,
			"body":       body,
			"created_at": createdAt,
		})
	}
	return comments, nil
}

func (s *Store) CreateComment(ctx context.Context, taskID, userID int, body string) (int, error) {
	var id int
	query := `INSERT INTO task_comment (task_id_fk, user_id_fk, body) VALUES ($1, $2, $3) RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query, taskID, userID, body).Scan(&id)
	if err != nil {
		slog.Error("failed to create comment", "error", err, "task_id", taskID, "user_id", userID)
		return 0, err
	}
	return id, nil
}

func (s *Store) GetTaskAudit(ctx context.Context, taskID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, task_id_fk, user_id_fk, field, old_value, new_value, changed_at 
		FROM task_audit 
		WHERE task_id_fk = $1 
		ORDER BY changed_at DESC`

	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auditLogs []map[string]interface{}
	for rows.Next() {
		var id, tID, uID int
		var field string
		var oldVal, newVal *string
		var changedAt time.Time

		err := rows.Scan(&id, &tID, &uID, &field, &oldVal, &newVal, &changedAt)
		if err != nil {
			return nil, err
		}

		auditLogs = append(auditLogs, map[string]interface{}{
			"id":         id,
			"task_id":    tID,
			"user_id":    uID,
			"field":      field,
			"old_value":  oldVal,
			"new_value":  newVal,
			"changed_at": changedAt,
		})
	}
	return auditLogs, nil
}

func (s *Store) GetTaskRelations(ctx context.Context, taskID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, bug_id_a_fk, bug_id_b_fk, type 
		FROM bug_relation 
		WHERE bug_id_a_fk = $1 OR bug_id_b_fk = $1`

	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []map[string]interface{}
	for rows.Next() {
		var id, idA, idB int
		var relType string
		if err := rows.Scan(&id, &idA, &idB, &relType); err != nil {
			return nil, err
		}
		relations = append(relations, map[string]interface{}{
			"id":        id,
			"task_id_a": idA,
			"task_id_b": idB,
			"type":      relType,
		})
	}
	return relations, nil
}

func (s *Store) CreateTaskRelation(ctx context.Context, taskA, taskB int, relType string) (int, error) {
	var id int
	query := `INSERT INTO bug_relation (bug_id_a_fk, bug_id_b_fk, type) VALUES ($1, $2, $3) RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query, taskA, taskB, relType).Scan(&id)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return 0, fmt.Errorf("relation_exists")
		}
		return 0, err
	}
	return id, nil
}

func (s *Store) DeleteRelation(ctx context.Context, relID int) (int, int, error) {
	var taskA, taskB int

	err := s.conn.QueryRow(ctx, `SELECT bug_id_a_fk, bug_id_b_fk FROM bug_relation WHERE id_pk = $1`, relID).Scan(&taskA, &taskB)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, fmt.Errorf("relation_not_found")
		}
		return 0, 0, err
	}

	_, err = s.conn.Exec(ctx, `DELETE FROM bug_relation WHERE id_pk = $1`, relID)
	return taskA, taskB, err
}

func (s *Store) GetTaskTags(ctx context.Context, taskID int) ([]string, error) {
	query := `SELECT tag FROM bug_tag WHERE bug_id_fk = $1 ORDER BY tag ASC`

	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (s *Store) GetAllTemplates(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, name, body, created_by_fk, created_at 
		FROM template 
		ORDER BY name ASC`

	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		slog.Error("failed to fetch templates", "error", err)
		return nil, err
	}
	defer rows.Close()

	var templates []map[string]interface{}
	for rows.Next() {
		var id, createdBy int
		var name, body string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &body, &createdBy, &createdAt); err != nil {
			return nil, err
		}
		templates = append(templates, map[string]interface{}{
			"id":         id,
			"name":       name,
			"body":       body,
			"created_by": createdBy,
			"created_at": createdAt,
		})
	}
	return templates, nil
}

func (s *Store) CreateTemplate(ctx context.Context, userID int, name, body string) (int, error) {
	var id int
	query := `INSERT INTO template (name, body, created_by_fk) VALUES ($1, $2, $3) RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query, name, body, userID).Scan(&id)
	if err != nil {
		slog.Error("failed to create template", "error", err, "user_id", userID)
		return 0, err
	}
	return id, nil
}

func (s *Store) DeleteTemplate(ctx context.Context, id int) error {
	query := `DELETE FROM template WHERE id_pk = $1`

	result, err := s.conn.Exec(ctx, query, id)
	if err != nil {
		slog.Error("failed to delete template", "error", err, "id", id)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("template_not_found")
	}
	return nil
}

func (s *Store) GetTaskStats(ctx context.Context, userID int, isTeacher bool) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if isTeacher {
		query = `SELECT COALESCE(status, 'No Status') as status, COUNT(*) as count FROM task GROUP BY status`
	} else {
		query = `
			SELECT COALESCE(t.status, 'No Status') as status, COUNT(t.id_pk) as count 
			FROM task t
			JOIN project_member pm ON t.project_id_fk = pm.project_id_fk
			WHERE pm.user_id_fk = $1
			GROUP BY t.status`
		args = append(args, userID)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		slog.Error("stats query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			slog.Error("stats scan failed", "error", err)
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"status": status,
			"count":  int(count),
		})
	}
	return stats, nil
}

func (s *Store) GetSingleTaskStats(ctx context.Context, taskID int) ([]map[string]interface{}, error) {
	query := `
		SELECT new_value as status, COUNT(*) as count 
		FROM task_audit 
		WHERE task_id_fk = $1 AND field = 'status' AND new_value IS NOT NULL
		GROUP BY new_value`

	rows, err := s.conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"status": status,
			"count":  int(count),
		})
	}
	return stats, nil
}

func (s *Store) GetStatsByProject(ctx context.Context, projectID int) ([]map[string]interface{}, error) {
	query := `
		SELECT COALESCE(status, 'No Status') as status, COUNT(*) as count
		FROM task
		WHERE project_id_fk = $1
		GROUP BY status`
	rows, err := s.conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"status": status,
			"count":  int(count),
		})
	}
	return stats, nil
}

func (s *Store) AddPhoto(ctx context.Context, entityType string, entityID int, objectKey, url string, uploaderID int) (int, error) {
	var id int
	query := `INSERT INTO photo (entity_type, entity_id, object_key, url, uploaded_by_fk) VALUES ($1, $2, $3, $4, $5) RETURNING id_pk`
	if err := s.conn.QueryRow(ctx, query, entityType, entityID, objectKey, url, uploaderID).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) GetPhotos(ctx context.Context, entityType string, entityID int) ([]map[string]interface{}, error) {
	query := `SELECT id_pk, url, object_key, uploaded_by_fk FROM photo WHERE entity_type = $1 AND entity_id = $2 ORDER BY id_pk ASC`
	rows, err := s.conn.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photos := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var url, key string
		var uploadedBy *int
		if err := rows.Scan(&id, &url, &key, &uploadedBy); err != nil {
			return nil, err
		}
		photos = append(photos, map[string]interface{}{
			"id":          id,
			"url":         url,
			"object_key":  key,
			"uploaded_by": uploadedBy,
		})
	}
	return photos, nil
}

func (s *Store) GetPhotoByID(ctx context.Context, photoID int) (map[string]interface{}, error) {
	query := `SELECT id_pk, entity_type, entity_id, object_key, url, uploaded_by_fk FROM photo WHERE id_pk = $1`
	var id, entityID int
	var entityType, objectKey, url string
	var uploadedBy *int
	err := s.conn.QueryRow(ctx, query, photoID).Scan(&id, &entityType, &entityID, &objectKey, &url, &uploadedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("photo_not_found")
		}
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"entity_type": entityType,
		"entity_id":   entityID,
		"object_key":  objectKey,
		"url":         url,
		"uploaded_by": uploadedBy,
	}, nil
}

func (s *Store) DeletePhoto(ctx context.Context, photoID int) error {
	result, err := s.conn.Exec(ctx, `DELETE FROM photo WHERE id_pk = $1`, photoID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("photo_not_found")
	}
	return nil
}
