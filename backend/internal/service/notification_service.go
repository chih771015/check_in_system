package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/repository"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// NotificationService pushes reminders via LINE Messaging API and email.
type NotificationService struct {
	userRepo     *repository.UserRepository
	scheduleRepo *repository.ScheduleRepository
	mailService  *MailService
	httpClient   *http.Client
}

// NewNotificationService creates a new NotificationService.
// The HTTP client is wrapped with otelhttp so every outbound LINE push
// shows up as a client span in Jaeger.
func NewNotificationService(
	userRepo *repository.UserRepository,
	scheduleRepo *repository.ScheduleRepository,
	mailService *MailService,
) *NotificationService {
	return &NotificationService{
		userRepo:     userRepo,
		scheduleRepo: scheduleRepo,
		mailService:  mailService,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// PushLine sends a plain-text push message to a LINE user.
// Requires LINE_CHANNEL_ACCESS_TOKEN to be set.
// Takes a context so the outbound span becomes a child of the caller trace.
func (n *NotificationService) PushLine(ctx context.Context, lineUserID, message string) error {
	cfg := config.AppConfig
	if cfg.LineChannelAccessToken == "" {
		return errors.New("LINE channel access token not configured")
	}
	if lineUserID == "" {
		return errors.New("line user id is empty")
	}
	payload := map[string]any{
		"to": lineUserID,
		"messages": []map[string]string{
			{"type": "text", "text": message},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.LineChannelAccessToken)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("line push failed with status %d", resp.StatusCode)
	}
	return nil
}

// SendScheduleReminders iterates over tomorrow's schedules and sends reminders
// via LINE (if line_user_id is set) and/or email (fallback). Errors from
// individual recipients are logged but do not abort the sweep.
//
// Opens a span named "notification.SendScheduleReminders" so the whole sweep
// (including every outbound LINE push span) hangs off one trace.
func (n *NotificationService) SendScheduleReminders() {
	tracer := otel.Tracer("translator-checkin/service")
	ctx, span := tracer.Start(context.Background(), "notification.SendScheduleReminders")
	defer span.End()

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	schedules, err := n.scheduleRepo.FindAll(0, tomorrow, tomorrow, "")
	if err != nil {
		log.Printf("[notification] failed to list schedules: %v", err)
		return
	}
	if len(schedules) == 0 {
		return
	}
	log.Printf("[notification] sending reminders for %d schedules on %s", len(schedules), tomorrow)
	for _, s := range schedules {
		user, err := n.userRepo.FindByID(s.TranslatorID)
		if err != nil {
			continue
		}
		msg := fmt.Sprintf("【明日排程提醒】\n日期：%s\n時間：%s - %s\n地點：%s\n病患：%s",
			tomorrow, s.StartTime, s.EndTime, s.Location, s.PatientName)

		if user.LineUserID != "" {
			if err := n.PushLine(ctx, user.LineUserID, msg); err != nil {
				log.Printf("[notification] line push failed for user %d: %v", user.ID, err)
			}
		}
		if user.Email != "" && config.AppConfig.SMTPHost != "" {
			if err := n.mailService.Send(user.Email, "【明日排程提醒】"+tomorrow, msg, nil); err != nil {
				log.Printf("[notification] email send failed for user %d: %v", user.ID, err)
			}
		}
	}
}
