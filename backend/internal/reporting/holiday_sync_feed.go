package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type HolidayFeedRecord struct {
	Date           string `json:"date"`           // YYYY-MM-DD
	Name           string `json:"name"`           // e.g. "Martin Luther King Jr. Day"
	Type           string `json:"type"`           // FULL_CLOSE, EARLY_CLOSE
	EarlyCloseTime string `json:"earlyCloseTime"` // HH:MM:SS (optional)
}

type HolidaySyncDaemon struct {
	db         *sqlx.DB
	httpClient *http.Client
}

func NewHolidaySyncDaemon(db *sqlx.DB) *HolidaySyncDaemon {
	return &HolidaySyncDaemon{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SyncProviderHolidays pulls external holidays and performs idempotent batch upsert
func (d *HolidaySyncDaemon) SyncProviderHolidays(
	ctx context.Context,
	tenantID, calendarID uuid.UUID,
	providerName, feedURL string,
) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("holiday feed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("holiday provider returned HTTP %d", resp.StatusCode)
	}

	var records []HolidayFeedRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return 0, fmt.Errorf("failed decoding holiday feed payload: %w", err)
	}

	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	upsertSQL := `
		INSERT INTO public.tenant_calendar_holidays (
			calendar_id, holiday_date, holiday_name, holiday_type, early_close_time
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (calendar_id, holiday_date) DO UPDATE SET
			holiday_name = EXCLUDED.holiday_name,
			holiday_type = EXCLUDED.holiday_type,
			early_close_time = EXCLUDED.early_close_time;
	`

	syncedCount := 0
	for _, rec := range records {
		var earlyClose interface{} = nil
		if rec.EarlyCloseTime != "" {
			earlyClose = rec.EarlyCloseTime
		}
		hType := "FULL_CLOSE"
		if rec.Type != "" {
			hType = rec.Type
		}

		_, err := tx.ExecContext(ctx, upsertSQL, calendarID, rec.Date, rec.Name, hType, earlyClose)
		if err == nil {
			syncedCount++
		}
	}

	// Update Sync Log
	_, _ = tx.ExecContext(ctx, `
		UPDATE public.exchange_holiday_sync_configs
		SET last_synced_at = NOW(), last_sync_status = 'SUCCESS', last_sync_error = NULL
		WHERE tenant_id = $1 AND calendar_id = $2 AND source_provider = $3
	`, tenantID, calendarID, providerName)

	return syncedCount, tx.Commit()
}
