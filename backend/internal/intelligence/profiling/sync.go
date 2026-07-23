package profiling

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type ClientProfile struct {
	UserID       int64
	AnxietyScore float64
}

// SyncHighAnxietyTags runs the profiling analysis and pushes tags to Postgres.
// Uses StarRocks to query Iceberg tables for analytics.
func SyncHighAnxietyTags(ctx context.Context, starrocksDB *sql.DB, pgDB *sql.DB) error {
	// 1. Run the StarRocks Query against Iceberg tables
	// We use ASOF JOIN to correlate user logins with market drawdowns.
	const query = `
		WITH 
			-- 1. Market Stats CTE: Calculate hourly close and max drawdown
			market_stats AS (
				SELECT
					DATE_TRUNC('hour', timestamp) as bucket_time,
					LAST_VALUE(price) OVER (
						PARTITION BY symbol, DATE_TRUNC('hour', timestamp)
						ORDER BY timestamp
					) as close_price,
					MAX(price) OVER (
						PARTITION BY symbol 
						ORDER BY timestamp 
						ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
					) as running_max,
					-- Calculate Drawdown % (Negative value)
					CASE WHEN running_max > 0 THEN (close_price - running_max) / running_max ELSE 0 END as drawdown_pct
				FROM iceberg_catalog.intelligence.market_ticks
				WHERE symbol = 'SPX' 
				  AND timestamp >= DATE_SUB(NOW(), INTERVAL 30 DAY)
			),
			
			-- 2. User Activity CTE: Aggregate logins per hour
			user_activity AS (
				SELECT
					user_id,
					DATE_TRUNC('hour', timestamp) as bucket_time,
					COUNT(*) as login_count
				FROM iceberg_catalog.intelligence.client_events
				WHERE event_type = 'login' 
				  AND timestamp >= DATE_SUB(NOW(), INTERVAL 30 DAY)
				GROUP BY user_id, DATE_TRUNC('hour', timestamp)
			)

		-- 3. Correlation Analysis
		SELECT 
			u.user_id,
			-- Calculate Pearson correlation between Login Count and Absolute Drawdown Severity
			CORR(u.login_count, ABS(m.drawdown_pct)) as anxiety_score,
			
			SUM(u.login_count) as total_logins_30d,
			MIN(m.drawdown_pct) as max_drawdown_exposure
		FROM user_activity u
		JOIN market_stats m ON u.bucket_time = m.bucket_time
		WHERE m.drawdown_pct < -0.01 -- Noise filter: Only analyze periods where market is down > 1%
		GROUP BY u.user_id
		HAVING 
			SUM(u.login_count) > 10 -- Minimum activity threshold
			AND CORR(u.login_count, ABS(m.drawdown_pct)) > 0.65 -- Threshold for "High Anxiety"
		ORDER BY anxiety_score DESC
	`

	// TODO(hasura-migration): StarRocks analytics query for Iceberg tables
	// Note: This complex correlation analysis may not be directly replaceable with Hasura.
	// Consider alternatives:
	// 1. Keep StarRocks for OLAP analytics (preferred for complex correlation/aggregations)
	// 2. Use Hasura Actions to call analytics service endpoint
	// 3. Pre-compute anxiety scores and store in Postgres, then query via Hasura
	// Example Hasura Action (if pre-computed):
	// query GetHighAnxietyUsers {
	//   client_profiles(
	//     where: {
	//       anxiety_score: {_gt: 0.65},
	//       total_logins_30d: {_gt: 10}
	//     },
	//     order_by: {anxiety_score: desc}
	//   ) {
	//     user_id
	//     anxiety_score
	//     total_logins_30d
	//     max_drawdown_exposure
	//   }
	// }
	rows, err := starrocksDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query starrocks: %w", err)
	}
	defer rows.Close()

	// 2. Prepare Postgres Transaction
	tx, err := pgDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare Statement for JSONB update
	// We use jsonb_set to safely insert the 'risk_profile' key without erasing other attributes.
	// COALESCE ensures we don't fail on null columns.
	// TODO(hasura-migration): Replace SQL UPDATE with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation TagHighAnxietyUser($userId: Int!, $riskProfile: String!) {
	//   update_users(
	//     where: {id: {_eq: $userId}},
	//     _set: {
	//       attributes: {_sql: "jsonb_set(COALESCE(attributes, '{}'::jsonb), '{risk_profile}', '\"high_anxiety\"', true)"},
	//       updated_at: "now()"
	//     }
	//   ) {
	//     affected_rows
	//     returning {
	//       id
	//       attributes
	//     }
	//   }
	// }
	// Note: For batch updates, consider using Hasura's bulk mutations or Actions
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE users 
		SET attributes = jsonb_set(
			COALESCE(attributes, '{}'::jsonb), 
			'{risk_profile}', 
			'"high_anxiety"', 
			true
		),
		updated_at = NOW()
		WHERE id = $1
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 3. Iterate and Execute
	// In a real-world scenario, we would use a buffered channel and a worker pool here
	// to limit the number of concurrent DB updates.
	count := 0
	for rows.Next() {
		var p ClientProfile
		// Scan only necessary fields. The query returns 4 columns: user_id, anxiety_score, total_logins, max_drawdown
		// We only need the first two for the struct, but we must scan all of them or discard them.
		var totalLogins uint64
		var maxDrawdown float64

		if err := rows.Scan(&p.UserID, &p.AnxietyScore, &totalLogins, &maxDrawdown); err != nil {
			return fmt.Errorf("row scan error: %w", err)
		}

		// Execute Update
		if _, err := stmt.ExecContext(ctx, p.UserID); err != nil {
			log.Printf("Error updating user %d: %v", p.UserID, err)
			continue // Don't abort the whole batch for one failure
		}
		count++
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// 4. Commit Transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit failed: %w", err)
	}

	log.Printf("Successfully tagged %d high-anxiety clients.", count)
	return nil
}
