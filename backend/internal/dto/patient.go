package dto

import "time"

// CreatePatientRequest is the payload for creating a patient record.
// IDType is restricted to passport/hn/unid.
type CreatePatientRequest struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	IDType   string `json:"idType" binding:"required,oneof=passport hn unid"`
	IDNumber string `json:"idNumber" binding:"required"`
}

// UpdatePatientRequest is the payload for editing a patient record.
type UpdatePatientRequest struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	IDType   string `json:"idType" binding:"required,oneof=passport hn unid"`
	IDNumber string `json:"idNumber" binding:"required"`
}

// PatientImportError describes one row that could not be imported.
type PatientImportError struct {
	Row    int    `json:"row"`    // 1-based row number in the sheet (header = 1)
	Reason string `json:"reason"` // why it was skipped (duplicate / invalid / ...)
}

// PatientImportResult summarises a bulk xlsx import.
type PatientImportResult struct {
	Created int                  `json:"created"`
	Skipped int                  `json:"skipped"`
	Errors  []PatientImportError `json:"errors"`
}

// PatientResponse is the full admin-facing view of a patient.
type PatientResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	IDType    string    `json:"idType"`
	IDNumber  string    `json:"idNumber"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// ActualTotal is the all-time sum of actual_amount across this patient's
	// schedule_patient rows. Populated by the list endpoint; 0 elsewhere.
	ActualTotal int64 `json:"actualTotal"`
}

// TranslatorPatientResponse is the trimmed-down translator view. It hides
// admin-only fields like timestamps so translators only see what they need
// to verify the patient in front of them.
type TranslatorPatientResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	IDType   string `json:"idType"`
	IDNumber string `json:"idNumber"`
}

// PatientListQuery captures the query parameters for the admin patient list
// endpoint. Search matches against name / phone / id_number (case-insensitive).
type PatientListQuery struct {
	Search   string `form:"search"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// PatientListResponse wraps the paginated patient list with totals so the
// frontend can render pagination without a second count call.
type PatientListResponse struct {
	Data     []PatientResponse `json:"data"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// PatientHistoryEntry is one row in a patient's visit history.
// Stage 2 returns an empty slice; the real aggregation is implemented in
// stage 4 once SchedulePatient / DiagnosisPhoto exist.
type PatientHistoryEntry struct {
	ScheduleID      uint     `json:"scheduleId"`
	Date            string   `json:"date"`
	StartTime       string   `json:"startTime"`
	EndTime         string   `json:"endTime"`
	Location        string   `json:"location"`
	TranslatorName  string   `json:"translatorName"`
	Status          string   `json:"status"`
	NoShowReason    string   `json:"noShowReason,omitempty"`
	DiagnosisPhotos []string `json:"diagnosisPhotos"`
	PrepaidAmount   int      `json:"prepaidAmount"`
	ActualAmount    int      `json:"actualAmount"`
}

// PatientHistoryResponse wraps the history list with the patient header so the
// frontend can render a single page from one network call.
type PatientHistoryResponse struct {
	Patient PatientResponse       `json:"patient"`
	History []PatientHistoryEntry `json:"history"`
	// ActualTotal is the sum of actual_amount over the returned history entries.
	// With a date range it is the range total; without one it is the all-time total.
	ActualTotal int64 `json:"actualTotal"`
}
