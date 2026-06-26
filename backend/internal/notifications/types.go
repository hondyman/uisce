package notifications

import (
	"time"
)

type NotificationRule struct {
	ID              string
	TenantID        string
	BPDefID         *string
	StepKey         *string
	TriggerEvent    string   // step_assigned, sla_warning, sla_breach, approved, rejected
	Channels        []string // email, slack, sms
	TemplateKey     string
	DelaySeconds    int
	RecipientRole   *string // current_approver, initiator, admin
	RecipientUserID *string
	RecipientGroup  *string
	Enabled         bool
}

type NotificationTemplate struct {
	ID            string
	TemplateKey   string
	Name          string
	Subject       string
	BodyText      string
	BodyHTML      string
	SlackTemplate map[string]interface{}
}

type NotificationContext struct {
	InstanceID     string
	BPKey          string
	StepKey        string
	ApplicantName  string
	Amount         string
	Entity         string
	RequesterID    string
	RequesterName  string
	ApproverRole   string
	SLAExpiresAt   time.Time
	HoursRemaining float64
	Message        string
	ApprovalURL    string
	CustomData     map[string]interface{}
}

type EmailClient interface {
	Send(ctx interface{}, toEmail, subject, htmlBody string) error // ctx is context.Context
}

type MessengerClient interface {
	PostMessage(ctx interface{}, userID string, blocks map[string]interface{}) error
}
