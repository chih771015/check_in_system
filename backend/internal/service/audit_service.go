package service

import (
	"context"

	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"
)

// AuditService records and queries administrative audit logs.
type AuditService struct {
	repo     *repository.AuditLogRepository
	userRepo *repository.UserRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo *repository.AuditLogRepository, userRepo *repository.UserRepository) *AuditService {
	return &AuditService{repo: repo, userRepo: userRepo}
}

// Log records an administrative action. Errors are swallowed — audit logging
// must never block the primary operation.
func (s *AuditService) Log(ctx context.Context, adminID uint, action, targetType string, targetID uint, detail string) {
	adminName := ""
	if s.userRepo != nil {
		if u, err := s.userRepo.WithCtx(ctx).FindByID(adminID); err == nil {
			adminName = u.Name
		}
	}
	_ = s.repo.WithCtx(ctx).Create(&model.AuditLog{
		AdminID:    adminID,
		AdminName:  adminName,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
	})
}

// List returns audit logs with pagination.
func (s *AuditService) List(ctx context.Context, f repository.AuditLogFilter) ([]model.AuditLog, int64, error) {
	return s.repo.WithCtx(ctx).List(f)
}
