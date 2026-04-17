package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"translator-checkin/internal/config"
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
// Implements account lockout: after N failed attempts the account is locked
// for LockDurationMinutes. A successful login resets the counter.
//
// The context is passed through to the repository so that SQL spans
// emitted by the GORM OTel plugin nest under the gin server span.
func (s *AuthService) Login(ctx context.Context, email, password string) (*dto.LoginResponse, error) {
	repo := s.userRepo.WithCtx(ctx)
	user, err := repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "active" {
		return nil, errors.New("account is disabled")
	}

	// Check lockout window.
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		remaining := time.Until(*user.LockedUntil).Round(time.Second)
		return nil, fmt.Errorf("account locked, try again in %s", remaining)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Increment login attempts; lock account if threshold reached.
		_ = repo.IncrementLoginAttempts(user.ID)
		cfg := config.AppConfig
		maxAttempts := 5
		lockMinutes := 15
		if cfg != nil {
			if cfg.MaxLoginAttempts > 0 {
				maxAttempts = cfg.MaxLoginAttempts
			}
			if cfg.LockDurationMinutes > 0 {
				lockMinutes = cfg.LockDurationMinutes
			}
		}
		if user.LoginAttempts+1 >= maxAttempts {
			until := time.Now().Add(time.Duration(lockMinutes) * time.Minute)
			_ = repo.LockUser(user.ID, until)
		}
		return nil, errors.New("invalid email or password")
	}

	// Success — reset attempt counter and lock state.
	if user.LoginAttempts > 0 || user.LockedUntil != nil {
		_ = repo.ResetLoginAttempts(user.ID)
	}

	token, err := middleware.GenerateToken(user.ID, user.Role, user.MustChangePW)
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
// On success it returns a freshly minted JWT so the caller can drop the
// stale "must change password" claim from their token.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, oldPW, newPW string) (string, error) {
	user, err := s.userRepo.WithCtx(ctx).FindByID(userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPW)); err != nil {
		return "", errors.New("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPW), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("failed to hash password")
	}

	user.PasswordHash = string(hash)
	user.MustChangePW = false

	if err := s.userRepo.WithCtx(ctx).Update(user); err != nil {
		return "", err
	}

	token, err := middleware.GenerateToken(user.ID, user.Role, false)
	if err != nil {
		return "", errors.New("failed to generate token")
	}
	return token, nil
}

// AdminResetPassword lets an administrator overwrite another user's password.
// The target user is forced to change it on next login. Resetting your own
// password through this path is rejected; admins must use ChangePassword.
func (s *AuthService) AdminResetPassword(ctx context.Context, adminID, targetID uint, newPassword string) error {
	if adminID == targetID {
		return errors.New("cannot reset your own password through this endpoint; use change-password instead")
	}
	repo := s.userRepo.WithCtx(ctx)
	if _, err := repo.FindByID(targetID); err != nil {
		return errors.New("target user not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}
	return repo.UpdatePasswordAndForceChange(targetID, string(hash))
}
