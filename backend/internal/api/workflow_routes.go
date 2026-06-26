package api

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/access"
	"github.com/hondyman/semlayer/backend/internal/jobs"
	"github.com/hondyman/semlayer/backend/internal/queue"
	"github.com/robfig/cron/v3"
)

func registerWorkflowRoutes(r chi.Router, db *sql.DB, cronJob *cron.Cron) {
	// Security & Work Queues
	accessService := access.NewAccessService(db)
	queueService := queue.NewQueueService(db)
	accessHandler := NewAccessHandler(accessService)
	queueHandler := NewQueueHandler(queueService)

	r.Route("/workflows", func(r chi.Router) {
		r.Get("/initiatable", accessHandler.ListInitiatableWorkflows)
		r.Post("/{bpDefId}/can-initiate", accessHandler.CanInitiate)
	})

	r.Route("/my-approvals", func(r chi.Router) {
		r.Get("/", queueHandler.GetMyApprovals)
	})

	r.Route("/instances/{instanceId}", func(r chi.Router) {
		r.Post("/assign-to-me", queueHandler.AssignToMe)
		r.Post("/unassign", queueHandler.Unassign)
	})

	// Background Jobs
	queueJob := jobs.NewQueueRefreshJob(queueService)
	queueJob.RegisterWithCron(cronJob)
}
