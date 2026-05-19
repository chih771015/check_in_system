package service

import (
	"testing"

	"translator-checkin/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestMailService_Send_NoSMTPConfig(t *testing.T) {
	initTestConfig()
	// override SMTP fields to be empty
	prev := *config.AppConfig
	defer func() { *config.AppConfig = prev }()
	config.AppConfig.SMTPHost = ""
	config.AppConfig.SMTPFrom = ""

	err := NewMailService().Send("to@x.com", "subject", "body", nil)
	require := assert.New(t)
	require.Error(err)
	require.Contains(err.Error(), "SMTP is not configured")
}

func TestMailService_Send_EmptyRecipient(t *testing.T) {
	initTestConfig()
	prev := *config.AppConfig
	defer func() { *config.AppConfig = prev }()
	config.AppConfig.SMTPHost = "smtp.example.com"
	config.AppConfig.SMTPFrom = "from@example.com"

	err := NewMailService().Send("", "subject", "body", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recipient email is empty")
}

func TestEncodeSubject_ASCIIPassthrough(t *testing.T) {
	got := encodeSubject("Hello World")
	assert.Equal(t, "Hello World", got)
}

func TestEncodeSubject_NonASCIIBase64Encoded(t *testing.T) {
	got := encodeSubject("打卡報表")
	assert.True(t,
		len(got) > 6 && got[:9] == "=?UTF-8?B",
		"non-ASCII subject should be RFC2047 base64-encoded, got %q", got)
	assert.Contains(t, got, "?=")
}
