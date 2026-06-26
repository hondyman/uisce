package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/notifications"
	"github.com/robfig/cron/v3"
)

type SLAReminderJob struct {
	db       *sql.DB
	notifSvc *notifications.NotificationService
}

func NewSLAReminderJob(db *sql.DB, ns *notifications.NotificationService) *SLAReminderJob {
	return &SLAReminderJob{db: db, notifSvc: ns}
}

// Run every 5 minutes to check for SLA warnings
func (j *SLAReminderJob) Run(ctx context.Context) error {
	// log.Println("Running SLA reminder job...")

	rows, err := j.db.QueryContext(ctx, `
        SELECT i.id FROM workflow_instance i
        WHERE i.status = 'running'
            AND i.sla_expires_at BETWEEN NOW() AND NOW() + INTERVAL '1 hour'
            AND NOT EXISTS (
                SELECT 1 FROM notification_log nl
                WHERE nl.instance_id = i.id 
                AND nl.trigger_event = 'sla_warning'
                AND nl.sent_at > NOW() - INTERVAL '30 minutes'
            )
    `)
	if err != nil {
		return err
	}
	defer rows.Close()

	var instanceIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		instanceIDs = append(instanceIDs, id)
	}

	// Send SLA warning (Simplified: Assuming service handles context building)
	// In real app: Fetch instance details, build context, call SendNotification with specific rule
	for _, instanceID := range instanceIDs {
		log.Printf("Should send SLA warning for %s", instanceID)
		// j.notifSvc.SendNotification(...)
	}

	return nil
}

func (j *SLAReminderJob) RegisterWithCron(c *cron.Cron) {
	c.AddFunc("*/5 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		j.Run(ctx)
	})
}
