package model

import "time"

// ExportSchedule stores periodic export configuration.
type ExportSchedule struct {
	ID         uint       `gorm:"primaryKey;autoIncrement"`
	AdminID    uint       `gorm:"uniqueIndex;not null"`
	Frequency  string     `gorm:"type:varchar(20);not null"` // "monthly"
	DayOfMonth int        `gorm:"not null"`                  // 1-28
	Format     string     `gorm:"type:varchar(20);not null"` // "excel" | "google_sheet"
	EmailTo    string     `gorm:"type:varchar(255)"`
	Enabled    bool       `gorm:"default:true"`
	LastRunAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ExportSchedule) TableName() string {
	return "export_schedules"
}
