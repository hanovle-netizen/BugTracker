package postgres

import (
	"context"
	"time"
)

func (s *Store) GetBugComments(ctx context.Context, bugID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id_pk, user_id_fk, body, created_at
		FROM bug_comment
		WHERE bug_id_fk = $1
		ORDER BY created_at ASC`

	rows, err := s.conn.Query(ctx, query, bugID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []map[string]interface{}{}
	for rows.Next() {
		var id, userID int
		var body string
		var createdAt time.Time
		if err := rows.Scan(&id, &userID, &body, &createdAt); err != nil {
			return nil, err
		}
		comments = append(comments, map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"body":       body,
			"created_at": createdAt,
		})
	}
	return comments, rows.Err()
}

func (s *Store) CreateBugComment(ctx context.Context, bugID, userID int, body string) (int, error) {
	var id int
	query := `INSERT INTO bug_comment (bug_id_fk, user_id_fk, body) VALUES ($1, $2, $3) RETURNING id_pk`
	err := s.conn.QueryRow(ctx, query, bugID, userID, body).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
