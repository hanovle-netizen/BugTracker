package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// Чат
func (s *Store) GetOrCreateOrgThread(ctx context.Context, orgID int) (int, error) {
	var threadID int

	querySelect := `SELECT id_pk FROM chat_thread WHERE scope = 'org' AND org_id_fk = $1`
	err := s.conn.QueryRow(ctx, querySelect, orgID).Scan(&threadID)

	if err == nil {
		return threadID, nil
	}

	if err != pgx.ErrNoRows {
		return 0, err
	}

	queryInsert := `INSERT INTO chat_thread (scope, org_id_fk) VALUES ('org', $1) RETURNING id_pk`
	err = s.conn.QueryRow(ctx, queryInsert, orgID).Scan(&threadID)
	return threadID, err
}

func (s *Store) GetOrgThreadsWithStats(ctx context.Context, orgID, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			t.id_pk, 
			t.scope, 
			o.name as title,
			m.body as last_message,
			m.created_at as last_message_at,
			(SELECT COUNT(*) FROM chat_message cm 
			 WHERE cm.thread_id_fk = t.id_pk 
			 AND cm.id_pk > COALESCE((SELECT last_read_message_id FROM chat_read_state rs WHERE rs.thread_id_fk = t.id_pk AND rs.user_id_fk = $2), 0)
			) as unread_count
		FROM chat_thread t
		JOIN organizations o ON t.org_id_fk = o.id_pk
		LEFT JOIN LATERAL (
			SELECT body, created_at FROM chat_message 
			WHERE thread_id_fk = t.id_pk 
			ORDER BY created_at DESC LIMIT 1
		) m ON true
		WHERE t.scope = 'org' AND t.org_id_fk = $1`

	rows, err := s.conn.Query(ctx, query, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []map[string]interface{}
	for rows.Next() {
		var id int
		var scope, title string
		var lastMsg *string
		var lastTime *time.Time
		var unread int
		if err := rows.Scan(&id, &scope, &title, &lastMsg, &lastTime, &unread); err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}{
			"id":              id,
			"scope":           scope,
			"title":           title,
			"last_message":    lastMsg,
			"last_message_at": lastTime,
			"unread_count":    unread,
		})
	}
	return res, nil
}

func (s *Store) GetOrCreateProjectThread(ctx context.Context, projectID int) (int, error) {
	var threadID int

	querySelect := `SELECT id_pk FROM chat_thread WHERE scope = 'project' AND project_id_fk = $1`
	err := s.conn.QueryRow(ctx, querySelect, projectID).Scan(&threadID)

	if err == nil {
		return threadID, nil
	}

	if err != pgx.ErrNoRows {
		return 0, err
	}

	queryInsert := `INSERT INTO chat_thread (scope, project_id_fk) VALUES ('project', $1) RETURNING id_pk`
	err = s.conn.QueryRow(ctx, queryInsert, projectID).Scan(&threadID)
	return threadID, err
}

func (s *Store) GetProjectThreadWithStats(ctx context.Context, projectID, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			t.id_pk, 
			t.scope, 
			p.name as title,
			m.body as last_message,
			m.created_at as last_message_at,
			(SELECT COUNT(*) FROM chat_message cm 
			 WHERE cm.thread_id_fk = t.id_pk 
			 AND cm.id_pk > COALESCE((SELECT last_read_message_id FROM chat_read_state rs WHERE rs.thread_id_fk = t.id_pk AND rs.user_id_fk = $2), 0)
			) as unread_count
		FROM chat_thread t
		JOIN projects p ON t.project_id_fk = p.id_pk
		LEFT JOIN LATERAL (
			SELECT body, created_at FROM chat_message 
			WHERE thread_id_fk = t.id_pk 
			ORDER BY created_at DESC LIMIT 1
		) m ON true
		WHERE t.scope = 'project' AND t.project_id_fk = $1`

	rows, err := s.conn.Query(ctx, query, projectID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []map[string]interface{}
	for rows.Next() {
		var id int
		var scope, title string
		var lastMsg *string
		var lastTime *time.Time
		var unread int
		if err := rows.Scan(&id, &scope, &title, &lastMsg, &lastTime, &unread); err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}{
			"id":              id,
			"scope":           scope,
			"title":           title,
			"last_message":    lastMsg,
			"last_message_at": lastTime,
			"unread_count":    unread,
		})
	}
	return res, nil
}

