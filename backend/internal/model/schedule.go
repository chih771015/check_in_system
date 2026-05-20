package model

import "time"

// Schedule represents a translator's appointment schedule. Stage 3 introduces
// multi-patient support via the Patients relation; PatientName is kept as a
// nullable string for backward compatibility with stage 1/2 data.
type Schedule struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TranslatorID      uint      `gorm:"not null;index" json:"translator_id"`
	Date              time.Time `gorm:"type:date;not null" json:"date"`
	StartTime         string    `gorm:"type:varchar(5);not null" json:"start_time"`
	EndTime           string    `gorm:"type:varchar(5);not null" json:"end_time"`
	Location          string    `gorm:"type:varchar(500);not null" json:"location"`
	PatientName       *string   `gorm:"type:varchar(255)" json:"patient_name,omitempty"`
	Note              string    `gorm:"type:text" json:"note"`
	RecurrenceRule    *string   `gorm:"type:varchar(255)" json:"recurrence_rule,omitempty"`
	RecurrenceGroupID *string   `gorm:"type:varchar(36)" json:"recurrence_group_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Translator User              `gorm:"foreignKey:TranslatorID" json:"translator,omitempty"`
	Patients   []SchedulePatient `gorm:"foreignKey:ScheduleID" json:"patients,omitempty"`
}

func (Schedule) TableName() string {
	return "schedules"
}
