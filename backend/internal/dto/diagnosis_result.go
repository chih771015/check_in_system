package dto

import "time"

// DiagnosisResultsQuery captures the optional filters for the admin
// "diagnosis results" overview list.
//
// Status is only honoured when it's "completed" or "no_show" — pending rows
// are never included in this view (they are by definition not yet a result).
type DiagnosisResultsQuery struct {
	DateFrom     string `form:"dateFrom"`     // YYYY-MM-DD inclusive
	DateTo       string `form:"dateTo"`       // YYYY-MM-DD inclusive
	TranslatorID uint   `form:"translatorId"` // 0 = all
	PatientName  string `form:"patientName"`  // LIKE %name%
	Status       string `form:"status"`       // "completed" | "no_show" | "" (=both)
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

// DiagnosisResultEntry is one row in the admin overview list.
//
// One row == one SchedulePatient that has reached a terminal status
// (completed or no_show).
type DiagnosisResultEntry struct {
	SchedulePatientID uint      `json:"schedulePatientId"`
	ScheduleID        uint      `json:"scheduleId"`
	Date              string    `json:"date"`
	StartTime         string    `json:"startTime"`
	EndTime           string    `json:"endTime"`
	Location          string    `json:"location"`
	Note              string    `json:"note"`
	TranslatorID      uint      `json:"translatorId"`
	TranslatorName    string    `json:"translatorName"`
	PatientID         uint      `json:"patientId"`
	PatientName       string    `json:"patientName"`
	PatientPhone      string    `json:"patientPhone"`
	IDType            string    `json:"idType"`
	IDNumber          string    `json:"idNumber"`
	Status            string    `json:"status"`
	NoShowReason      string    `json:"noShowReason,omitempty"`
	DiagnosisPhotos   []string  `json:"diagnosisPhotos"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// DiagnosisResultsResponse wraps the paginated list.
type DiagnosisResultsResponse struct {
	Data     []DiagnosisResultEntry `json:"data"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}
