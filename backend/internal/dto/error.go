// Package dto defines the API request/response payloads.
//
// error.go centralises the error envelope returned by HTTP handlers along
// with a registry of stable error codes the frontend can translate via i18n.
package dto

// ErrorResponse is the unified shape every non-2xx handler returns.
//
// The frontend treats `Code` as the i18n lookup key and falls back to
// `Message` when no translation exists.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewError constructs a new ErrorResponse.
func NewError(code, message string) ErrorResponse {
	return ErrorResponse{Code: code, Message: message}
}

// Error codes — keep in SCREAMING_SNAKE_CASE. Frontend mirrors these as
// i18n keys under `errors.<CODE>`.
const (
	// Generic
	CodeBadRequest    = "BAD_REQUEST"
	CodeInternal      = "INTERNAL_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeNotFound      = "NOT_FOUND"
	CodeValidation    = "VALIDATION_ERROR"
	CodeNoFieldsToSet = "NO_FIELDS_TO_UPDATE"

	// Auth
	CodeInvalidCredentials    = "INVALID_CREDENTIALS"
	CodeAccountDisabled       = "ACCOUNT_DISABLED"
	CodeAccountLocked         = "ACCOUNT_LOCKED"
	CodeOldPasswordIncorrect  = "OLD_PASSWORD_INCORRECT"
	CodePasswordHashFailed    = "PASSWORD_HASH_FAILED"
	CodeTokenGenerationFailed = "TOKEN_GENERATION_FAILED"
	CodePasswordChangeReq     = "PASSWORD_CHANGE_REQUIRED"
	CodeCannotResetSelf       = "CANNOT_RESET_SELF"
	CodeUserNotFound          = "USER_NOT_FOUND"
	CodeUserContextMissing    = "USER_CONTEXT_MISSING"

	// Admin / Translator
	CodeEmailTaken          = "EMAIL_TAKEN"
	CodeAdminNotFound       = "ADMIN_NOT_FOUND"
	CodeCannotDeleteSelf    = "CANNOT_DELETE_SELF"
	CodeNotAnAdmin          = "NOT_AN_ADMIN"
	CodeInvalidAdminID      = "INVALID_ADMIN_ID"
	CodeTranslatorNotFound  = "TRANSLATOR_NOT_FOUND"
	CodeNotATranslator      = "NOT_A_TRANSLATOR"
	CodeInvalidTranslatorID = "INVALID_TRANSLATOR_ID"
	CodeInvalidStatus       = "INVALID_STATUS"

	// Schedule
	CodeScheduleNotFound      = "SCHEDULE_NOT_FOUND"
	CodeInvalidScheduleID     = "INVALID_SCHEDULE_ID"
	CodeInvalidDate           = "INVALID_DATE"
	CodeInvalidRecurrence     = "INVALID_RECURRENCE"
	CodeRecurrenceUntilReq    = "RECURRENCE_UNTIL_REQUIRED"
	CodeRecurrenceBeforeStart = "RECURRENCE_UNTIL_BEFORE_START"
	CodeNoDatesGenerated      = "NO_RECURRENCE_DATES"
	CodeFileRequired          = "FILE_REQUIRED"
	CodeFileOpenFailed        = "FILE_OPEN_FAILED"
	CodeInvalidExcel          = "INVALID_EXCEL"
	CodeReadRowsFailed        = "READ_ROWS_FAILED"

	// Checkin
	CodeCheckinNotFound       = "CHECKIN_NOT_FOUND"
	CodeInvalidCheckinID      = "INVALID_CHECKIN_ID"
	CodeScheduleNotOwned      = "SCHEDULE_NOT_OWNED"
	CodeDuplicateCheckin      = "DUPLICATE_CHECKIN"
	CodeArriveBeforeLeave     = "ARRIVE_BEFORE_LEAVE"
	CodeArriveVerifyFailed    = "ARRIVE_VERIFY_FAILED"
	CodeSelfieRequired        = "SELFIE_REQUIRED"
	CodeEnvironmentRequired   = "ENVIRONMENT_PHOTO_REQUIRED"
	CodeCheckinCreateFailed   = "CHECKIN_CREATE_FAILED"
	CodeGoogleNotConfigured   = "GOOGLE_NOT_CONFIGURED"
	CodeExportFailed          = "EXPORT_FAILED"

	// Patient
	CodePatientNotFound  = "PATIENT_NOT_FOUND"
	CodePatientDuplicate = "PATIENT_DUPLICATE"
	CodeInvalidPatientID = "INVALID_PATIENT_ID"

	// Stage 4 — Schedule patient / diagnosis
	CodeSchedulePatientsRequired   = "SCHEDULE_PATIENTS_REQUIRED"
	CodeDuplicatePatientInSchedule = "DUPLICATE_PATIENT_IN_SCHEDULE"
	CodePatientTimeOutOfRange      = "PATIENT_TIME_OUT_OF_RANGE"
	CodePatientEndBeforeStart      = "PATIENT_END_BEFORE_START"
	CodeSchedulePatientNotFound    = "SCHEDULE_PATIENT_NOT_FOUND"
	CodeDiagnosisPhotoLimit        = "DIAGNOSIS_PHOTO_LIMIT"
	CodeDiagnosisNotOwned          = "DIAGNOSIS_NOT_OWNED"
	CodeNoShowReasonRequired       = "NO_SHOW_REASON_REQUIRED"
	CodeCheckoutBlockedByPending   = "CHECKOUT_BLOCKED_BY_PENDING"

	// Audit
	CodeInvalidPage = "INVALID_PAGE"
)
