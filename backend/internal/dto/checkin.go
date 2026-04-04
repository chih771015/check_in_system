package dto

import "time"

// CheckinRequest is the payload for a normal check-in (photos via multipart form).
type CheckinRequest struct {
	ScheduleID uint   `form:"scheduleId" binding:"required"`
	Type       string `form:"type" binding:"required,oneof=arrive leave"`
}

// CheckinMakeupRequest is the payload for a makeup check-in.
type CheckinMakeupRequest struct {
	ScheduleID   uint   `form:"scheduleId" binding:"required"`
	Type         string `form:"type" binding:"required,oneof=arrive leave"`
	MakeupReason string `form:"makeupReason" binding:"required"`
}

// CheckinResponse is returned after a successful check-in.
type CheckinResponse struct {
	ID             uint      `json:"id"`
	ScheduleID     uint      `json:"scheduleId"`
	TranslatorID   uint      `json:"translatorId"`
	TranslatorName string    `json:"translatorName"`
	Type           string    `json:"type"`
	CheckinTime    time.Time `json:"checkinTime"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Address        string    `json:"address"`
	SelfieURL      string    `json:"selfieUrl"`
	EnvironmentURL string    `json:"environmentUrl"`
	IsMakeup       bool      `json:"isMakeup"`
	MakeupReason   string    `json:"makeupReason"`
	CreatedAt      time.Time `json:"createdAt"`
}
