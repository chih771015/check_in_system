package model

import "time"

// AuditLog records admin actions for accountability.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    uint      `gorm:"not null;index" json:"admin_id"`
	AdminName  string    `gorm:"type:varchar(255)" json:"admin_name"`
	Action     string    `gorm:"type:varchar(100);not null" json:"action"`
	TargetType string    `gorm:"type:varchar(50)" json:"target_type"`
	TargetID   uint      `json:"target_id"`
	Detail     string    `gorm:"type:text" json:"detail"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
