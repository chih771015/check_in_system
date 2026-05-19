package handler

import (
	"errors"
	"net/http"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestMapError_KnownSentinels(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid credentials", service.ErrInvalidCredentials, http.StatusUnauthorized, dto.CodeInvalidCredentials},
		{"account disabled", service.ErrAccountDisabled, http.StatusForbidden, dto.CodeAccountDisabled},
		{"account locked", service.ErrAccountLocked, http.StatusForbidden, dto.CodeAccountLocked},
		{"user not found", service.ErrUserNotFound, http.StatusNotFound, dto.CodeUserNotFound},
		{"old password incorrect", service.ErrOldPasswordIncorrect, http.StatusBadRequest, dto.CodeOldPasswordIncorrect},
		{"cannot reset self", service.ErrCannotResetSelf, http.StatusBadRequest, dto.CodeCannotResetSelf},
		{"email taken", service.ErrEmailTaken, http.StatusConflict, dto.CodeEmailTaken},
		{"admin not found", service.ErrAdminNotFound, http.StatusNotFound, dto.CodeAdminNotFound},
		{"cannot delete self", service.ErrCannotDeleteSelf, http.StatusBadRequest, dto.CodeCannotDeleteSelf},
		{"not an admin", service.ErrNotAnAdmin, http.StatusBadRequest, dto.CodeNotAnAdmin},
		{"translator not found", service.ErrTranslatorNotFound, http.StatusNotFound, dto.CodeTranslatorNotFound},
		{"not a translator", service.ErrNotATranslator, http.StatusBadRequest, dto.CodeNotATranslator},
		{"invalid status", service.ErrInvalidStatus, http.StatusBadRequest, dto.CodeInvalidStatus},
		{"schedule not found", service.ErrScheduleNotFound, http.StatusNotFound, dto.CodeScheduleNotFound},
		{"invalid date", service.ErrInvalidDateFormat, http.StatusBadRequest, dto.CodeInvalidDate},
		{"invalid recurrence", service.ErrInvalidRecurrence, http.StatusBadRequest, dto.CodeInvalidRecurrence},
		{"checkin not found", service.ErrCheckinNotFound, http.StatusNotFound, dto.CodeCheckinNotFound},
		{"schedule not owned", service.ErrScheduleNotOwned, http.StatusForbidden, dto.CodeScheduleNotOwned},
		{"duplicate checkin", service.ErrDuplicateCheckin, http.StatusConflict, dto.CodeDuplicateCheckin},
		{"arrive before leave", service.ErrArriveBeforeLeave, http.StatusBadRequest, dto.CodeArriveBeforeLeave},
		{"checkin create failed", service.ErrCheckinCreate, http.StatusInternalServerError, dto.CodeCheckinCreateFailed},
		{"no fields to update", service.ErrNoFieldsToUpdate, http.StatusBadRequest, dto.CodeNoFieldsToSet},
		{"patient duplicate", service.ErrPatientDuplicate, http.StatusConflict, dto.CodePatientDuplicate},
		{"patient not found", service.ErrPatientNotFound, http.StatusNotFound, dto.CodePatientNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := mapError(tc.err)
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantCode, code)
		})
	}
}

func TestMapError_UnknownDefaultsToInternal(t *testing.T) {
	status, code := mapError(errors.New("some random error"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, dto.CodeInternal, code)
}

func TestMapError_WrappedSentinel(t *testing.T) {
	// errors.Is should match through wrapping
	wrapped := errors.New("wrapper: " + service.ErrEmailTaken.Error())
	// 純字串 join 不會被 errors.Is 認出，這裡確認 wrapper 走 default
	status, code := mapError(wrapped)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, dto.CodeInternal, code)

	// 用 fmt.Errorf("...%w", ...) 包裝才會被 Is 認出
	wrapped2 := errFromFmtW(service.ErrEmailTaken)
	status2, code2 := mapError(wrapped2)
	assert.Equal(t, http.StatusConflict, status2)
	assert.Equal(t, dto.CodeEmailTaken, code2)
}

// errFromFmtW returns err wrapped with fmt.Errorf's %w verb so errors.Is works.
func errFromFmtW(err error) error {
	return wrapErrorf("context: %w", err)
}

// wrapErrorf is a tiny shim so we don't need to import fmt in tests redundantly.
func wrapErrorf(format string, err error) error {
	return &wrappedErr{msg: format, err: err}
}

type wrappedErr struct {
	msg string
	err error
}

func (w *wrappedErr) Error() string { return w.msg }
func (w *wrappedErr) Unwrap() error { return w.err }
