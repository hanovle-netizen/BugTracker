package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserResponse struct {
	Id    int    `json:"id"`
	Token string `json:"token"`
	Role  string `json:"role"`
}

func validatePassword(s string) bool {
	var hasUpper, hasLower, hasDigit bool
	if len(s) < 8 {
		return false
	}
	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// Аутентификация

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {

	// Frontend always sends role="qa". Admin is assigned manually.
	req.Role = "qa"
	slog.Info("registering new user", "login", req.Login, "role", req.Role)

	if req.Password == "" {
		slog.Warn("registration failed: missing password", "login", req.Login)
		return nil, errors.New("password_required")
	}

	if !validatePassword(req.Password) {
		errCode := "password_too_weak"
		if len(req.Password) < 8 {
			errCode = "password_too_short"
		}
		slog.Warn("registration failed: invalid password", "login", req.Login, "reason", errCode)
		return nil, errors.New(errCode)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err, "login", req.Login)
		return nil, err
	}

	id, err := s.store.CreateUser(ctx, req.Login, string(hashed), req.Role)
	if err != nil {

		return nil, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  id,
		"role": req.Role,
		"ver":  0,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("failed to sign token", "error", err, "user_id", id)
		return nil, err
	}

	slog.Info("user registered and token generated", "user_id", id, "login", req.Login)

	return &UserResponse{
		Id:    id,
		Token: tokenString,
		Role:  req.Role,
	}, nil
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*UserResponse, error) {
	slog.Info("login attempt", "login", req.Login)

	id, hashedPwd, role, ver, err := s.store.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if err.Error() == "user_not_found" {

			slog.Warn("login failed: user not found", "login", req.Login)
			return nil, errors.New("invalid_credentials")
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.Password))
	if err != nil {
		slog.Warn("login failed: wrong password", "login", req.Login, "user_id", id)
		return nil, errors.New("invalid_credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  id,
		"role": role,
		"ver":  ver,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("failed to sign token during login", "error", err, "user_id", id)
		return nil, err
	}

	slog.Info("user logged in successfully", "user_id", id, "login", req.Login)

	return &UserResponse{
		Id:    id,
		Token: tokenString,
		Role:  role,
	}, nil
}

// Пользователь (текущий)
type MeResponse struct {
	Id    int    `json:"id"`
	Login string `json:"login"`
	Role  string `json:"role"`
}

func (s *Service) GetMe(ctx context.Context, userID int) (*MeResponse, error) {
	id, login, _, role, _, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &MeResponse{
		Id:    id,
		Login: login,
		Role:  role,
	}, nil
}

type UpdateLoginRequest struct {
	NewLogin        string `json:"new_login"`
	CurrentPassword string `json:"current_password"`
}

func (s *Service) UpdateLogin(ctx context.Context, userID int, req UpdateLoginRequest) error {

	_, _, hashedPwd, _, _, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.CurrentPassword)); err != nil {
		slog.Warn("login update failed: wrong password", "user_id", userID)
		return errors.New("invalid_password")
	}

	return s.store.UpdateUserLogin(ctx, userID, req.NewLogin)
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Service) UpdatePassword(ctx context.Context, userID int, req UpdatePasswordRequest) error {
	_, _, hashedPwd, _, _, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.CurrentPassword)); err != nil {
		slog.Warn("password update failed: wrong current password", "user_id", userID)
		return errors.New("invalid_password")
	}

	if !validatePassword(req.NewPassword) {
		if len(req.NewPassword) < 8 {
			return errors.New("password_too_short")
		}
		return errors.New("password_too_weak")
	}

	newHashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.store.UpdateUserPassword(ctx, userID, string(newHashed))
}

func (s *Service) AdminResetPassword(ctx context.Context, targetID int, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password_too_short")
	}
	if !validatePassword(newPassword) {
		return errors.New("password_too_weak")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.store.UpdateUserPassword(ctx, targetID, string(hashed))
}

func (s *Service) LogoutAll(ctx context.Context, userID int) error {
	slog.Info("executing logout-all", "user_id", userID)
	return s.store.IncrementUserVersion(ctx, userID)
}

func (s *Service) GetUserByID(ctx context.Context, id int) (*MeResponse, error) {
	id_pk, login, _, role, _, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &MeResponse{
		Id:    id_pk,
		Login: login,
		Role:  role,
	}, nil
}
