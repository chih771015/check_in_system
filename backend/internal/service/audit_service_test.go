package service

import (
	"context"
	"testing"

	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuditService(t *testing.T) (*AuditService, *repository.AuditLogRepository, *repository.UserRepository) {
	db := newTestDB(t)
	auditRepo := repository.NewAuditLogRepository(db)
	userRepo := repository.NewUserRepository(db)
	return NewAuditService(auditRepo, userRepo), auditRepo, userRepo
}

func TestAuditService_Log_PersistsWithAdminName(t *testing.T) {
	svc, auditRepo, userRepo := newAuditService(t)
	admin := &model.User{Email: "a@x.com", PasswordHash: "h", Name: "Admin Alice", Role: "admin", Status: "active"}
	require.NoError(t, userRepo.Create(admin))

	svc.Log(context.Background(), admin.ID, "create_translator", "user", 42, "email=foo")

	logs, total, err := auditRepo.List(repository.AuditLogFilter{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	assert.Equal(t, "create_translator", logs[0].Action)
	assert.Equal(t, "Admin Alice", logs[0].AdminName, "admin name should be looked up at log time")
	assert.Equal(t, uint(42), logs[0].TargetID)
}

func TestAuditService_Log_UnknownAdminGetsEmptyName(t *testing.T) {
	svc, auditRepo, _ := newAuditService(t)
	// adminID that doesn't exist — log should still be created
	svc.Log(context.Background(), 99999, "delete_x", "thing", 1, "")

	logs, _, err := auditRepo.List(repository.AuditLogFilter{})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Empty(t, logs[0].AdminName)
}

func TestAuditService_List_FilterByAction(t *testing.T) {
	svc, auditRepo, userRepo := newAuditService(t)
	admin := &model.User{Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "admin", Status: "active"}
	require.NoError(t, userRepo.Create(admin))

	svc.Log(context.Background(), admin.ID, "create_translator", "user", 1, "")
	svc.Log(context.Background(), admin.ID, "delete_translator", "user", 2, "")
	svc.Log(context.Background(), admin.ID, "create_translator", "user", 3, "")

	logs, total, err := auditRepo.List(repository.AuditLogFilter{Action: "create_translator"})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	for _, l := range logs {
		assert.Equal(t, "create_translator", l.Action)
	}
}
