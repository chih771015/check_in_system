package dto

// CreateScheduleRequest is the payload for creating a new schedule.
type CreateScheduleRequest struct {
	TranslatorID    uint   `json:"translatorId" binding:"required"`
	Date            string `json:"date" binding:"required"`
	StartTime       string `json:"startTime" binding:"required"`
	EndTime         string `json:"endTime" binding:"required"`
	Location        string `json:"location" binding:"required"`
	PatientName     string `json:"patientName" binding:"required"`
	Note            string `json:"note"`
	RecurrenceRule  string `json:"recurrenceRule"`  // e.g. "daily", "weekly:1,3,5", "monthly:5,20"
	RecurrenceUntil string `json:"recurrenceUntil"` // YYYY-MM-DD, required if RecurrenceRule != ""
}

// UpdateScheduleRequest is the payload for updating a schedule.
type UpdateScheduleRequest struct {
	Date        *string `json:"date"`
	StartTime   *string `json:"startTime"`
	EndTime     *string `json:"endTime"`
	Location    *string `json:"location"`
	PatientName *string `json:"patientName"`
	Note        *string `json:"note"`
}

// ScheduleResponse represents a schedule with checkin status.
type ScheduleResponse struct {
	ID                uint    `json:"id"`
	TranslatorID      uint    `json:"translatorId"`
	TranslatorName    string  `json:"translatorName"`
	Date              string  `json:"date"`
	StartTime         string  `json:"startTime"`
	EndTime           string  `json:"endTime"`
	Location          string  `json:"location"`
	PatientName       string  `json:"patientName"`
	Note              string  `json:"note"`
	CheckinStatus     string  `json:"checkinStatus"`
	RecurrenceGroupID *string `json:"recurrenceGroupId,omitempty"`
}

// ScheduleListQuery holds optional query parameters for listing schedules.
type ScheduleListQuery struct {
	DateFrom     string `form:"dateFrom"`
	DateTo       string `form:"dateTo"`
	TranslatorID string `form:"translatorId"`
	Location     string `form:"location"`
}
