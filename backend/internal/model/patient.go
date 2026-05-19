package model

import "time"

// Patient represents a clinic patient that translators help during scheduled
// appointments. The combination of (id_type, id_number) is unique — the same
// physical person should only ever exist as a single row.
//
// ID types currently in use:
//   - "passport": passport number
//   - "hn":       hospital number (medical record number)
//   - "unid":     refugee / unidentified ID
//
// IDNumber is stored uppercased to make matching case-insensitive; callers
// must uppercase before insertion/lookup. See service.PatientService.
type Patient struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Phone     string    `gorm:"type:varchar(50);not null" json:"phone"`
	IDType    string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_patient_id_type_number,priority:1" json:"idType"`
	IDNumber  string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_patient_id_type_number,priority:2" json:"idNumber"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName overrides the default GORM table name (it would otherwise be
// "patients" already, but kept explicit for clarity).
func (Patient) TableName() string {
	return "patients"
}
