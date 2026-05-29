package service

import (
	"context"
	"errors"
	"testing"

	"translator-checkin/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetPhotos returns the diagnosis photo URLs attached to a SchedulePatient.
// Admin-side helper used by the schedule detail modal.

func TestDiagnosisService_GetPhotos_Success(t *testing.T) {
	fx := newDiagFixture(t)
	ctx := context.Background()
	require.NoError(t, fx.svc.UploadDiagnosis(ctx, fx.translator.ID, fx.sp.ID, []string{"/u/a.jpg", "/u/b.jpg"}))

	urls, err := fx.svc.GetPhotos(ctx, fx.sp.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"/u/a.jpg", "/u/b.jpg"}, urls)
}

func TestDiagnosisService_GetPhotos_EmptyWhenNoUploads(t *testing.T) {
	fx := newDiagFixture(t)
	urls, err := fx.svc.GetPhotos(context.Background(), fx.sp.ID)
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestDiagnosisService_GetPhotos_SchedulePatientNotFound(t *testing.T) {
	fx := newDiagFixture(t)
	_, err := fx.svc.GetPhotos(context.Background(), 99999)
	assert.True(t, errors.Is(err, ErrSchedulePatientNotFound))
	// Defensive: model constant is referenced to anchor any future schema changes.
	_ = model.SchedulePatientStatusPending
}
