package repository

import (
	"context"

	"translator-checkin/internal/model"

	"gorm.io/gorm"
)

// AuditLogRepository handles database operations for audit logs.
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// WithCtx returns a copy whose *gorm.DB carries the request context so
// the GORM OTel plugin nests SQL spans under the active HTTP span.
func (r *AuditLogRepository) WithCtx(ctx context.Context) *AuditLogRepository {
	return &AuditLogRepository{db: r.db.WithContext(ctx)}
}

// Create inserts a new audit log record.
func (r *AuditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

// AuditLogFilter describes optional filters for listing audit logs.
type AuditLogFilter struct {
	AdminID    uint
	Action     string
	TargetType string
	StartDate  string
	EndDate    string
	Page       int
	PageSize   int
}

// List returns audit logs matching filter with pagination.
func (r *AuditLogRepository) List(f AuditLogFilter) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	q := r.db.Model(&model.AuditLog{})
	if f.AdminID > 0 {
		q = q.Where("admin_id = ?", f.AdminID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.TargetType != "" {
		q = q.Where("target_type = ?", f.TargetType)
	}
	if f.StartDate != "" {
		q = q.Where("created_at >= ?", f.StartDate)
	}
	if f.EndDate != "" {
		q = q.Where("created_at <= ?", f.EndDate)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PageSize

	if err := q.Order("created_at DESC").
		Limit(f.PageSize).Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
