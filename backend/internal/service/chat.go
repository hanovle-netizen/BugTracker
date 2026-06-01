package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Чат
func (s *Service) GetOrgChat(ctx context.Context, orgID, userID int) (int, error) {
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInOrg(ctx, orgID, userID)
		if err != nil {
			return 0, errors.New("access_denied")
		}
	}

	return s.store.GetOrCreateOrgThread(ctx, orgID)
}

func (s *Service) GetOrgThreads(ctx context.Context, orgID, userID int) ([]ThreadStatsResponse, error) {
	_, err := s.store.GetUserRoleInOrg(ctx, orgID, userID)
	if err != nil {
		return nil, errors.New("access_denied")
	}

	data, err := s.store.GetOrgThreadsWithStats(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	res := []ThreadStatsResponse{}
	for _, d := range data {
		res = append(res, ThreadStatsResponse{
			ID:            d["id"].(int),
			Scope:         d["scope"].(string),
			Title:         d["title"].(string),
			LastMessage:   d["last_message"].(*string),
			LastMessageAt: d["last_message_at"].(*time.Time),
			UnreadCount:   d["unread_count"].(int),
		})
	}
	return res, nil
}

func (s *Service) GetOrCreateThread(ctx context.Context, scope, login string, targetID, userID int) (int, error) {
	globalRole, _ := ctx.Value("role").(string)

	switch scope {
	case "org":
		if globalRole != "admin" {
			if _, err := s.store.GetUserRoleInOrg(ctx, targetID, userID); err != nil {
				return 0, errors.New("access_denied")
			}
		}
		return s.store.GetOrCreateOrgThread(ctx, targetID)

	case "project":
		if globalRole != "admin" {
			if _, err := s.store.GetUserRoleInProject(ctx, targetID, userID); err != nil {
				return 0, errors.New("access_denied")
			}
		}
		return s.store.GetOrCreateProjectThread(ctx, targetID)

	case "dm":

		targetUserID, err := s.store.GetUserIDByLogin(ctx, login)
		if err != nil {
			return 0, errors.New("user_not_found")
		}
		if targetUserID == userID {
			return 0, errors.New("cannot_chat_with_self")
		}
		return s.store.GetOrCreateDMThread(ctx, userID, targetUserID)

	default:
		return 0, errors.New("unsupported_scope")
	}
}

type ThreadStatsResponse struct {
	ID            int        `json:"id"`
	Scope         string     `json:"scope"`
	Title         string     `json:"title,omitempty"`
	PeerLogin     string     `json:"peer_login,omitempty"`
	LastMessage   *string    `json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at"`
	UnreadCount   int        `json:"unread_count"`
}

func (s *Service) GetThreadsByScope(ctx context.Context, scope string, targetID, userID int) ([]ThreadStatsResponse, error) {
	var data []map[string]interface{}
	var err error

	switch scope {
	case "org":

		data, err = s.store.GetOrgThreadsWithStats(ctx, targetID, userID)
	case "project":

		data, err = s.store.GetProjectThreadWithStats(ctx, targetID, userID)
	case "dm":

		data, err = s.store.GetDMThreadsWithStats(ctx, userID)
	default:
		return nil, errors.New("unsupported_scope")
	}

	if err != nil {
		return nil, err
	}

	res := []ThreadStatsResponse{}
	for _, d := range data {
		t := ThreadStatsResponse{
			ID:            d["id"].(int),
			Scope:         d["scope"].(string),
			LastMessage:   d["last_message"].(*string),
			LastMessageAt: d["last_message_at"].(*time.Time),
			UnreadCount:   d["unread_count"].(int),
		}
		if title, ok := d["title"].(string); ok {
			t.Title = title
		}
		if login, ok := d["peer_login"].(string); ok {
			t.PeerLogin = login
		}
		res = append(res, t)
	}
	return res, nil
}

func (s *Service) HasThreadAccess(ctx context.Context, threadID, userID int) bool {
	globalRole, _ := ctx.Value("role").(string)
	if globalRole == "admin" {
		return true
	}

	thread, err := s.store.GetThreadByID(ctx, threadID)
	if err != nil {
		return false
	}

	scope := thread["scope"].(string)

	switch scope {
	case "org":
		orgID := thread["org_id"].(*int)
		_, err := s.store.GetUserRoleInOrg(ctx, *orgID, userID)
		return err == nil
	case "project":
		projID := thread["project_id"].(*int)
		_, err := s.store.GetUserRoleInProject(ctx, *projID, userID)
		return err == nil
	case "dm":
		userA := thread["user_a"].(*int)
		userB := thread["user_b"].(*int)
		return *userA == userID || *userB == userID
	}

	return false
}

func (s *Service) GetMessages(ctx context.Context, threadID, userID, limit, before, after int) ([]map[string]interface{}, error) {

	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	if !s.HasThreadAccess(ctx, threadID, userID) {
		return nil, errors.New("access_denied")
	}

	return s.store.GetThreadMessages(ctx, threadID, limit, before, after)
}

func (s *Service) SendMessage(ctx context.Context, threadID, userID int, body string) (int, error) {
	if body == "" {
		return 0, errors.New("message_body_required")
	}

	if !s.HasThreadAccess(ctx, threadID, userID) {
		slog.Warn("unauthorized chat message attempt", "user_id", userID, "thread_id", threadID)
		return 0, errors.New("access_denied")
	}

	return s.store.CreateMessage(ctx, threadID, userID, body)
}

func (s *Service) EditMessage(ctx context.Context, messageID, userID int, body string) error {
	if body == "" {
		return errors.New("message_body_required")
	}

	return s.store.UpdateChatMessage(ctx, messageID, userID, body)
}

func (s *Service) DeleteMessage(ctx context.Context, messageID, userID int) error {
	slog.Info("user is deleting message", "message_id", messageID, "user_id", userID)
	return s.store.DeleteChatMessage(ctx, messageID, userID)
}

func (s *Service) ReadThread(ctx context.Context, threadID, userID int) error {
	if !s.HasThreadAccess(ctx, threadID, userID) {
		return errors.New("access_denied")
	}
	return s.store.MarkThreadAsRead(ctx, threadID, userID)
}

func (s *Service) GetTyping(ctx context.Context, threadID, userID int) ([]string, error) {
	if !s.HasThreadAccess(ctx, threadID, userID) {
		return nil, errors.New("access_denied")
	}
	return s.store.GetTypingUsers(ctx, threadID, userID)
}

func (s *Service) ReportTyping(ctx context.Context, threadID, userID int, isTyping bool) error {
	if !s.HasThreadAccess(ctx, threadID, userID) {
		return errors.New("access_denied")
	}
	return s.store.SetUserTyping(ctx, threadID, userID, isTyping)
}
