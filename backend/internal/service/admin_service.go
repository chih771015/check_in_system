package service

import (
	"context"
	"errors"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// AdminService handles admin account management operations.
type AdminService struct {
	userRepo *repository.UserRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(userRepo *repository.UserRepository) *AdminService {
	return &AdminService{userRepo: userRepo}
}

// ListAdmins returns all admin accounts.
func (s *AdminService) ListAdmins(ctx context.Context) ([]dto.AdminListItem, error) {
	users, err := s.userRepo.WithCtx(ctx).FindAllAdmins()
	if err != nil {
		return nil, err
	}
	result := make([]dto.AdminListItem, len(users))
	for i, u := range users {
		result[i] = dto.AdminListItem{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Status:    u.Status,
			CreatedAt: u.CreatedAt,
		}
	}
	return result, nil
}

// CreateAdmin creates a new admin account.
// The new account will have MustChangePW = true so the user must set a new
// password on first login.
func (s *AdminService) CreateAdmin(ctx context.Context, req dto.CreateAdminRequest) error {
	repo := s.userRepo.WithCtx(ctx)

	existing, _ := repo.FindByEmail(req.Email)
	if existing != nil {
		return errors.New("此 Email 已被使用")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密碼雜湊失敗")
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		Role:         "admin",
		Status:       "active",
		MustChangePW: true,
	}
	return repo.Create(user)
}

// DeleteAdmin removes an admin account.
// An admin cannot delete their own account.
func (s *AdminService) DeleteAdmin(ctx context.Context, requesterID, targetID uint) error {
	if requesterID == targetID {
		return errors.New("無法刪除自己的管理員帳號")
	}

	repo := s.userRepo.WithCtx(ctx)
	target, err := repo.FindByID(targetID)
	if err != nil {
		return errors.New("找不到此管理員帳號")
	}
	if target.Role != "admin" {
		return errors.New("目標帳號不是管理員")
	}
	return repo.DeleteByID(targetID)
}
