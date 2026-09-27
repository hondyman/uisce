package orm

import (
	"context"
	"database/sql"
	"time"
)

// IsTradingOpen reports whether a market's session is currently open.
func IsTradingOpen(ctx context.Context, tx *sql.Tx, tenantID, mic string, at time.Time) (bool, error) {
	var open sql.NullTime
	var close sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT open_time, close_time
		FROM orm.trading_session
		WHERE tenant_id = $1
		  AND mic = $2
		  AND session_date = $3
		  AND session_type = 'CONTINUOUS'
	`, tenantID, mic, at.Format("2006-01-02")).Scan(&open, &close)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !open.Valid || !close.Valid {
		return false, nil
	}
	t := at.Format("15:04:05")
	return t >= open.Time.Format("15:04:05") && t <= close.Time.Format("15:04:05"), nil
}

// ActiveHalt returns the halt type if the security is currently halted.
func ActiveHalt(ctx context.Context, tx *sql.Tx, tenantID, securityID string) (sql.NullString, error) {
	var haltType sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT halt_type
		FROM orm.trading_halt
		WHERE tenant_id = $1
		  AND security_id = $2
		  AND is_resumed = false
		  AND (halt_end IS NULL OR halt_end > now())
		ORDER BY halt_start DESC
		LIMIT 1
	`, tenantID, securityID).Scan(&haltType)
	if err == sql.ErrNoRows {
		return haltType, nil
	}
	return haltType, err
}
