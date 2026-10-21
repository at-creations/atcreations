// Package mail sends transactional email. Templates use html/template so all
// customer-supplied values are escaped.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Message struct {
	To, ToName, Subject, HTML, Text string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

// LogSender writes emails to the log; for development only.
type LogSender struct{}

func (LogSender) Send(ctx context.Context, m Message) error {
	slog.InfoContext(ctx, "email (not sent: MAIL_PROVIDER=log)", "subject", m.Subject, "text", m.Text)
	return nil
}

// SendGrid sends via the SendGrid v3 HTTP API.
type SendGrid struct {
	apiKey, from, fromName string
	client                 *http.Client
}

func NewSendGrid(apiKey, from, fromName string) *SendGrid {
	return &SendGrid{apiKey: apiKey, from: from, fromName: fromName, client: &http.Client{Timeout: 10 * time.Second}}
}

func (s *SendGrid) Send(ctx context.Context, m Message) error {
	type addr struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}
	body := map[string]any{
		"personalizations": []any{map[string]any{"to": []addr{{Email: m.To, Name: m.ToName}}}},
		"from":             addr{Email: s.from, Name: s.fromName},
		"subject":          m.Subject,
		"content": []map[string]string{
			{"type": "text/plain", "value": m.Text},
			{"type": "text/html", "value": m.HTML},
		},
		"tracking_settings": map[string]any{"click_tracking": map[string]bool{"enable": false}},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("sendgrid status %d: %s", resp.StatusCode, snippet)
	}
	return nil
}
