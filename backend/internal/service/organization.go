package service

import (
	"context"
	"errors"
)

// Группы (Организации)
type OrgResponse struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func (s *Service) GetMyOrganizations(ctx context.Context, userID int) ([]OrgResponse, error) {
	data, err := s.store.GetUserOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}

	var res []OrgResponse
	for _, item := range data {
		res = append(res, OrgResponse{
			Id:   item["id"].(int),
			Name: item["name"].(string),
			Role: item["role"].(string),
		})
	}
	return res, nil
}

func (s *Service) CreateOrg(ctx context.Context, name string, userID int) (*OrgResponse, error) {
	if name == "" {
		return nil, errors.New("name_required")
	}

	id, err := s.store.CreateOrganization(ctx, name, userID)
	if err != nil {
		return nil, err
	}

	return &OrgResponse{
		Id:   id,
		Name: name,
		Role: "owner",
	}, nil
}

type OrgMemberResponse struct {
	UserID int    `json:"user_id"`
	Login  string `json:"login"`
	Role   string `json:"role"`
}

func (s *Service) GetOrgMembers(ctx context.Context, orgID, requesterID int) ([]OrgMemberResponse, error) {
	_, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)

	userRole, _ := ctx.Value("role").(string)

	if err != nil && userRole != "admin" {
		return nil, errors.New("access_denied")
	}

	data, err := s.store.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var res []OrgMemberResponse
	for _, m := range data {
		res = append(res, OrgMemberResponse{
			UserID: m["user_id"].(int),
			Login:  m["login"].(string),
			Role:   m["role"].(string),
		})
	}
	return res, nil
}

func (s *Service) AddMemberToOrg(ctx context.Context, orgID, requesterID int, login, role string) error {
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		reqRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqRole != "owner" && reqRole != "admin") {
			return errors.New("access_denied")
		}
	}

	targetID, err := s.store.GetUserIDByLogin(ctx, login)
	if err != nil {
		return err
	}

	return s.store.AddOrgMember(ctx, orgID, targetID, role)
}

func (s *Service) UpdateMemberRole(ctx context.Context, orgID, requesterID, targetID int, newRole string) error {
	if newRole != "owner" && newRole != "admin" && newRole != "member" {
		return errors.New("invalid_role")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		reqRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqRole != "owner" && reqRole != "admin") {
			return errors.New("access_denied")
		}

		if reqRole == "admin" && newRole == "owner" {
			return errors.New("insufficient_permissions")
		}
	}

	return s.store.UpdateOrgMemberRole(ctx, orgID, targetID, newRole)
}

func (s *Service) RemoveMember(ctx context.Context, orgID, requesterID, targetID int) error {
	globalRole, _ := ctx.Value("role").(string)

	if requesterID != targetID && globalRole != "admin" {
		reqRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqRole != "owner" && reqRole != "admin") {
			return errors.New("access_denied")
		}

		targetRole, err := s.store.GetUserRoleInOrg(ctx, orgID, targetID)
		if err == nil && targetRole == "owner" && reqRole == "admin" {
			return errors.New("insufficient_permissions")
		}
	}

	return s.store.RemoveOrgMember(ctx, orgID, targetID)
}
