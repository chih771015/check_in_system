package service

import (
	"context"
	"testing"

	"translator-checkin/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNotificationService_PushLine_NoToken(t *testing.T) {
	initTestConfig()
	prev := *config.AppConfig
	defer func() { *config.AppConfig = prev }()
	config.AppConfig.LineChannelAccessToken = ""

	svc := NewNotificationService(nil, nil, nil)
	err := svc.PushLine(context.Background(), "U123", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LINE channel access token")
}

func TestNotificationService_PushLine_EmptyUserID(t *testing.T) {
	initTestConfig()
	prev := *config.AppConfig
	defer func() { *config.AppConfig = prev }()
	config.AppConfig.LineChannelAccessToken = "test-token"

	svc := NewNotificationService(nil, nil, nil)
	err := svc.PushLine(context.Background(), "", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line user id")
}
