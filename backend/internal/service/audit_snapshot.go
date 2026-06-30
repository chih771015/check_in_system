package service

import (
	"encoding/json"
	"time"

	"translator-checkin/internal/model"
)

// auditChange is the structured JSON payload stored in AuditLog.Detail so the
// admin operation log can show what a record looked like before an update or
// delete (and after, for updates). Sensitive fields such as password hashes are
// never included — the snapshot* helpers whitelist safe fields only.
type auditChange struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

// auditDetailJSON marshals a before/after change set into a compact JSON string.
// On marshal failure it returns "" so audit logging degrades gracefully and
// never blocks the primary operation.
func auditDetailJSON(before, after any) string {
	b, err := json.Marshal(auditChange{Before: before, After: after})
	if err != nil {
		return ""
	}
	return string(b)
}

// snapshotUser returns a redacted, JSON-able view of a user. PasswordHash and
// other security-sensitive fields are intentionally excluded.
func snapshotUser(u *model.User) map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":     u.ID,
		"email":  u.Email,
		"name":   u.Name,
		"phone":  u.Phone,
		"role":   u.Role,
		"status": u.Status,
	}
}

// snapshotSchedule returns a JSON-able view of a schedule's core fields.
func snapshotSchedule(s *model.Schedule) map[string]any {
	if s == nil {
		return nil
	}
	m := map[string]any{
		"id":           s.ID,
		"translatorId": s.TranslatorID,
		"date":         s.Date.Format("2006-01-02"),
		"startTime":    s.StartTime,
		"endTime":      s.EndTime,
		"location":     s.Location,
		"note":         s.Note,
	}
	if s.PatientName != nil {
		m["patientName"] = *s.PatientName
	}
	if s.RecurrenceGroupID != nil {
		m["recurrenceGroupId"] = *s.RecurrenceGroupID
	}
	return m
}

// snapshotPatient returns a JSON-able view of a patient's core fields.
func snapshotPatient(p *model.Patient) map[string]any {
	if p == nil {
		return nil
	}
	return map[string]any{
		"id":       p.ID,
		"name":     p.Name,
		"phone":    p.Phone,
		"idType":   p.IDType,
		"idNumber": p.IDNumber,
	}
}

// snapshotCheckin returns a JSON-able view of a checkin's core fields.
func snapshotCheckin(c *model.Checkin) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"id":           c.ID,
		"scheduleId":   c.ScheduleID,
		"translatorId": c.TranslatorID,
		"type":         c.Type,
		"checkinTime":  c.CheckinTime.Format(time.RFC3339),
		"address":      c.Address,
		"makeupReason": c.MakeupReason,
	}
}
