package service

import (
	"errors"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/middleware"
	"translator-checkin/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication-related business logic.
type AuthService struct {
	userRepo *repository.UserRepository
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(email, password string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "active" {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := middleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:           user.ID,
			Email:        user.Email,
			Name:         user.Name,
			Phone:        user.Phone,
			Role:         user.Role,
			Status:       user.Status,
			MustChangePW: user.MustChangePW,
		},
	}, nil
}

// ChangePassword updates a user's password after verifying the old one.
func (s *AuthService) ChangePassword(userID uint, oldPW, newPW string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPW)); err != nil {
		return errors.New("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPW), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user.PasswordHash = string(hash)
	user.MustChangePW = false

	return s.userRepo.Update(user)
}
