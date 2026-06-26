package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

// AlertDispatcher dispatches alerts to various channels
type AlertDispatcher struct {
	channels map[string]AlertChannel
}

// AlertChannel is the interface for alert notification channels
type AlertChannel interface {
	Send(ctx context.Context, alert *Alert) error
	Name() string
}

// NewAlertDispatcher creates a new alert dispatcher
func NewAlertDispatcher() *AlertDispatcher {
	return &AlertDispatcher{
		channels: make(map[string]AlertChannel),
	}
}

// RegisterChannel registers a new alert channel
func (d *AlertDispatcher) RegisterChannel(name string, channel AlertChannel) {
	d.channels[name] = channel
}

// Dispatch sends an alert to the specified channels
func (d *AlertDispatcher) Dispatch(ctx context.Context, alert *Alert, channelNames []string) {
	for _, name := range channelNames {
		if channel, ok := d.channels[name]; ok {
			go func(ch AlertChannel) {
				if err := ch.Send(ctx, alert); err != nil {
					log.Printf("Failed to send alert via %s: %v", ch.Name(), err)
				}
			}(channel)
		}
	}
}

// DispatchToAll sends an alert to all registered channels
func (d *AlertDispatcher) DispatchToAll(ctx context.Context, alert *Alert) {
	for _, channel := range d.channels {
		go func(ch AlertChannel) {
			if err := ch.Send(ctx, alert); err != nil {
				log.Printf("Failed to send alert via %s: %v", ch.Name(), err)
			}
		}(channel)
	}
}

// SlackChannel sends alerts to Slack
type SlackChannel struct {
	WebhookURL string
	Channel    string
	Username   string
}

// NewSlackChannel creates a new Slack channel
func NewSlackChannel(webhookURL, channel, username string) *SlackChannel {
	return &SlackChannel{
		WebhookURL: webhookURL,
		Channel:    channel,
		Username:   username,
	}
}

func (s *SlackChannel) Name() string { return "slack" }

func (s *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	// Determine color based on severity
	color := "#36a64f" // green
	switch alert.Severity {
	case string(SeverityCritical):
		color = "#ff0000" // red
	case string(SeverityWarning):
		color = "#ffcc00" // yellow
	}

	// Build Slack message
	payload := map[string]interface{}{
		"channel":  s.Channel,
		"username": s.Username,
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  fmt.Sprintf("[%s] Alert", alert.Severity),
				"text":   alert.Message,
				"footer": "Semlayer Observability",
				"ts":     alert.FiredAt.Unix(),
				"fields": []map[string]interface{}{
					{"title": "Status", "value": alert.Status, "short": true},
					{"title": "Value", "value": fmt.Sprintf("%.2f", alert.Value), "short": true},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EmailChannel sends alerts via email
type EmailChannel struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
	ToAddresses  []string
}

// NewEmailChannel creates a new email channel
func NewEmailChannel(host string, port int, user, password, from string, to []string) *EmailChannel {
	return &EmailChannel{
		SMTPHost:     host,
		SMTPPort:     port,
		SMTPUser:     user,
		SMTPPassword: password,
		FromAddress:  from,
		ToAddresses:  to,
	}
}

func (e *EmailChannel) Name() string { return "email" }

func (e *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	subject := fmt.Sprintf("[%s] Semlayer Alert: %s", alert.Severity, alert.Message[:min(50, len(alert.Message))])

	body := fmt.Sprintf(`
Semlayer Alert Notification

Severity: %s
Status: %s
Message: %s

Alert Details:
- Value: %.2f
- Threshold: %.2f
- Fired At: %s

--
Semlayer Observability Platform
`, alert.Severity, alert.Status, alert.Message, alert.Value, alert.Threshold, alert.FiredAt.Format(time.RFC3339))

	// Build email
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s",
		e.ToAddresses[0], subject, body))

	addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)
	auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)

	err := smtp.SendMail(addr, auth, e.FromAddress, e.ToAddresses, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// WebhookChannel sends alerts to a generic webhook
type WebhookChannel struct {
	URL     string
	Headers map[string]string
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Headers: headers,
	}
}

func (w *WebhookChannel) Name() string { return "webhook" }

func (w *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	payload := map[string]interface{}{
		"id":        alert.ID.String(),
		"tenant_id": alert.TenantID.String(),
		"severity":  alert.Severity,
		"message":   alert.Message,
		"status":    alert.Status,
		"value":     alert.Value,
		"threshold": alert.Threshold,
		"fired_at":  alert.FiredAt.Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PagerDutyChannel sends alerts to PagerDuty
type PagerDutyChannel struct {
	RoutingKey string
}

// NewPagerDutyChannel creates a new PagerDuty channel
func NewPagerDutyChannel(routingKey string) *PagerDutyChannel {
	return &PagerDutyChannel{
		RoutingKey: routingKey,
	}
}

func (p *PagerDutyChannel) Name() string { return "pagerduty" }

func (p *PagerDutyChannel) Send(ctx context.Context, alert *Alert) error {
	eventAction := "trigger"
	if alert.Status == string(AlertResolved) {
		eventAction = "resolve"
	}

	payload := map[string]interface{}{
		"routing_key":  p.RoutingKey,
		"event_action": eventAction,
		"dedup_key":    alert.ID.String(),
		"payload": map[string]interface{}{
			"summary":   alert.Message,
			"severity":  mapSeverityToPD(alert.Severity),
			"source":    "semlayer-observability",
			"timestamp": alert.FiredAt.Format(time.RFC3339),
			"custom_details": map[string]interface{}{
				"value":     alert.Value,
				"threshold": alert.Threshold,
				"tenant_id": alert.TenantID.String(),
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pagerduty event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pagerduty returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func mapSeverityToPD(severity string) string {
	switch severity {
	case string(SeverityCritical):
		return "critical"
	case string(SeverityWarning):
		return "warning"
	default:
		return "info"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
