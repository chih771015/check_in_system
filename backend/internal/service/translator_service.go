package service

import (
	"context"
	"errors"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors returned by TranslatorService.
var (
	ErrTranslatorNotFound = errors.New("translator not found")
	ErrNotATranslator     = errors.New("user is not a translator")
	ErrInvalidStatus      = errors.New("status must be 'active' or 'disabled'")
)

// TranslatorService handles translator management business logic.
type TranslatorService struct {
	userRepo *repository.UserRepository
}

// NewTranslatorService creates a new TranslatorService.
func NewTranslatorService(userRepo *repository.UserRepository) *TranslatorService {
	return &TranslatorService{userRepo: userRepo}
}

// List returns all translators, optionally filtered by status.
func (s *TranslatorService) List(ctx context.Context, status string) ([]dto.TranslatorListResponse, error) {
	users, err := s.userRepo.WithCtx(ctx).FindAll(status)
	if err != nil {
		return nil, err
	}

	result := make([]dto.TranslatorListResponse, len(users))
	for i, u := range users {
		result[i] = dto.TranslatorListResponse{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Phone:     u.Phone,
			Status:    u.Status,
			CreatedAt: u.CreatedAt,
		}
	}
	return result, nil
}

// Create adds a new translator account.
func (s *TranslatorService) Create(ctx context.Context, req dto.CreateTranslatorRequest) error {
	repo := s.userRepo.WithCtx(ctx)
	// Check if email already exists
	existing, _ := repo.FindByEmail(req.Email)
	if existing != nil {
		return ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return ErrPasswordHashFailed
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		Phone:        req.Phone,
		Role:         "translator",
		Status:       "active",
		MustChangePW: true,
	}

	return repo.Create(user)
}

// Update modifies an existing translator's information.
func (s *TranslatorService) Update(ctx context.Context, id uint, req dto.UpdateTranslatorRequest) error {
	user, err := s.userRepo.WithCtx(ctx).FindByID(id)
	if err != nil {
		return ErrTranslatorNotFound
	}

	if user.Role != "translator" {
		return ErrNotATranslator
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "disabled" {
			return ErrInvalidStatus
		}
		user.Status = *req.Status
	}

	return s.userRepo.WithCtx(ctx).Update(user)
}

// Disable sets a translator's status to disabled.
func (s *TranslatorService) Disable(ctx context.Context, id uint) error {
	user, err := s.userRepo.WithCtx(ctx).FindByID(id)
	if err != nil {
		return ErrTranslatorNotFound
	}

	if user.Role != "translator" {
		return ErrNotATranslator
	}

	user.Status = "disabled"
	return s.userRepo.WithCtx(ctx).Update(user)
}
