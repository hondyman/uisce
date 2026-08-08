package api

import (
	"database/sql"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/jobs"
	"github.com/hondyman/uisce/backend/internal/notifications"
	"github.com/robfig/cron/v3"
)

func registerNotificationRoutes(r chi.Router, db *sql.DB, cronJob *cron.Cron) {
	emailKey := os.Getenv("SENDGRID_API_KEY")
	if emailKey == "" {
		emailKey = "SG.dev_mock_key"
	}
	emailFrom := os.Getenv("NOTIFICATIONS_FROM_EMAIL")
	if emailFrom == "" {
		emailFrom = "no-reply@test.com"
	}
	emailFromNames := os.Getenv("NOTIFICATIONS_FROM_NAME")
	if emailFromNames == "" {
		emailFromNames = "Workflow"
	}
	emailClient := notifications.NewSendGridClient(emailKey, emailFrom, emailFromNames)

	slackToken := os.Getenv("SLACK_API_TOKEN")
	if slackToken == "" {
		slackToken = "xoxb-dev-mock-token"
	}
	slackClient := notifications.NewSlackClient(slackToken)

	notifService := notifications.NewNotificationService(db, emailClient, slackClient)
	slackHandler := NewSlackHandler(db, slackClient)

	r.Route("/slack", func(r chi.Router) {
		r.Get("/install", slackHandler.InstallSlack)
		r.Post("/callback", slackHandler.SlackCallback)
		r.Post("/interactive", slackHandler.HandleSlackInteraction)
	})

	// SLA Reminder Job
	slaJob := jobs.NewSLAReminderJob(db, notifService)
	slaJob.RegisterWithCron(cronJob)
}
