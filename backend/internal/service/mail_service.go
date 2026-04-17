package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"translator-checkin/internal/config"
)

// MailService sends emails via SMTP. It is intentionally minimal: PLAIN auth,
// optional single attachment, plain-text body.
type MailService struct{}

// NewMailService creates a new MailService.
func NewMailService() *MailService {
	return &MailService{}
}

// Attachment represents a single file attached to an outgoing email.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Send delivers an email through the configured SMTP server. It returns an
// error if SMTP isn't configured rather than silently succeeding.
func (m *MailService) Send(to, subject, body string, att *Attachment) error {
	cfg := config.AppConfig
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return errors.New("SMTP is not configured (set SMTP_HOST and SMTP_FROM env vars)")
	}
	if to == "" {
		return errors.New("recipient email is empty")
	}

	boundary := "=_TC_BOUNDARY_8f3a1b"
	var msg bytes.Buffer

	msg.WriteString(fmt.Sprintf("From: %s\r\n", cfg.SMTPFrom))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(subject)))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if att == nil {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(body)
	} else {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

		// text part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		msg.WriteString(body)
		msg.WriteString("\r\n")

		// attachment part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename))
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", att.Filename))
		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// wrap base64 to 76-char lines as per RFC 2045
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			msg.WriteString(encoded[i:end])
			msg.WriteString("\r\n")
		}
		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	}

	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{to}, msg.Bytes())
}

// encodeSubject wraps non-ASCII subject lines using RFC 2047 base64 encoding so
// that Chinese characters render correctly in mail clients.
func encodeSubject(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return strings.TrimSpace(s)
}
