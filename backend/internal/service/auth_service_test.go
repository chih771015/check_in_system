package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// newAuthServiceWithUser sets up an in-memory DB with a single user whose
// password is "Password123!" by default. Returns the service, the repo, and
// the seeded user.
func newAuthServiceWithUser(t *testing.T, opts ...func(*model.User)) (*AuthService, *repository.UserRepository, *model.User) {
	t.Helper()
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)

	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	require.NoError(t, err)

	u := &model.User{
		Email:        "alice@example.com",
		PasswordHash: string(hash),
		Name:         "Alice",
		Role:         "translator",
		Status:       "active",
	}
	for _, opt := range opts {
		opt(u)
	}
	require.NoError(t, repo.Create(u))

	return NewAuthService(repo), repo, u
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, _, u := newAuthServiceWithUser(t)

	resp, err := svc.Login(context.Background(), u.Email, "Password123!")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, u.ID, resp.User.ID)
	assert.Equal(t, u.Email, resp.User.Email)
	assert.Equal(t, "translator", resp.User.Role)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, repo, u := newAuthServiceWithUser(t)

	_, err := svc.Login(context.Background(), u.Email, "wrong")
	assert.True(t, errors.Is(err, ErrInvalidCredentials))

	// login_attempts should be incremented
	reloaded, _ := repo.FindByID(u.ID)
	assert.Equal(t, 1, reloaded.LoginAttempts)
}

func TestAuthService_Login_UnknownEmail_ReturnsInvalidCredentials(t *testing.T) {
	svc, _, _ := newAuthServiceWithUser(t)

	// Security: do NOT leak whether the email exists; should return same code as wrong password.
	_, err := svc.Login(context.Background(), "nonexistent@example.com", "anything")
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
}

func TestAuthService_Login_DisabledAccount(t *testing.T) {
	svc, _, u := newAuthServiceWithUser(t, func(m *model.User) {
		m.Status = "disabled"
	})

	_, err := svc.Login(context.Background(), u.Email, "Password123!")
	assert.True(t, errors.Is(err, ErrAccountDisabled))
}

func TestAuthService_Login_Locked(t *testing.T) {
	until := time.Now().Add(10 * time.Minute)
	svc, _, u := newAuthServiceWithUser(t, func(m *model.User) {
		m.LockedUntil = &until
	})

	_, err := svc.Login(context.Background(), u.Email, "Password123!")
	assert.True(t, errors.Is(err, ErrAccountLocked))
}

func TestAuthService_Login_AutoLockAfterMaxAttempts(t *testing.T) {
	svc, repo, u := newAuthServiceWithUser(t, func(m *model.User) {
		m.LoginAttempts = 4 // 第 5 次失敗將觸發鎖定（cfg MaxLoginAttempts = 5）
	})

	_, err := svc.Login(context.Background(), u.Email, "wrong")
	assert.True(t, errors.Is(err, ErrInvalidCredentials))

	reloaded, _ := repo.FindByID(u.ID)
	require.NotNil(t, reloaded.LockedUntil, "account should now be locked")
	assert.True(t, reloaded.LockedUntil.After(time.Now()))
}

func TestAuthService_Login_SuccessResetsAttempts(t *testing.T) {
	svc, repo, u := newAuthServiceWithUser(t, func(m *model.User) {
		m.LoginAttempts = 3
	})

	_, err := svc.Login(context.Background(), u.Email, "Password123!")
	require.NoError(t, err)

	reloaded, _ := repo.FindByID(u.ID)
	assert.Equal(t, 0, reloaded.LoginAttempts)
	assert.Nil(t, reloaded.LockedUntil)
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	svc, repo, u := newAuthServiceWithUser(t, func(m *model.User) {
		m.MustChangePW = true
	})

	token, err := svc.ChangePassword(context.Background(), u.ID, "Password123!", "NewPassword456!")
	require.NoError(t, err)
	assert.NotEmpty(t, token, "should return fresh token")

	// DB 端應寫入新 hash 且 must_change_pw 變 false
	reloaded, _ := repo.FindByID(u.ID)
	assert.False(t, reloaded.MustChangePW)
	// 用新密碼驗證新 hash 應成功
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(reloaded.PasswordHash), []byte("NewPassword456!")))
}

func TestAuthService_ChangePassword_WrongOldPassword(t *testing.T) {
	svc, _, u := newAuthServiceWithUser(t)

	_, err := svc.ChangePassword(context.Background(), u.ID, "wrong", "AnyNew123!")
	assert.True(t, errors.Is(err, ErrOldPasswordIncorrect))
}

func TestAuthService_ChangePassword_UserNotFound(t *testing.T) {
	svc, _, _ := newAuthServiceWithUser(t)

	_, err := svc.ChangePassword(context.Background(), 99999, "anything", "AnyNew123!")
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestAuthService_AdminResetPassword_CannotResetSelf(t *testing.T) {
	svc, _, u := newAuthServiceWithUser(t)

	err := svc.AdminResetPassword(context.Background(), u.ID, u.ID, "Whatever123!")
	assert.True(t, errors.Is(err, ErrCannotResetSelf))
}

func TestAuthService_AdminResetPassword_TargetNotFound(t *testing.T) {
	svc, _, admin := newAuthServiceWithUser(t)

	err := svc.AdminResetPassword(context.Background(), admin.ID, 99999, "NewPass123!")
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestAuthService_AdminResetPassword_Success(t *testing.T) {
	svc, repo, admin := newAuthServiceWithUser(t)

	// Seed a target translator
	hash, _ := bcrypt.GenerateFromPassword([]byte("OldPass"), bcrypt.MinCost)
	target := &model.User{
		Email: "bob@example.com", PasswordHash: string(hash), Name: "Bob",
		Role: "translator", Status: "active",
	}
	require.NoError(t, repo.Create(target))

	err := svc.AdminResetPassword(context.Background(), admin.ID, target.ID, "BrandNewPass!")
	require.NoError(t, err)

	reloaded, _ := repo.FindByID(target.ID)
	assert.True(t, reloaded.MustChangePW, "target must_change_pw should be true after admin reset")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(reloaded.PasswordHash), []byte("BrandNewPass!")))
}
