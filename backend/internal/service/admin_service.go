package service

import (
	"context"
	"errors"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors returned by AdminService.
var (
	ErrEmailTaken       = errors.New("email already exists")
	ErrAdminNotFound    = errors.New("admin not found")
	ErrCannotDeleteSelf = errors.New("cannot delete your own admin account")
	ErrNotAnAdmin       = errors.New("target user is not an admin")
)

// AdminService handles admin account management operations.
type AdminService struct {
	userRepo *repository.UserRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(userRepo *repository.UserRepository) *AdminService {
	return &AdminService{userRepo: userRepo}
}

// ListAdmins returns one page of admin accounts plus the total count.
// PageSize <= 0 returns every row.
func (s *AdminService) ListAdmins(ctx context.Context, page, pageSize int) ([]dto.AdminListItem, int64, error) {
	users, total, err := s.userRepo.WithCtx(ctx).FindAllAdmins(page, pageSize)
	if err != nil {
		return nil, 0, err
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
	return result, total, nil
}

// CreateAdmin creates a new admin account.
// The new account will have MustChangePW = true so the user must set a new
// password on first login.
// CreateAdmin persists a new admin and returns the newly created user's ID.
func (s *AdminService) CreateAdmin(ctx context.Context, req dto.CreateAdminRequest) (uint, error) {
	repo := s.userRepo.WithCtx(ctx)

	existing, _ := repo.FindByEmail(req.Email)
	if existing != nil {
		return 0, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, ErrPasswordHashFailed
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		Role:         "admin",
		Status:       "active",
		MustChangePW: true,
	}
	if err := repo.Create(user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

// DeleteAdmin removes an admin account.
// An admin cannot delete their own account.
func (s *AdminService) DeleteAdmin(ctx context.Context, requesterID, targetID uint) error {
	if requesterID == targetID {
		return ErrCannotDeleteSelf
	}

	repo := s.userRepo.WithCtx(ctx)
	target, err := repo.FindByID(targetID)
	if err != nil {
		return ErrAdminNotFound
	}
	if target.Role != "admin" {
		return ErrNotAnAdmin
	}
	return repo.DeleteByID(targetID)
}
