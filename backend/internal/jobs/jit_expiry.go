package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"
)

// StartJITGrantExpiryJob launches a background goroutine to expire JIT grants every interval.
func StartJITGrantExpiryJob(ctx context.Context, db *sql.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := services.ExpireJITAddonGrants(ctx, db); err != nil {
					log.Printf("[JIT Expiry Job] error: %v", err)
				}
			}
		}
	}()
}
