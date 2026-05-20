package dto

// SchedulePatientPayload is one entry in the Patients array of a
// create/update schedule request.
type SchedulePatientPayload struct {
	PatientID uint   `json:"patientId" binding:"required"`
	StartTime string `json:"startTime" binding:"required"`
	EndTime   string `json:"endTime" binding:"required"`
}

// SchedulePatientResponse is one entry in the Patients array of a schedule
// response, with denormalised patient identity fields for display.
type SchedulePatientResponse struct {
	ID           uint   `json:"id"`
	PatientID    uint   `json:"patientId"`
	PatientName  string `json:"patientName"`
	PatientPhone string `json:"patientPhone"`
	IDType       string `json:"idType"`
	IDNumber     string `json:"idNumber"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	Status       string `json:"status"`
	NoShowReason string `json:"noShowReason,omitempty"`
}

// CreateScheduleRequest is the payload for creating a new schedule.
//
// Stage 3 introduces the Patients array. PatientName remains optional for
// backward compat; when Patients is non-empty the service uses it as the
// source of truth.
type CreateScheduleRequest struct {
	TranslatorID    uint                     `json:"translatorId" binding:"required"`
	Date            string                   `json:"date" binding:"required"`
	StartTime       string                   `json:"startTime" binding:"required"`
	EndTime         string                   `json:"endTime" binding:"required"`
	Location        string                   `json:"location" binding:"required"`
	PatientName     string                   `json:"patientName"`
	Patients        []SchedulePatientPayload `json:"patients"`
	Note            string                   `json:"note"`
	RecurrenceRule  string                   `json:"recurrenceRule"`  // "daily", "weekly:1,3,5", "monthly:5,20"
	RecurrenceUntil string                   `json:"recurrenceUntil"` // YYYY-MM-DD
}

// UpdateScheduleRequest is the payload for updating a schedule.
// When Patients is non-nil it replaces the entire existing patient list.
type UpdateScheduleRequest struct {
	Date        *string                   `json:"date"`
	StartTime   *string                   `json:"startTime"`
	EndTime     *string                   `json:"endTime"`
	Location    *string                   `json:"location"`
	PatientName *string                   `json:"patientName"`
	Patients    *[]SchedulePatientPayload `json:"patients"`
	Note        *string                   `json:"note"`
}

// ScheduleResponse represents a schedule with checkin status and patient list.
type ScheduleResponse struct {
	ID                uint                      `json:"id"`
	TranslatorID      uint                      `json:"translatorId"`
	TranslatorName    string                    `json:"translatorName"`
	Date              string                    `json:"date"`
	StartTime         string                    `json:"startTime"`
	EndTime           string                    `json:"endTime"`
	Location          string                    `json:"location"`
	PatientName       string                    `json:"patientName"`
	Note              string                    `json:"note"`
	CheckinStatus     string                    `json:"checkinStatus"`
	RecurrenceGroupID *string                   `json:"recurrenceGroupId"`
	Patients          []SchedulePatientResponse `json:"patients"`
}

// ScheduleListQuery holds optional query parameters for listing schedules.
type ScheduleListQuery struct {
	DateFrom     string `form:"dateFrom"`
	DateTo       string `form:"dateTo"`
	TranslatorID string `form:"translatorId"`
	Location     string `form:"location"`
}
