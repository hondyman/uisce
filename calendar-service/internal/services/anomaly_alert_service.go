package services

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/database"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type AlertIntegration interface {
	SendAlert(ctx context.Context, recipient, subject, message string) error
	Name() string
}

type AnomalyAlertService struct {
	dbClient     *database.Client
	integrations map[string]AlertIntegration
	logger       *logrus.Entry
	tenantConfig map[string]TenantAlertConfig
}

type TenantAlertConfig struct {
	EmailRecipients []string
	SlackChannels   []string
	PagerDutyKey    string
}

func NewAnomalyAlertService(db *database.Client, logger *logrus.Entry) *AnomalyAlertService {
	return &AnomalyAlertService{
		dbClient:     db,
		integrations: make(map[string]AlertIntegration),
		logger:       logger.WithField("component", "anomaly_alert_service"),
		tenantConfig: make(map[string]TenantAlertConfig),
	}
}

func (s *AnomalyAlertService) RegisterIntegration(integration AlertIntegration) {
	s.integrations[integration.Name()] = integration
}

func (s *AnomalyAlertService) TriggerAlert(ctx context.Context, anomalyID string, anomalyType string, severity string, description string) error {
	s.logger.WithFields(logrus.Fields{
		"anomaly_id":   anomalyID,
		"anomaly_type": anomalyType,
		"severity":     severity,
	}).Info("Triggering alert for anomaly")

	channels := []string{"email"}
	if severity == "critical" {
		channels = append(channels, "slack", "pagerduty")
	}

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

		if integration, ok := s.integrations[channel]; ok {
			subject := fmt.Sprintf("[%s] Calendar Service Anomaly: %s", severity, anomalyType)
			go func(c string, r string, sub string, msg string) {
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
	query := `
		INSERT INTO anomaly_alerts (id, anomaly_id, channel, recipient, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.dbClient.Pool().Exec(ctx, query,
		uuid.New(), anomalyID, channel, recipient, message, "pending", time.Now(),
	)

	return err
}
