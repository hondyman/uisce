package tiles

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type CalendarConfig struct {
	SettlementConvention string `json:"settlement_convention"` // "T+0", "T+1", "T+2"
	AllowSameDay         bool   `json:"allow_same_day"`
	CalendarServiceURL   string `json:"calendar_service_url"` // optional; skip check if empty
}

func NewCalendarValidator(cfg CalendarConfig) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string

		for _, rec := range records {
			semantic, ok := rec["semantic"].(map[string]any)
			if !ok {
				errs = append(errs, "calendar: missing semantic map")
				continue
			}

			settlementDateStr, _ := semantic["settlement_date"].(string)
			if settlementDateStr == "" {
				out = append(out, rec)
				continue
			}

			t, err := time.Parse(time.RFC3339, settlementDateStr)
			if err != nil {
				errs = append(errs, fmt.Sprintf("calendar: invalid settlement_date format: %v", err))
				continue
			}

			if cfg.CalendarServiceURL == "" {
				if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
					errs = append(errs, fmt.Sprintf("calendar: settlement_date %s falls on a weekend", settlementDateStr))
					continue
				}
			} else {
				req, err := http.NewRequestWithContext(ctx, "GET", cfg.CalendarServiceURL+"?date="+settlementDateStr, nil)
				if err != nil {
					errs = append(errs, fmt.Sprintf("calendar: failed to create request: %v", err))
					continue
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					errs = append(errs, fmt.Sprintf("calendar: service call failed: %v", err))
					continue
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					errs = append(errs, fmt.Sprintf("calendar: date %s rejected by calendar service", settlementDateStr))
					continue
				}
			}

			out = append(out, rec)
		}
		return out, errs, nil
	}
}
