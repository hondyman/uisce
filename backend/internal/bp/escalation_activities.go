package bp

import (
	"context"
	"fmt"
)

type EscalationActivities struct{}

func (a *EscalationActivities) NotifyApproverActivity(
	ctx context.Context,
	approverRole string,
	stepKey string,
	escalationLevel string,
) error {
	// Look up user(s) with this role
	// Send email/Slack/notification
	fmt.Printf("Notifying %s for step %s (escalation: %s)\n", approverRole, stepKey, escalationLevel)
	return nil
}

func (a *EscalationActivities) FinalEscalationActivity(
	ctx context.Context,
	stepKey string,
	lastApprover string,
) error {
	// Notify exec or compliance
	fmt.Printf("FINAL ESCALATION for step %s (was with %s)\n", stepKey, lastApprover)
	return nil
}
