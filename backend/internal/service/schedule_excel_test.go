package service

import (
	"context"
	"testing"

	"translator-checkin/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatchImportSchedulesV2_GroupsByCode covers the stage-3 "flat with
// schedule code merging" format:
//
//   A=Code | B=TranslatorID | C=Date | D=OverallStart | E=OverallEnd |
//   F=Location | G=PatientID | H=PatientStart | I=PatientEnd
//
// Rows sharing the same A merge into one schedule with multiple patients.
func TestBatchImportSchedulesV2_GroupsByCode(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	p2 := fx.seedPatient(t, "P2", "PID2")
	p3 := fx.seedPatient(t, "P3", "PID3")

	rows := []ScheduleImportRowV2{
		// SCH-001 with 2 patients
		{RowNumber: 2, Code: "SCH-001", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L1",
			PatientID: p1.ID, PatientStart: "09:00", PatientEnd: "10:00"},
		{RowNumber: 3, Code: "SCH-001", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L1",
			PatientID: p2.ID, PatientStart: "10:00", PatientEnd: "11:00"},
		// SCH-002 with 1 patient
		{RowNumber: 4, Code: "SCH-002", TranslatorID: fx.translator.ID, Date: "2026-06-02",
			OverallStart: "14:00", OverallEnd: "17:00", Location: "L2",
			PatientID: p3.ID, PatientStart: "14:00", PatientEnd: "15:00"},
	}

	result, err := fx.svc.BatchImportSchedulesV2(context.Background(), rows)
	require.NoError(t, err)
	assert.Equal(t, 2, result.SuccessSchedules)
	assert.Equal(t, 3, result.SuccessPatients)
	assert.Empty(t, result.Failed)
}

func TestBatchImportSchedulesV2_RejectsConflictingMeta(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	p2 := fx.seedPatient(t, "P2", "PID2")

	// Same code but different translatorID → error
	rows := []ScheduleImportRowV2{
		{RowNumber: 2, Code: "SCH-001", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p1.ID, PatientStart: "09:00", PatientEnd: "10:00"},
		{RowNumber: 3, Code: "SCH-001", TranslatorID: fx.translator.ID + 999, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p2.ID, PatientStart: "10:00", PatientEnd: "11:00"},
	}
	result, err := fx.svc.BatchImportSchedulesV2(context.Background(), rows)
	require.NoError(t, err)
	assert.Equal(t, 0, result.SuccessSchedules, "conflicting meta should reject whole group")
	require.Len(t, result.Failed, 1, "the conflicting row should be reported")
	assert.Equal(t, 3, result.Failed[0].RowNumber)
	assert.Contains(t, result.Failed[0].Error, "code")
}

func TestBatchImportSchedulesV2_InvalidPatientInOneGroupSkipsGroupOnly(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")

	rows := []ScheduleImportRowV2{
		// SCH-001 will fail (patient not found)
		{RowNumber: 2, Code: "SCH-001", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: 99999, PatientStart: "09:00", PatientEnd: "10:00"},
		// SCH-002 should still succeed
		{RowNumber: 3, Code: "SCH-002", TranslatorID: fx.translator.ID, Date: "2026-06-02",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p1.ID, PatientStart: "09:00", PatientEnd: "10:00"},
	}
	result, err := fx.svc.BatchImportSchedulesV2(context.Background(), rows)
	require.NoError(t, err)
	assert.Equal(t, 1, result.SuccessSchedules, "second group should still go through")
	assert.Equal(t, 1, result.SuccessPatients)
	assert.Len(t, result.Failed, 1)
}

func TestBatchImportSchedulesV2_EmptyCodeRejected(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p := fx.seedPatient(t, "P", "PID")
	rows := []ScheduleImportRowV2{
		{RowNumber: 2, Code: "", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p.ID, PatientStart: "09:00", PatientEnd: "10:00"},
	}
	result, err := fx.svc.BatchImportSchedulesV2(context.Background(), rows)
	require.NoError(t, err)
	assert.Equal(t, 0, result.SuccessSchedules)
	require.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Error, "code")
}

func TestBatchImportSchedulesV2_PersistsToDB(t *testing.T) {
	fx := newScheduleMultiFixture(t)
	p1 := fx.seedPatient(t, "P1", "PID1")
	p2 := fx.seedPatient(t, "P2", "PID2")

	rows := []ScheduleImportRowV2{
		{RowNumber: 2, Code: "SCH-A", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p1.ID, PatientStart: "09:00", PatientEnd: "10:00"},
		{RowNumber: 3, Code: "SCH-A", TranslatorID: fx.translator.ID, Date: "2026-06-01",
			OverallStart: "09:00", OverallEnd: "12:00", Location: "L",
			PatientID: p2.ID, PatientStart: "10:00", PatientEnd: "11:00"},
	}
	result, err := fx.svc.BatchImportSchedulesV2(context.Background(), rows)
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessSchedules)

	// Verify in DB
	list, err := fx.svc.List(context.Background(), fx.translator.ID, "", "", "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Len(t, list[0].Patients, 2)
	// patients are sorted by start_time
	assert.Equal(t, "P1", list[0].Patients[0].PatientName)
	assert.Equal(t, "P2", list[0].Patients[1].PatientName)
	// schedule_patients table reflects the data
	rowsDB, _ := fx.spRepo.FindByScheduleID(list[0].ID)
	assert.Len(t, rowsDB, 2)
	assert.Equal(t, model.SchedulePatientStatusPending, rowsDB[0].Status)
}
