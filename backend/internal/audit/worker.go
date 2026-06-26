package audit

import (
	"context"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AsyncAuditService handles asynchronous writing of audit logs
type AsyncAuditService struct {
	tracker    *BitemporalTracker
	changeChan chan EntityChange
	workerWg   sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAsyncAuditService creates a new async audit service
func NewAsyncAuditService(tracker *BitemporalTracker, bufferSize int) *AsyncAuditService {
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncAuditService{
		tracker:    tracker,
		changeChan: make(chan EntityChange, bufferSize),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker goroutines
func (s *AsyncAuditService) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		s.workerWg.Add(1)
		go s.workerLoop(i)
	}
	logging.GetLogger().Sugar().Infof("AsyncAuditService started with %d workers", workerCount)
}

// Stop stops the workers gracefully
func (s *AsyncAuditService) Stop() {
	logging.GetLogger().Sugar().Info("Stopping AsyncAuditService...")
	close(s.changeChan) // Close channel to stop accepting new work
	s.workerWg.Wait()   // Wait for workers to finish processing channel
	s.cancel()          // Cancel context
	logging.GetLogger().Sugar().Info("AsyncAuditService stopped")
}

// TrackChangeAsync enqueues a change for async processing
func (s *AsyncAuditService) TrackChangeAsync(change EntityChange) {
	select {
	case s.changeChan <- change:
		// success
	default:
		// Buffer full, log error or drop (fail open vs fail closed)
		// For audit, we prefer to log error but not block main path?
		// Or maybe we should block if critical?
		// Given user requirement "routed to a queue", let's assume we want to drop and log if completely full to avoid bringing down app
		logging.GetLogger().Sugar().Errorf("Audit queue full! Dropping event for entity %s:%s", change.EntityType, change.EntityID)
	}
}

func (s *AsyncAuditService) workerLoop(workerID int) {
	defer s.workerWg.Done()

	const maxRetries = 3

	for change := range s.changeChan {
		var lastErr error

		// Retry logic with exponential backoff
		for attempt := 0; attempt < maxRetries; attempt++ {
			ctx := context.Background()
			if attempt > 0 {
				// Exponential backoff: 100ms, 200ms, 400ms
				backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
				time.Sleep(backoff)
				logging.GetLogger().Sugar().Infof("[Audit Worker %d] Retrying (attempt %d/%d) for %s:%s",
					workerID, attempt+1, maxRetries, change.EntityType, change.EntityID)
			}

			if err := s.tracker.TrackEntityChange(ctx, change); err != nil {
				lastErr = err
				logging.GetLogger().Sugar().Warnf("[Audit Worker %d] Failed to track entity change (attempt %d/%d): %v",
					workerID, attempt+1, maxRetries, err)
				continue
			}

			// Success
			logging.GetLogger().Sugar().Infof("[Audit Worker %d] Successfully tracked change for %s:%s",
				workerID, change.EntityType, change.EntityID)
			lastErr = nil
			break
		}

		// If all retries failed, log to dead letter
		if lastErr != nil {
			logging.GetLogger().Sugar().Errorf("[Audit Worker %d] DEAD LETTER: Failed to track %s:%s after %d attempts. Error: %v. Change: %+v",
				workerID, change.EntityType, change.EntityID, maxRetries, lastErr, change)
			// In production: would send to dead letter queue (Kafka, SQS, etc.)
		}
	}
}
