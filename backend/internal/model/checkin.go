package model

import "time"

// Checkin represents a translator's check-in record (arrive or leave).
type Checkin struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ScheduleID     uint      `gorm:"not null;index" json:"schedule_id"`
	TranslatorID   uint      `gorm:"not null;index" json:"translator_id"`
	Type           string    `gorm:"type:varchar(10);not null" json:"type"`
	// Indexed: the admin list defaults to ORDER BY checkin_time DESC and filters
	// on a checkin_time range, so this index backs both sort and date filtering.
	CheckinTime    time.Time `gorm:"not null;index" json:"checkin_time"`
	Latitude       float64   `gorm:"type:decimal(10,7)" json:"latitude"`
	Longitude      float64   `gorm:"type:decimal(10,7)" json:"longitude"`
	Address        string    `gorm:"type:varchar(500)" json:"address"`
	SelfieURL      string    `gorm:"type:varchar(500);not null" json:"selfie_url"`
	// EnvironmentURL is nullable since stage 4 — historical rows keep their
	// value, new check-ins do not require an environment photo.
	EnvironmentURL string    `gorm:"type:varchar(500)" json:"environment_url"`
	IsMakeup       bool      `gorm:"default:false" json:"is_makeup"`
	MakeupReason   string    `gorm:"type:text" json:"makeup_reason"`
	CreatedAt      time.Time `json:"created_at"`

	Schedule   Schedule `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	Translator User     `gorm:"foreignKey:TranslatorID" json:"translator,omitempty"`
}

func (Checkin) TableName() string {
	return "checkins"
}
