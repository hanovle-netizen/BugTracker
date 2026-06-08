package service

import (
	"context"
	"errors"
)

type ProjectResponse struct {
	Id    int    `json:"id"`
	OrgId int    `json:"org_id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (s *Service) GetOrgProjects(ctx context.Context, orgID, userID int) ([]ProjectResponse, error) {
	globalRole, _ := ctx.Value("role").(string)
	isTeacher := globalRole == "admin"

	if !isTeacher {
		_, err := s.store.GetUserRoleInOrg(ctx, orgID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	data, err := s.store.GetProjectsByOrg(ctx, orgID, userID, isTeacher)
	if err != nil {
		return nil, err
	}

	var res []ProjectResponse
	for _, d := range data {
		res = append(res, ProjectResponse{
			Id:    d["id"].(int),
			OrgId: d["org_id"].(int),
			Name:  d["name"].(string),
			Role:  d["role"].(string),
		})
	}
	return res, nil
}

func (s *Service) CreateProject(ctx context.Context, orgID, userID int, name string) (*ProjectResponse, error) {
	if name == "" {
		return nil, errors.New("name_required")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		// Создавать проект может любой участник организации, независимо от роли в ней
		// (owner/admin/member) — так требует логика курса.
		if _, err := s.store.GetUserRoleInOrg(ctx, orgID, userID); err != nil {
			return nil, errors.New("access_denied")
		}
	}

	id, err := s.store.CreateProject(ctx, orgID, userID, name)
	if err != nil {
		return nil, err
	}

	return &ProjectResponse{
		Id:    id,
		OrgId: orgID,
		Name:  name,
		Role:  "pm",
	}, nil
}

type ProjectMemberResponse struct {
	UserID   int    `json:"user_id"`
	Login    string `json:"login"`
	Role     string `json:"role"`
	Position string `json:"position"`
}

func (s *Service) GetProjectMembers(ctx context.Context, projectID, requesterID int) ([]ProjectMemberResponse, error) {
	globalRole, _ := ctx.Value("role").(string)

	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, requesterID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	data, err := s.store.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var res []ProjectMemberResponse
	for _, m := range data {
		res = append(res, ProjectMemberResponse{
			UserID:   m["user_id"].(int),
			Login:    m["login"].(string),
			Role:     m["role"].(string),
			Position: m["position"].(string),
		})
	}
	return res, nil
}

func (s *Service) AddMemberToProject(ctx context.Context, projectID, requesterID int, login, role, position string) error {
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		orgID, err := s.store.GetProjectOrgID(ctx, projectID)
		if err != nil {
			return err
		}
		reqOrgRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqOrgRole != "owner" && reqOrgRole != "admin") {
			return errors.New("access_denied")
		}
	}

	targetID, err := s.store.GetUserIDByLogin(ctx, login)
	if err != nil {
		return err
	}

	return s.store.AddProjectMember(ctx, projectID, targetID, role, position)
}

func (s *Service) UpdateProjectMember(ctx context.Context, projectID, requesterID, targetID int, role, position string) error {
	validRoles := map[string]bool{"pm": true, "dev": true, "qa": true, "viewer": true}
	if !validRoles[role] {
		return errors.New("invalid_role")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		orgID, err := s.store.GetProjectOrgID(ctx, projectID)
		if err != nil {
			return err
		}
		reqOrgRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqOrgRole != "owner" && reqOrgRole != "admin") {
			return errors.New("access_denied")
		}
	}

	return s.store.UpdateProjectMemberRole(ctx, projectID, targetID, role, position)
}

func (s *Service) RemoveProjectMember(ctx context.Context, projectID, requesterID, targetID int) error {
	globalRole, _ := ctx.Value("role").(string)

	if requesterID != targetID && globalRole != "admin" {
		orgID, err := s.store.GetProjectOrgID(ctx, projectID)
		if err != nil {
			return err
		}

		reqOrgRole, err := s.store.GetUserRoleInOrg(ctx, orgID, requesterID)
		if err != nil || (reqOrgRole != "owner" && reqOrgRole != "admin") {
			return errors.New("access_denied")
		}
	}

	return s.store.RemoveProjectMember(ctx, projectID, targetID)
}
