package jobs

import (
	"context"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/queue"
	"github.com/robfig/cron/v3"
)

type QueueRefreshJob struct {
	queueService *queue.QueueService
}

func NewQueueRefreshJob(qs *queue.QueueService) *QueueRefreshJob {
	return &QueueRefreshJob{queueService: qs}
}

// Run every 60 seconds to keep work queue fresh
func (j *QueueRefreshJob) Run(ctx context.Context) error {
	// log.Println("Refreshing work queue materialized view...") // verbose logging usually disabled in prod
	if err := j.queueService.RefreshQueuesView(ctx); err != nil {
		log.Printf("Queue refresh failed: %v", err)
		return err
	}
	// log.Println("Queue refresh complete")
	return nil
}

// RegisterWithCron registers this job with the cron scheduler
func (j *QueueRefreshJob) RegisterWithCron(c *cron.Cron) {
	c.AddFunc("*/1 * * * *", func() { // every 1 minute
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := j.Run(ctx); err != nil {
			// log error
		}
	})
}
