package service

import (
	"context"
	"errors"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPatientService(t *testing.T) *PatientService {
	db := newTestDB(t)
	return NewPatientService(repository.NewPatientRepository(db))
}

func TestPatientService_Create_NormalizesIDNumber(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name:     "Alice",
		Phone:    "0900000000",
		IDType:   "passport",
		IDNumber: "  ab12cd  ", // 含空白與小寫
	})
	require.NoError(t, err)
	assert.Equal(t, "AB12CD", p.IDNumber, "IDNumber should be trimmed + uppercased")
	assert.Equal(t, "Alice", p.Name)
}

func TestPatientService_Create_DuplicateRejected(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "A", Phone: "1", IDType: "passport", IDNumber: "XYZ",
	})
	require.NoError(t, err)

	// 完全相同：應拒絕
	_, err = svc.Create(ctx, dto.CreatePatientRequest{
		Name: "B", Phone: "2", IDType: "passport", IDNumber: "XYZ",
	})
	assert.True(t, errors.Is(err, ErrPatientDuplicate),
		"expected ErrPatientDuplicate, got %v", err)

	// 大小寫不同也應被視為重複（因為入庫前 uppercase）
	_, err = svc.Create(ctx, dto.CreatePatientRequest{
		Name: "C", Phone: "3", IDType: "passport", IDNumber: "xyz",
	})
	assert.True(t, errors.Is(err, ErrPatientDuplicate),
		"case-insensitive duplicate should be rejected, got %v", err)
}

func TestPatientService_Create_DifferentIDTypeAllowed(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "A", Phone: "1", IDType: "passport", IDNumber: "100",
	})
	require.NoError(t, err)

	// 同 IDNumber 但不同 IDType：允許
	_, err = svc.Create(ctx, dto.CreatePatientRequest{
		Name: "B", Phone: "2", IDType: "hn", IDNumber: "100",
	})
	assert.NoError(t, err)
}

func TestPatientService_Update_SelfExclude(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "A", Phone: "1", IDType: "passport", IDNumber: "111",
	})
	require.NoError(t, err)

	// 編輯自己保留同 IDNumber：應成功（自我排除）
	updated, err := svc.Update(ctx, p.ID, dto.UpdatePatientRequest{
		Name: "A2", Phone: "1", IDType: "passport", IDNumber: "111",
	})
	require.NoError(t, err)
	assert.Equal(t, "A2", updated.Name)
}

func TestPatientService_Update_CollidesWithOther(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "A", Phone: "1", IDType: "passport", IDNumber: "111",
	})
	require.NoError(t, err)
	p2, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "B", Phone: "2", IDType: "passport", IDNumber: "222",
	})
	require.NoError(t, err)

	// 把 p2 的 IDNumber 改成 p1 的 → 應拒絕
	_, err = svc.Update(ctx, p2.ID, dto.UpdatePatientRequest{
		Name: "B", Phone: "2", IDType: "passport", IDNumber: "111",
	})
	assert.True(t, errors.Is(err, ErrPatientDuplicate))
}

func TestPatientService_Update_NotFound(t *testing.T) {
	svc := newPatientService(t)
	_, err := svc.Update(context.Background(), 999, dto.UpdatePatientRequest{
		Name: "x", Phone: "x", IDType: "passport", IDNumber: "x",
	})
	assert.True(t, errors.Is(err, ErrPatientNotFound))
}

func TestPatientService_Delete_NotFound(t *testing.T) {
	svc := newPatientService(t)
	err := svc.Delete(context.Background(), 999)
	assert.True(t, errors.Is(err, ErrPatientNotFound))
}

func TestPatientService_GetHistory_EmptyButValid(t *testing.T) {
	svc := newPatientService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, dto.CreatePatientRequest{
		Name: "A", Phone: "1", IDType: "passport", IDNumber: "777",
	})
	require.NoError(t, err)

	hist, err := svc.GetHistory(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, hist.Patient.ID)
	// Stage 2 contract: history is empty placeholder
	assert.Empty(t, hist.History)
}

func TestPatientService_NormalizeIDNumber(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"abc", "ABC"},
		{"  abc  ", "ABC"},
		{"AbC123", "ABC123"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, normalizeIDNumber(tt.in), "input=%q", tt.in)
	}
}
