package handler

import (
	"errors"
	"net/http"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/service"

	"github.com/gin-gonic/gin"
)

// respondError maps a service-layer error to an HTTP status + ErrorResponse.
// Sentinel errors are translated to stable codes the frontend can i18n.
// Unknown errors fall back to INTERNAL_ERROR.
func respondError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	status, code := mapError(err)
	c.JSON(status, dto.NewError(code, err.Error()))
}

// respondBadRequest is a shorthand for binding / validation errors.
func respondBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, dto.NewError(dto.CodeBadRequest, err.Error()))
}

// respondCode replies with a custom code + status (used when no sentinel exists).
func respondCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.NewError(code, message))
}

// mapError matches sentinel errors and returns (httpStatus, errorCode).
func mapError(err error) (int, string) {
	switch {
	// Auth
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized, dto.CodeInvalidCredentials
	case errors.Is(err, service.ErrAccountDisabled):
		return http.StatusForbidden, dto.CodeAccountDisabled
	case errors.Is(err, service.ErrAccountLocked):
		return http.StatusForbidden, dto.CodeAccountLocked
	case errors.Is(err, service.ErrUserNotFound):
		return http.StatusNotFound, dto.CodeUserNotFound
	case errors.Is(err, service.ErrOldPasswordIncorrect):
		return http.StatusBadRequest, dto.CodeOldPasswordIncorrect
	case errors.Is(err, service.ErrPasswordHashFailed):
		return http.StatusInternalServerError, dto.CodePasswordHashFailed
	case errors.Is(err, service.ErrTokenGenFailed):
		return http.StatusInternalServerError, dto.CodeTokenGenerationFailed
	case errors.Is(err, service.ErrCannotResetSelf):
		return http.StatusBadRequest, dto.CodeCannotResetSelf

	// Admin / Translator
	case errors.Is(err, service.ErrEmailTaken):
		return http.StatusConflict, dto.CodeEmailTaken
	case errors.Is(err, service.ErrAdminNotFound):
		return http.StatusNotFound, dto.CodeAdminNotFound
	case errors.Is(err, service.ErrCannotDeleteSelf):
		return http.StatusBadRequest, dto.CodeCannotDeleteSelf
	case errors.Is(err, service.ErrNotAnAdmin):
		return http.StatusBadRequest, dto.CodeNotAnAdmin
	case errors.Is(err, service.ErrTranslatorNotFound):
		return http.StatusNotFound, dto.CodeTranslatorNotFound
	case errors.Is(err, service.ErrNotATranslator):
		return http.StatusBadRequest, dto.CodeNotATranslator
	case errors.Is(err, service.ErrInvalidStatus):
		return http.StatusBadRequest, dto.CodeInvalidStatus

	// Schedule
	case errors.Is(err, service.ErrScheduleNotFound):
		return http.StatusNotFound, dto.CodeScheduleNotFound
	case errors.Is(err, service.ErrInvalidDateFormat):
		return http.StatusBadRequest, dto.CodeInvalidDate
	case errors.Is(err, service.ErrInvalidRecurrence):
		return http.StatusBadRequest, dto.CodeInvalidRecurrence
	case errors.Is(err, service.ErrRecurrenceUntilReq):
		return http.StatusBadRequest, dto.CodeRecurrenceUntilReq
	case errors.Is(err, service.ErrRecurrenceBeforeStart):
		return http.StatusBadRequest, dto.CodeRecurrenceBeforeStart
	case errors.Is(err, service.ErrNoDatesGenerated):
		return http.StatusBadRequest, dto.CodeNoDatesGenerated

	// Checkin
	case errors.Is(err, service.ErrCheckinNotFound):
		return http.StatusNotFound, dto.CodeCheckinNotFound
	case errors.Is(err, service.ErrScheduleNotOwned):
		return http.StatusForbidden, dto.CodeScheduleNotOwned
	case errors.Is(err, service.ErrDuplicateCheckin):
		return http.StatusConflict, dto.CodeDuplicateCheckin
	case errors.Is(err, service.ErrArriveBeforeLeave):
		return http.StatusBadRequest, dto.CodeArriveBeforeLeave
	case errors.Is(err, service.ErrArriveVerifyFailed):
		return http.StatusInternalServerError, dto.CodeArriveVerifyFailed
	case errors.Is(err, service.ErrCheckinCreate):
		return http.StatusInternalServerError, dto.CodeCheckinCreateFailed
	case errors.Is(err, service.ErrNoFieldsToUpdate):
		return http.StatusBadRequest, dto.CodeNoFieldsToSet

	// Patient
	case errors.Is(err, service.ErrPatientDuplicate):
		return http.StatusConflict, dto.CodePatientDuplicate
	case errors.Is(err, service.ErrPatientNotFound):
		return http.StatusNotFound, dto.CodePatientNotFound

	// Stage 4 — Schedule patient / diagnosis
	case errors.Is(err, service.ErrSchedulePatientsRequired):
		return http.StatusBadRequest, dto.CodeSchedulePatientsRequired
	case errors.Is(err, service.ErrDuplicatePatientInSchedule):
		return http.StatusBadRequest, dto.CodeDuplicatePatientInSchedule
	case errors.Is(err, service.ErrPatientTimeOutOfRange):
		return http.StatusBadRequest, dto.CodePatientTimeOutOfRange
	case errors.Is(err, service.ErrPatientEndBeforeStart):
		return http.StatusBadRequest, dto.CodePatientEndBeforeStart
	case errors.Is(err, service.ErrSchedulePatientNotFound):
		return http.StatusNotFound, dto.CodeSchedulePatientNotFound
	case errors.Is(err, service.ErrDiagnosisPhotoLimit):
		return http.StatusBadRequest, dto.CodeDiagnosisPhotoLimit
	case errors.Is(err, service.ErrDiagnosisNotOwned):
		return http.StatusForbidden, dto.CodeDiagnosisNotOwned
	case errors.Is(err, service.ErrDiagnosisPhotoNotFound):
		return http.StatusNotFound, dto.CodeDiagnosisPhotoNotFound
	case errors.Is(err, service.ErrNoShowReasonRequired):
		return http.StatusBadRequest, dto.CodeNoShowReasonRequired
	case errors.Is(err, service.ErrCheckoutBlockedByPending):
		return http.StatusBadRequest, dto.CodeCheckoutBlockedByPending

	default:
		return http.StatusInternalServerError, dto.CodeInternal
	}
}
