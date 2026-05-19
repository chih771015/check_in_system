package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError_FieldsAndJSON(t *testing.T) {
	e := NewError(CodeInvalidCredentials, "bad creds")
	assert.Equal(t, CodeInvalidCredentials, e.Code)
	assert.Equal(t, "bad creds", e.Message)

	// JSON encoding shape must remain `{"code":"...","message":"..."}`
	b, err := json.Marshal(e)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"code":"INVALID_CREDENTIALS","message":"bad creds"}`, string(b))
}

func TestErrorCodes_Stability(t *testing.T) {
	// Stable error codes that the frontend i18n keys rely on.
	// Changing these breaks translation lookup — keep them in lockstep.
	assert.Equal(t, "EMAIL_TAKEN", CodeEmailTaken)
	assert.Equal(t, "INVALID_CREDENTIALS", CodeInvalidCredentials)
	assert.Equal(t, "PASSWORD_CHANGE_REQUIRED", CodePasswordChangeReq)
	assert.Equal(t, "PATIENT_NOT_FOUND", CodePatientNotFound)
	assert.Equal(t, "PATIENT_DUPLICATE", CodePatientDuplicate)
	assert.Equal(t, "GOOGLE_NOT_CONFIGURED", CodeGoogleNotConfigured)
}
