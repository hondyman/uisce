package finops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ThresholdTier string

const (
	Threshold80Percent  ThresholdTier = "WARNING_80"
	Threshold95Percent  ThresholdTier = "CRITICAL_95"
	Threshold100Percent ThresholdTier = "EXCEEDED_100"
)

type TenantBudgetAlertConfig struct {
	ConfigID            uuid.UUID `db:"config_id"`
	TenantID            uuid.UUID `db:"tenant_id"`
	SlackWebhookURL     *string   `db:"slack_webhook_url"`
	EmailRecipientsJSON []byte    `db:"email_notification_recipients"`
	WarningThresholdPct float64   `db:"warning_threshold_pct"`
	CriticalThresholdPct float64  `db:"critical_threshold_pct"`
	IsActive            bool      `db:"is_active"`
}

type BudgetAlertService struct {
	db         *sqlx.DB
	httpClient *http.Client
}

func NewBudgetAlertService(db *sqlx.DB) *BudgetAlertService {
	return &BudgetAlertService{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EvaluateTenantBudgetAndAlert checks monthly spend against quotas and fires webhooks at 80% and 95%
func (s *BudgetAlertService) EvaluateTenantBudgetAndAlert(
	ctx context.Context,
	tenantID uuid.UUID,
	billingPeriod string, // YYYY-MM
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return nil
	}

	// 1. Fetch Tenant Quota & Current Spend
	var quota struct {
		MonthlyBudgetUSD float64 `db:"monthly_budget_usd"`
		CurrentSpendUSD  float64 `db:"current_spend_usd"`
	}
	quotaQuery := `
		SELECT monthly_budget_usd, current_spend_usd 
		FROM finops.tenant_compute_quotas 
		WHERE tenant_id = $1 AND billing_period = $2;
	`
	err := s.db.GetContext(ctx, &quota, quotaQuery, tenantID, billingPeriod)
	if err != nil {
		return err
	}

	if quota.MonthlyBudgetUSD <= 0 {
		return nil
	}

	spendPct := (quota.CurrentSpendUSD / quota.MonthlyBudgetUSD) * 100.0

	// 2. Fetch Alert Configuration (Rule 1: Config-Before-Code)
	var config TenantBudgetAlertConfig
	configQuery := `
		SELECT config_id, tenant_id, slack_webhook_url, email_notification_recipients,
		       warning_threshold_pct, critical_threshold_pct, is_active
		FROM finops.budget_alert_configurations
		WHERE tenant_id = $1 AND is_active = TRUE;
	`
	err = s.db.GetContext(ctx, &config, configQuery, tenantID)
	if err != nil {
		return nil // No active alert configuration found for tenant
	}

	// 3. Determine Required Threshold Alert
	var targetTier ThresholdTier
	if spendPct >= 100.0 {
		targetTier = Threshold100Percent
	} else if spendPct >= config.CriticalThresholdPct {
		targetTier = Threshold95Percent
	} else if spendPct >= config.WarningThresholdPct {
		targetTier = Threshold80Percent
	} else {
		return nil // Spend within normal bounds
	}

	// 4. Check If Already Alerted for this Tier in the Current Period
	var alreadyAlerted bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM finops.budget_alert_history
			WHERE tenant_id = $1 AND billing_period = $2 AND threshold_tier = $3 AND delivery_status = 'SENT'
		);
	`
	_ = s.db.GetContext(ctx, &alreadyAlerted, checkQuery, tenantID, billingPeriod, string(targetTier))
	if alreadyAlerted {
		return nil
	}

	// 5. Dispatch Slack Webhook
	if config.SlackWebhookURL != nil && *config.SlackWebhookURL != "" {
		slackErr := s.sendSlackAlert(ctx, *config.SlackWebhookURL, tenantID, billingPeriod, targetTier, spendPct, quota.CurrentSpendUSD, quota.MonthlyBudgetUSD)
		status := "SENT"
		errStr := ""
		if slackErr != nil {
			status = "FAILED"
			errStr = slackErr.Error()
		}

		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO finops.budget_alert_history (
				tenant_id, billing_period, threshold_tier, spend_percentage,
				current_spend_usd, budget_limit_usd, channel_type, delivery_status, error_message
			) VALUES ($1, $2, $3, $4, $5, $6, 'SLACK_WEBHOOK', $7, $8);
		`, tenantID, billingPeriod, string(targetTier), spendPct, quota.CurrentSpendUSD, quota.MonthlyBudgetUSD, status, errStr)
	}

	return nil
}

func (s *BudgetAlertService) sendSlackAlert(
	ctx context.Context,
	webhookURL string,
	tenantID uuid.UUID,
	period string,
	tier ThresholdTier,
	spendPct, currentSpend, budget float64,
) error {
	color := "#f59e0b" // Amber for 80%
	title := fmt.Sprintf("⚠️ FinOps Budget Alert: 80%% Threshold Reached (%s)", period)
	if tier == Threshold95Percent || tier == Threshold100Percent {
		color = "#ef4444" // Red for 95%+
		title = fmt.Sprintf("🚨 URGENT FinOps Budget Alert: %s (%s)", tier, period)
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": title,
				"fields": []map[string]interface{}{
					{"title": "Tenant ID", "value": tenantID.String(), "short": true},
					{"title": "Spend Ratio", "value": fmt.Sprintf("%.1f%% of Budget", spendPct), "short": true},
					{"title": "Current Spend", "value": fmt.Sprintf("$%.2f", currentSpend), "short": true},
					{"title": "Monthly Budget", "value": fmt.Sprintf("$%.2f", budget), "short": true},
				},
				"footer": "Uisce Semantic OS • FinOps Cost Governor",
				"ts":     time.Now().Unix(),
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
