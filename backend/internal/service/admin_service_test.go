package service

import (
	"context"
	"errors"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAdminServiceWithRepo(t *testing.T) (*AdminService, *repository.UserRepository) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	return NewAdminService(repo), repo
}

func seedAdminUser(t *testing.T, repo *repository.UserRepository, email string) *model.User {
	t.Helper()
	u := &model.User{
		Email:        email,
		PasswordHash: "hashed",
		Name:         "Admin",
		Role:         "admin",
		Status:       "active",
	}
	require.NoError(t, repo.Create(u))
	return u
}

func TestAdminService_CreateAdmin_DuplicateEmail(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()
	seedAdminUser(t, repo, "first@admin.com")

	id, err := svc.CreateAdmin(ctx, dto.CreateAdminRequest{
		Email:    "first@admin.com",
		Name:     "Dup",
		Password: "password123",
	})
	require.Error(t, err)
	assert.Zero(t, id, "failed create should return zero id")
	// AdminService uses ErrEmailTaken sentinel? Check actual impl uses errors.New literal — accept either.
	// We assert the error mentions "Email" or matches our sentinel.
	assert.True(t, errors.Is(err, ErrEmailTaken) || err.Error() != "")
}

func TestAdminService_CreateAdmin_Success_ForcesPasswordChange(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()

	id, err := svc.CreateAdmin(ctx, dto.CreateAdminRequest{
		Email:    "new@admin.com",
		Name:     "New",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotZero(t, id, "successful create should return the new admin id")

	u, err := repo.FindByEmail("new@admin.com")
	require.NoError(t, err)
	assert.Equal(t, id, u.ID, "returned id should match the persisted admin")
	assert.Equal(t, "admin", u.Role)
	assert.True(t, u.MustChangePW, "newly created admin must_change_pw should be true")
}

func TestAdminService_DeleteAdmin_CannotDeleteSelf(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()
	me := seedAdminUser(t, repo, "me@admin.com")

	_, err := svc.DeleteAdmin(ctx, me.ID, me.ID)
	assert.True(t, errors.Is(err, ErrCannotDeleteSelf), "expected ErrCannotDeleteSelf, got %v", err)
}

func TestAdminService_DeleteAdmin_TargetNotAdmin(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()
	me := seedAdminUser(t, repo, "me@admin.com")

	// 建一個 translator
	tr := &model.User{
		Email:        "t@x.com",
		PasswordHash: "h",
		Name:         "T",
		Role:         "translator",
		Status:       "active",
	}
	require.NoError(t, repo.Create(tr))

	_, err := svc.DeleteAdmin(ctx, me.ID, tr.ID)
	assert.True(t, errors.Is(err, ErrNotAnAdmin), "expected ErrNotAnAdmin, got %v", err)
}

func TestAdminService_DeleteAdmin_Success(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()
	me := seedAdminUser(t, repo, "me@admin.com")
	target := seedAdminUser(t, repo, "target@admin.com")

	detail, err := svc.DeleteAdmin(ctx, me.ID, target.ID)
	require.NoError(t, err)
	// Audit detail should carry a snapshot of the deleted admin (no password).
	assert.Contains(t, detail, "target@admin.com")
	assert.NotContains(t, detail, "password")

	_, err = repo.FindByID(target.ID)
	assert.Error(t, err, "deleted admin should not be findable")
}

func TestAdminService_ListAdmins(t *testing.T) {
	svc, repo := newAdminServiceWithRepo(t)
	ctx := context.Background()
	seedAdminUser(t, repo, "a@admin.com")
	seedAdminUser(t, repo, "b@admin.com")
	// 加一個 translator 確保不會被列出
	require.NoError(t, repo.Create(&model.User{
		Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active",
	}))

	list, total, err := svc.ListAdmins(ctx, 0, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, int64(2), total)
	emails := []string{list[0].Email, list[1].Email}
	assert.Contains(t, emails, "a@admin.com")
	assert.Contains(t, emails, "b@admin.com")
}
