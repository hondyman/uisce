package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

type NotificationService struct {
	db          *sql.DB
	emailClient interface {
		Send(context.Context, string, string, string) error
	}
	slackClient interface {
		PostMessage(context.Context, string, map[string]interface{}) error
	}
}

func NewNotificationService(db *sql.DB, email interface {
	Send(context.Context, string, string, string) error
}, slack interface {
	PostMessage(context.Context, string, map[string]interface{}) error
}) *NotificationService {
	return &NotificationService{db: db, emailClient: email, slackClient: slack}
}

// GetRulesForEvent returns all notification rules for a given trigger event
func (s *NotificationService) GetRulesForEvent(
	ctx context.Context,
	tenantID, bpDefID, stepKey, triggerEvent string,
) ([]NotificationRule, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT 
            id, tenant_id, bp_def_id, step_key, trigger_event, channels,
            template_key, delay_seconds, recipient_role, recipient_user_id,
            recipient_group_id, enabled
        FROM notification_rule
        WHERE tenant_id = $1
            AND trigger_event = $2
            AND enabled = true
            AND (bp_def_id IS NULL OR bp_def_id = $3)
            AND (step_key IS NULL OR step_key = $4)
        ORDER BY bp_def_id DESC, step_key DESC 
    `, tenantID, triggerEvent, bpDefID, stepKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []NotificationRule
	for rows.Next() {
		var rule NotificationRule
		var channels []string
		var bpDefID, stepKey, role, userID, groupID *string

		err := rows.Scan(
			&rule.ID, &rule.TenantID, &bpDefID, &stepKey, &rule.TriggerEvent,
			(*pq.StringArray)(&channels), &rule.TemplateKey, &rule.DelaySeconds,
			&role, &userID, &groupID, &rule.Enabled,
		)
		if err != nil {
			return nil, err
		}

		rule.BPDefID = bpDefID
		rule.StepKey = stepKey
		rule.Channels = channels
		rule.RecipientRole = role
		rule.RecipientUserID = userID
		rule.RecipientGroup = groupID
		rules = append(rules, rule)
	}

	return rules, nil
}

// SendNotification sends a notification via specified channels
func (s *NotificationService) SendNotification(
	ctx context.Context,
	rule NotificationRule,
	instanceID string,
	notifCtx NotificationContext,
) error {
	// Get template
	var tmpl NotificationTemplate
	var slackJSON []byte
	err := s.db.QueryRowContext(ctx, `
        SELECT id, template_key, name, subject, body_text, body_html, slack_template
        FROM notification_template
        WHERE tenant_id = $1 AND template_key = $2
    `, rule.TenantID, rule.TemplateKey).Scan(
		&tmpl.ID, &tmpl.TemplateKey, &tmpl.Name, &tmpl.Subject, &tmpl.BodyText, &tmpl.BodyHTML, &slackJSON,
	)
	if err != nil {
		log.Printf("Template not found: %s", rule.TemplateKey)
		return err
	}
	if slackJSON != nil {
		json.Unmarshal(slackJSON, &tmpl.SlackTemplate)
	}

	// Get recipients (Simplified for MVP)
	// In real app: check RecipientRole vs instance
	recipientUserID := "user123" // Mock
	if rule.RecipientUserID != nil {
		recipientUserID = *rule.RecipientUserID
	}

	// Mock resolve email/slack IDs
	userEmail := "demo@example.com"
	slackUserID := "U123456"

	var successChannels []string
	var failedChannels []string

	// Send Email
	if contains(rule.Channels, "email") && s.emailClient != nil {
		subject := s.renderTemplate(tmpl.Subject, notifCtx)
		body := s.renderTemplate(tmpl.BodyHTML, notifCtx)
		if err := s.emailClient.Send(ctx, userEmail, subject, body); err != nil {
			failedChannels = append(failedChannels, "email")
		} else {
			successChannels = append(successChannels, "email")
		}
	}

	// Send Slack
	if contains(rule.Channels, "slack") && s.slackClient != nil {
		blocks := s.renderSlackBlocks(tmpl.SlackTemplate, notifCtx)
		if err := s.slackClient.PostMessage(ctx, slackUserID, blocks); err != nil {
			failedChannels = append(failedChannels, "slack")
		} else {
			successChannels = append(successChannels, "slack")
		}
	}

	// Log
	logJSON := fmt.Sprintf(`{"msg": "notification sent", "recip": "%s"}`, recipientUserID)
	_, _ = s.db.ExecContext(ctx, `
        INSERT INTO notification_log
        (instance_id, rule_id, trigger_event, recipient_user_id, channels_attempted, channels_succeeded, body, sent_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
    `, instanceID, rule.ID, rule.TriggerEvent, recipientUserID,
		pq.Array(append(successChannels, failedChannels...)), pq.Array(successChannels),
		logJSON)

	return nil
}

func (s *NotificationService) renderTemplate(templateStr string, ctx NotificationContext) string {
	result := templateStr
	result = strings.ReplaceAll(result, "{{applicant_name}}", ctx.ApplicantName)
	result = strings.ReplaceAll(result, "{{amount}}", ctx.Amount)
	result = strings.ReplaceAll(result, "{{entity}}", ctx.Entity)
	result = strings.ReplaceAll(result, "{{approval_url}}", ctx.ApprovalURL)
	result = strings.ReplaceAll(result, "{{hours_remaining}}", fmt.Sprintf("%.1f", ctx.HoursRemaining))
	return result
}

func (s *NotificationService) renderSlackBlocks(template map[string]interface{}, ctx NotificationContext) map[string]interface{} {
	jsonStr, _ := json.Marshal(template)
	jsonStr = []byte(s.renderTemplate(string(jsonStr), ctx))
	var blocks map[string]interface{}
	json.Unmarshal(jsonStr, &blocks)
	return blocks
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
