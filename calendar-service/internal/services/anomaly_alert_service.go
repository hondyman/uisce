package services

import (
	"context"
	"fmt"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// AlertIntegration defines an interface for sending alerts to different channels
type AlertIntegration interface {
	SendAlert(ctx context.Context, recipient, subject, message string) error
	Name() string
}

// AnomalyAlertService manages anomaly alerts
type AnomalyAlertService struct {
	hasuraClient *hasura.Client
	integrations map[string]AlertIntegration
	logger       *logrus.Entry
	tenantConfig map[string]TenantAlertConfig
}

// TenantAlertConfig holds tenant-specific alert routing
type TenantAlertConfig struct {
	EmailRecipients []string
	SlackChannels   []string
	PagerDutyKey    string
}

// NewAnomalyAlertService creates a new anomaly alert service
func NewAnomalyAlertService(hasuraClient *hasura.Client, logger *logrus.Entry) *AnomalyAlertService {
	return &AnomalyAlertService{
		hasuraClient: hasuraClient,
		integrations: make(map[string]AlertIntegration),
		logger:       logger.WithField("component", "anomaly_alert_service"),
		tenantConfig: make(map[string]TenantAlertConfig),
	}
}

// RegisterIntegration adds a new alert channel integration
func (s *AnomalyAlertService) RegisterIntegration(integration AlertIntegration) {
	s.integrations[integration.Name()] = integration
}

// TriggerAlert is called by the AnomalyDetector to initiate alerting workflow
func (s *AnomalyAlertService) TriggerAlert(ctx context.Context, anomalyID string, anomalyType string, severity string, description string) error {
	s.logger.WithFields(logrus.Fields{
		"anomaly_id":   anomalyID,
		"anomaly_type": anomalyType,
		"severity":     severity,
	}).Info("Triggering alert for anomaly")

	// Determine channels based on severity
	channels := []string{"email"}
	if severity == "critical" {
		channels = append(channels, "slack", "pagerduty")
	}

	// Create alert records in the database
	for _, channel := range channels {
		recipient := s.getRecipientForChannel(channel)
		if recipient == "" {
			continue
		}

		err := s.recordAlert(ctx, anomalyID, channel, recipient, description)
		if err != nil {
			s.logger.WithError(err).Errorf("Failed to record alert for channel %s", channel)
			continue
		}

		// Send the actual alert using integration
		if integration, ok := s.integrations[channel]; ok {
			subject := fmt.Sprintf("[%s] Calendar Service Anomaly: %s", severity, anomalyType)
			go func(c string, r string, sub string, msg string) {
				// Fire and forget, or handle retries
				if err := integration.SendAlert(context.Background(), r, sub, msg); err != nil {
					s.logger.WithError(err).Errorf("Failed to send %s alert", c)
				}
			}(channel, recipient, subject, description)
		} else {
			s.logger.Warnf("No integration registered for channel %s", channel)
		}
	}

	return nil
}

func (s *AnomalyAlertService) getRecipientForChannel(channel string) string {
	// In a real system, this would lookup tenant config
	switch channel {
	case "email":
		return "admin@example.com"
	case "slack":
		return "#calendar-alerts"
	case "pagerduty":
		return "pd-routing-key"
	default:
		return ""
	}
}

func (s *AnomalyAlertService) recordAlert(ctx context.Context, anomalyID, channel, recipient, message string) error {
	mutation := `
	mutation RecordAlert($input: anomaly_alerts_insert_input!) {
		insert_anomaly_alerts_one(object: $input) {
			id
		}
	}
	`
	// Simplified; assume default tenant for now or extract from context
	input := map[string]interface{}{
		"anomaly_id": anomalyID,
		"channel":    channel,
		"recipient":  recipient,
		"message":    message,
		"status":     "pending",
	}

	return s.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{
		"input": input,
	}, &struct{}{})
}
