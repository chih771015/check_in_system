package model

import "time"

// DiagnosisPhoto stores a single photo uploaded by a translator after seeing
// a patient. Tied to SchedulePatient — at most 3 photos per (schedule, patient).
//
// Schema only in stage 3; the upload/list workflow is wired in stage 4.
type DiagnosisPhoto struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SchedulePatientID uint      `gorm:"not null;index" json:"schedulePatientId"`
	PhotoURL          string    `gorm:"type:varchar(500);not null" json:"photoUrl"`
	UploadedAt        time.Time `json:"uploadedAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

func (DiagnosisPhoto) TableName() string {
	return "diagnosis_photos"
}
