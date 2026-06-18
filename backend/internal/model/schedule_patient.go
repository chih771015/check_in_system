package model

import "time"

// SchedulePatient links one patient to one schedule with per-patient time slot
// and visit status. A schedule may have many SchedulePatients.
//
// Status enum:
//   - pending    : 翻譯員到達後尚未處理該病人
//   - completed  : 已上傳診斷證明
//   - no_show    : 標記未到
type SchedulePatient struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ScheduleID   uint      `gorm:"not null;index;uniqueIndex:idx_schedule_patient_unique,priority:1" json:"scheduleId"`
	PatientID    uint      `gorm:"not null;index;uniqueIndex:idx_schedule_patient_unique,priority:2" json:"patientId"`
	StartTime    string    `gorm:"type:varchar(5);not null" json:"startTime"`
	EndTime      string    `gorm:"type:varchar(5);not null" json:"endTime"`
	OrderIdx     int       `gorm:"default:0" json:"order"`
	Status       string    `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	NoShowReason string    `gorm:"type:text" json:"noShowReason,omitempty"`
	// Money (integer TWD). PrepaidAmount is set by the admin at scheduling time;
	// ActualAmount is filled by the translator after the visit (0 on no_show).
	PrepaidAmount int       `gorm:"not null;default:0" json:"prepaidAmount"`
	ActualAmount  int       `gorm:"not null;default:0" json:"actualAmount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	Patient Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (SchedulePatient) TableName() string {
	return "schedule_patients"
}

// Status constants for SchedulePatient.
const (
	SchedulePatientStatusPending   = "pending"
	SchedulePatientStatusCompleted = "completed"
	SchedulePatientStatusNoShow    = "no_show"
)
