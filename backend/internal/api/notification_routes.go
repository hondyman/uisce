package api

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/jobs"
	"github.com/hondyman/semlayer/backend/internal/notifications"
	"github.com/robfig/cron/v3"
)

func registerNotificationRoutes(r chi.Router, db *sql.DB, cronJob *cron.Cron) {
	// Notifications
	emailKey := getEnv("SENDGRID_API_KEY", "FAKE_KEY")
	emailFrom := getEnv("NOTIFICATIONS_FROM_EMAIL", "no-reply@test.com")
	emailFromNames := getEnv("NOTIFICATIONS_FROM_NAME", "Workflow")
	emailClient := notifications.NewSendGridClient(emailKey, emailFrom, emailFromNames)

	slackToken := getEnv("SLACK_API_TOKEN", "FAKE_TOKEN")
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
