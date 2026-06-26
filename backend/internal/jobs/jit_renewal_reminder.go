package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// StartJITRenewalReminderJob sends reminders for grants expiring soon.
func StartJITRenewalReminderJob(ctx context.Context, db *sql.DB, interval time.Duration, notify func(ctx context.Context, userID, message string) error) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// TODO: Replace SQL with Hasura GraphQL query:
				// query GetExpiringGrants($now: timestamptz!, $oneHourLater: timestamptz!) {
				//   jit_addon_grant(where: {
				//     status: {_eq: "active"},
				//     expires_at: {_gt: $now, _lte: $oneHourLater}
				//   }) {
				//     user_id
				//     expires_at
				//   }
				// }
				// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
				rows, err := db.QueryContext(ctx, `SELECT user_id, expires_at FROM jit_addon_grant WHERE status = 'active' AND expires_at > $1 AND expires_at <= $2`, time.Now(), time.Now().Add(1*time.Hour))
				if err != nil {
					log.Printf("[JIT Renewal Reminder] error: %v", err)
					continue
				}
				defer rows.Close()
				for rows.Next() {
					var userID string
					var expiresAt time.Time
					if err := rows.Scan(&userID, &expiresAt); err == nil {
						msg := "Your JIT access expires at " + expiresAt.Format(time.RFC1123) + ". Click to renew."
						notify(ctx, userID, msg)
					}
				}
			}
		}
	}()
}