func (s *Store) GetOrCreateDMThread(ctx context.Context, userA, userB int) (int, error) {
	var threadID int
	querySelect := `
		SELECT id_pk FROM chat_thread 
		WHERE scope = 'dm' AND (
			(dm_user_a_fk = $1 AND dm_user_b_fk = $2) OR 
			(dm_user_a_fk = $2 AND dm_user_b_fk = $1)
		)`
	err := s.conn.QueryRow(ctx, querySelect, userA, userB).Scan(&threadID)

	if err == nil {
		return threadID, nil
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}

	queryInsert := `
		INSERT INTO chat_thread (scope, dm_user_a_fk, dm_user_b_fk) 
		VALUES ('dm', $1, $2) RETURNING id_pk`
	err = s.conn.QueryRow(ctx, queryInsert, userA, userB).Scan(&threadID)
	return threadID, err
}

func (s *Store) GetDMThreadsWithStats(ctx context.Context, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			t.id_pk, 
			t.scope, 
			u.login as peer_login,
			m.body as last_message,
			m.created_at as last_message_at,
			(SELECT COUNT(*) FROM chat_message cm 
			 WHERE cm.thread_id_fk = t.id_pk 
			 AND cm.id_pk > COALESCE((SELECT last_read_message_id FROM chat_read_state rs WHERE rs.thread_id_fk = t.id_pk AND rs.user_id_fk = $1), 0)
			) as unread_count
		FROM chat_thread t
		-- Джойним таблицу пользователей, чтобы найти собеседника
		JOIN "user" u ON u.id_pk = CASE WHEN t.dm_user_a_fk = $1 THEN t.dm_user_b_fk ELSE t.dm_user_a_fk END
		LEFT JOIN LATERAL (
			SELECT body, created_at FROM chat_message 
			WHERE thread_id_fk = t.id_pk 
			ORDER BY created_at DESC LIMIT 1
		) m ON true
		WHERE t.scope = 'dm' AND (t.dm_user_a_fk = $1 OR t.dm_user_b_fk = $1)
		ORDER BY last_message_at DESC NULLS LAST`

	rows, err := s.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []map[string]interface{}
	for rows.Next() {
		var id int
		var scope, peerLogin string
		var lastMsg *string
		var lastTime *time.Time
		var unread int
		if err := rows.Scan(&id, &scope, &peerLogin, &lastMsg, &lastTime, &unread); err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}{
			"id":              id,
			"scope":           scope,
			"peer_login":      peerLogin,
			"last_message":    lastMsg,
			"last_message_at": lastTime,
			"unread_count":    unread,
		})
	}
	return res, nil
}

func (s *Store) GetThreadMessages(ctx context.Context, threadID, limit, beforeID, afterID int) ([]map[string]interface{}, error) {
	query := `
		SELECT m.id_pk, m.thread_id_fk, m.user_id_fk, u.login, 
		       CASE WHEN m.deleted_at IS NOT NULL THEN '' ELSE m.body END as body, 
		       m.created_at, m.edited_at, m.deleted_at
		FROM chat_message m
		JOIN "user" u ON m.user_id_fk = u.id_pk
		WHERE m.thread_id_fk = $1 `

	args := []interface{}{threadID, limit}
	argIdx := 3

	if beforeID > 0 {
		query += fmt.Sprintf("AND m.id_pk < $%d ", argIdx)
		args = append(args, beforeID)
		argIdx++
	}
	if afterID > 0 {
		query += fmt.Sprintf("AND m.id_pk > $%d ", argIdx)
		args = append(args, afterID)
		argIdx++
	}

	query += "ORDER BY m.created_at ASC LIMIT $2"

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var id, tID, uID int
		var login, body string
		var created time.Time
		var edited, deleted *time.Time

		if err := rows.Scan(&id, &tID, &uID, &login, &body, &created, &edited, &deleted); err != nil {
			return nil, err
		}
		messages = append(messages, map[string]interface{}{
			"id":         id,
			"thread_id":  tID,
			"user_id":    uID,
			"user_login": login,
			"body":       body,
			"created_at": created,
			"edited_at":  edited,
			"deleted_at": deleted,
		})
	}
	return messages, nil
}

func (s *Store) GetThreadByID(ctx context.Context, threadID int) (map[string]interface{}, error) {
	var scope string
	var orgID, projID, userA, userB *int
	query := `SELECT scope, org_id_fk, project_id_fk, dm_user_a_fk, dm_user_b_fk FROM chat_thread WHERE id_pk = $1`

	err := s.conn.QueryRow(ctx, query, threadID).Scan(&scope, &orgID, &projID, &userA, &userB)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"scope":      scope,
		"org_id":     orgID,
		"project_id": projID,
		"user_a":     userA,
		"user_b":     userB,
	}, nil
}

func (s *Store) CreateMessage(ctx context.Context, threadID, userID int, body string) (int, error) {
	var id int
	query := `INSERT INTO chat_message (thread_id_fk, user_id_fk, body) VALUES ($1, $2, $3) RETURNING id_pk`

	err := s.conn.QueryRow(ctx, query, threadID, userID, body).Scan(&id)
	if err != nil {
		slog.Error("failed to create chat message", "error", err, "thread_id", threadID)
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateChatMessage(ctx context.Context, messageID, userID int, body string) error {
	query := `
		UPDATE chat_message 
		SET body = $1, edited_at = NOW() 
		WHERE id_pk = $2 AND user_id_fk = $3 AND deleted_at IS NULL`

	result, err := s.conn.Exec(ctx, query, body, messageID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("message_not_found_or_access_denied")
	}
	return nil
}

func (s *Store) DeleteChatMessage(ctx context.Context, messageID, userID int) error {
	query := `
		UPDATE chat_message 
		SET deleted_at = NOW() 
		WHERE id_pk = $1 AND user_id_fk = $2 AND deleted_at IS NULL`

	result, err := s.conn.Exec(ctx, query, messageID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("message_not_found_or_access_denied")
	}
	return nil
}

func (s *Store) MarkThreadAsRead(ctx context.Context, threadID, userID int) error {
	var lastMsgID int
	err := s.conn.QueryRow(ctx, `SELECT COALESCE(MAX(id_pk), 0) FROM chat_message WHERE thread_id_fk = $1`, threadID).Scan(&lastMsgID)
	if err != nil {
		return err
	}

	if lastMsgID == 0 {
		return nil
	}

	query := `
		INSERT INTO chat_read_state (thread_id_fk, user_id_fk, last_read_message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (thread_id_fk, user_id_fk) 
		DO UPDATE SET last_read_message_id = EXCLUDED.last_read_message_id`

	_, err = s.conn.Exec(ctx, query, threadID, userID, lastMsgID)
	return err
}

func (s *Store) SetUserTyping(ctx context.Context, threadID, userID int, isTyping bool) error {
	query := `
		INSERT INTO chat_typing_state (thread_id_fk, user_id_fk, is_typing, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (thread_id_fk, user_id_fk) 
		DO UPDATE SET is_typing = $3, updated_at = NOW()`
	_, err := s.conn.Exec(ctx, query, threadID, userID, isTyping)
	return err
}

func (s *Store) GetTypingUsers(ctx context.Context, threadID, excludeUserID int) ([]string, error) {
	query := `
		SELECT u.login 
		FROM chat_typing_state cts
		JOIN "user" u ON cts.user_id_fk = u.id_pk
		WHERE cts.thread_id_fk = $1 
		  AND cts.user_id_fk != $2 
		  AND cts.updated_at > NOW() - INTERVAL '7 seconds'`

	rows, err := s.conn.Query(ctx, query, threadID, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logins := []string{}
	for rows.Next() {
		var login string
		if err := rows.Scan(&login); err != nil {
			return nil, err
		}
		logins = append(logins, login)
	}
	return logins, nil
}
