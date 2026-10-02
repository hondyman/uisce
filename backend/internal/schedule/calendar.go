package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Calendars gives a tenant's effective business calendar: the core calendar
// (the tenant's own with that code, else the gold copy's - decision D5) with
// the tenant's layer applied.
type Calendars interface {
	Days(ctx context.Context, tenantID, calendarCD string, from, to time.Time) (Days, error)
	List(ctx context.Context, tenantID string) ([]CalendarInfo, error)
}

// CalendarInfo describes a calendar a schedule can use.
type CalendarInfo struct {
	Code           string `db:"code" json:"code"`
	Name           string `db:"name" json:"name"`
	Owner          string `db:"owner" json:"owner"` // core | tenant
	FirstDate      string `db:"first_date" json:"first_date,omitempty"`
	LastDate       string `db:"last_date" json:"last_date,omitempty"`
	HasTenantLayer bool   `db:"has_tenant_layer" json:"has_tenant_layer"`
}

// MDMCalendars reads mdm.calendar_master / calendar_day / calendar_hierarchy
// (in crims). Reads only, filtered on tenant or gold copy explicitly.
type MDMCalendars struct {
	DB           *sqlx.DB
	GoldTenantID string
}

// core resolves the calendar a code means for a tenant.
func (c *MDMCalendars) core(ctx context.Context, tenantID, cd string) (string, error) {
	var id string
	err := c.DB.GetContext(ctx, &id, `SELECT id::text FROM mdm.calendar_master
		WHERE calendar_cd = $1 AND tenant_id::text IN ($2, $3) AND status = 'ACTIVE'
		ORDER BY (tenant_id::text = $2) DESC LIMIT 1`, cd, tenantID, c.GoldTenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", msgCalendarNotFound(cd)
	}
	return id, err
}

type dayRow struct {
	Date     time.Time      `db:"calendar_date"`
	Business bool           `db:"is_business_day"`
	Half     bool           `db:"is_half_day"`
	Holiday  sql.NullString `db:"holiday"`
}

const dayCols = `calendar_date, is_business_day, is_half_day,
	additional_holidays -> 0 ->> 'name' AS holiday`

func (c *MDMCalendars) Days(ctx context.Context, tenantID, cd string, from, to time.Time) (Days, error) {
	coreID, err := c.core(ctx, tenantID, cd)
	if err != nil {
		return nil, err
	}
	var core []dayRow
	if err := c.DB.SelectContext(ctx, &core, `SELECT `+dayCols+` FROM mdm.calendar_day
		WHERE calendar_id = $1::uuid AND calendar_date BETWEEN $2 AND $3`, coreID, from, to); err != nil {
		return nil, fmt.Errorf("reading calendar %s: %w", cd, err)
	}
	days := Days{}
	for _, r := range core {
		days[dateKey(r.Date)] = Day{Date: r.Date, Business: r.Business, HalfDay: r.Half, Holiday: r.Holiday.String}
	}
	// The tenant's layers over this calendar, lowest precedence first.
	var layers []struct {
		ChildID string `db:"child_id"`
		Mode    string `db:"inheritance_mode"`
	}
	if err := c.DB.SelectContext(ctx, &layers, `SELECT h.child_calendar_id::text AS child_id, h.inheritance_mode
		FROM mdm.calendar_hierarchy h JOIN mdm.calendar_master child ON child.id = h.child_calendar_id
		WHERE h.parent_calendar_id = $1::uuid AND child.tenant_id::text = $2 AND h.is_current
		  AND h.effective_from <= $4 AND (h.effective_to IS NULL OR h.effective_to >= $3)
		ORDER BY h.precedence DESC`, coreID, tenantID, from, to); err != nil {
		return nil, fmt.Errorf("reading calendar layers for %s: %w", cd, err)
	}
	for _, l := range layers {
		if l.Mode == "INHERIT_ALL" {
			continue
		}
		var rows []dayRow
		if err := c.DB.SelectContext(ctx, &rows, `SELECT `+dayCols+` FROM mdm.calendar_day
			WHERE calendar_id = $1::uuid AND calendar_date BETWEEN $2 AND $3`, l.ChildID, from, to); err != nil {
			return nil, fmt.Errorf("reading calendar layer for %s: %w", cd, err)
		}
		for _, r := range rows {
			k := dateKey(r.Date)
			base, ok := days[k]
			if !ok {
				continue // a layer cannot extend a calendar's coverage
			}
			switch l.Mode {
			case "ADDITIVE": // may only close more days
				if !r.Business {
					base.Business, base.Holiday = false, r.Holiday.String
				}
			default: // INHERIT_EXCEPT, OVERRIDE: the tenant's day wins (approved change)
				base.Business, base.HalfDay = r.Business, r.Half
				if !r.Business {
					base.Holiday = r.Holiday.String
				}
			}
			days[k] = base
		}
	}
	return days, nil
}

func (c *MDMCalendars) List(ctx context.Context, tenantID string) ([]CalendarInfo, error) {
	var out []CalendarInfo
	err := c.DB.SelectContext(ctx, &out, `
		SELECT m.calendar_cd AS code, m.name,
		       CASE WHEN m.tenant_id::text = $2 THEN 'core' ELSE 'tenant' END AS owner,
		       COALESCE((SELECT min(calendar_date)::text FROM mdm.calendar_day d WHERE d.calendar_id = m.id), '') AS first_date,
		       COALESCE((SELECT max(calendar_date)::text FROM mdm.calendar_day d WHERE d.calendar_id = m.id), '') AS last_date,
		       EXISTS (SELECT 1 FROM mdm.calendar_hierarchy h JOIN mdm.calendar_master child ON child.id = h.child_calendar_id
		               WHERE h.parent_calendar_id = m.id AND child.tenant_id::text = $1 AND h.is_current) AS has_tenant_layer
		FROM mdm.calendar_master m
		WHERE m.status = 'ACTIVE' AND m.tenant_id::text IN ($1, $2)
		  AND NOT EXISTS (SELECT 1 FROM mdm.calendar_hierarchy h WHERE h.child_calendar_id = m.id AND h.is_current)
		ORDER BY m.calendar_cd`, tenantID, c.GoldTenantID)
	return out, err
}
