package pagebuilder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// WidgetPolicy is the server-governed (field_type, cardinality) -> widget
// mapping. It replaces the frontend FieldDataType->FieldWidget enum and the
// backend component_extensibility/forms/generator.go 3-type stub as the
// single source of truth. AllowedWidgetKeys is what a page's widget override
// is validated against.
type WidgetPolicy struct {
	FieldType         string         `db:"field_type"`
	Cardinality       string         `db:"cardinality"`
	DefaultWidgetKey  string         `db:"default_widget_key"`
	AllowedWidgetKeys pq.StringArray `db:"allowed_widget_keys"`
}

// Breakpoint is the set of non-desktop targets the responsive designer knows
// about. Desktop is always the base and never needs a fallback row.
type Breakpoint string

const (
	BreakpointMobile Breakpoint = "mobile"
	BreakpointTablet Breakpoint = "tablet"
)

type WidgetPolicyRepository struct {
	db *sqlx.DB
}

func NewWidgetPolicyRepository(db *sqlx.DB) *WidgetPolicyRepository {
	return &WidgetPolicyRepository{db: db}
}

func (r *WidgetPolicyRepository) GetPolicy(ctx context.Context, fieldType, cardinality string) (*WidgetPolicy, error) {
	var p WidgetPolicy
	err := r.db.GetContext(ctx, &p, `
		SELECT field_type, cardinality, default_widget_key, allowed_widget_keys
		FROM bo_widget_policy
		WHERE field_type = $1 AND cardinality = $2
	`, fieldType, cardinality)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get widget policy for %s/%s: %w", fieldType, cardinality, err)
	}
	return &p, nil
}

// ResolveForBreakpoint returns the widget key a page should actually render
// for widgetKey at bp. Three outcomes:
//   - (key, true, nil):  widget has no breakpoint-specific row -> renders as-is
//   - (fallbackKey, true, nil): a fallback row exists with a non-null substitute
//   - ("", false, nil):  widget is explicitly not_supported at bp — the page
//     must declare an alternative or fail validation; this is NOT an error,
//     it's a signal for the caller to reject the save.
func (r *WidgetPolicyRepository) ResolveForBreakpoint(ctx context.Context, widgetKey string, bp Breakpoint) (string, bool, error) {
	var fallback sql.NullString
	err := r.db.GetContext(ctx, &fallback, `
		SELECT fallback_widget_key FROM bo_widget_breakpoint_fallback
		WHERE widget_key = $1 AND breakpoint = $2
	`, widgetKey, string(bp))
	if errors.Is(err, sql.ErrNoRows) {
		return widgetKey, true, nil // no row: renders as-is at every breakpoint
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve breakpoint fallback for %s/%s: %w", widgetKey, bp, err)
	}
	if !fallback.Valid {
		return "", false, nil // explicit not_supported
	}
	return fallback.String, true, nil
}

// ValidatePageWidgets checks every widget key a page references against every
// declared breakpoint and returns the set that has no honest mobile/tablet
// equivalent. A non-empty result means the page must not be saved as-is —
// this is the build-time check that replaces silent runtime degradation.
func (r *WidgetPolicyRepository) ValidatePageWidgets(ctx context.Context, widgetKeys []string) (map[string][]Breakpoint, error) {
	unsupported := map[string][]Breakpoint{}
	for _, key := range widgetKeys {
		for _, bp := range []Breakpoint{BreakpointMobile, BreakpointTablet} {
			_, ok, err := r.ResolveForBreakpoint(ctx, key, bp)
			if err != nil {
				return nil, err
			}
			if !ok {
				unsupported[key] = append(unsupported[key], bp)
			}
		}
	}
	return unsupported, nil
}
