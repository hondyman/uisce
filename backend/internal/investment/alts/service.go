package alts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AlternativeAsset represents a non-standard investment asset
type AlternativeAsset struct {
	ID                 uuid.UUID       `db:"asset_id" json:"id"`
	TenantID           uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Slug               string          `db:"slug" json:"slug"`
	Name               string          `db:"name" json:"name"`
	AssetType          string          `db:"asset_type" json:"asset_type"`
	CommonAttributes   json.RawMessage `db:"common_attributes" json:"common_attributes"`
	SpecificAttributes json.RawMessage `db:"specific_attributes" json:"specific_attributes"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at" json:"updated_at"`
}

// ValuationEvent represents a point-in-time value or cash flow
type ValuationEvent struct {
	ID        uuid.UUID       `db:"event_id" json:"id"`
	AssetID   uuid.UUID       `db:"asset_id" json:"asset_id"`
	EventDate time.Time       `db:"event_date" json:"event_date"`
	EventType string          `db:"event_type" json:"event_type"`
	Amount    float64         `db:"amount" json:"amount"`
	Currency  string          `db:"currency" json:"currency"`
	Source    string          `db:"source" json:"source"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// DailyNAV represents the calculated value for a specific date
type DailyNAV struct {
	AssetID    uuid.UUID `db:"asset_id" json:"asset_id"`
	ReportDate time.Time `db:"report_date" json:"report_date"`
	NAV        float64   `db:"nav" json:"nav"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: sqlx.NewDb(db, "postgres")}
}

// CreateAsset creates a new alternative asset definition
func (s *Service) CreateAsset(ctx context.Context, asset *AlternativeAsset) error {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	query := `
		INSERT INTO alternative_assets (asset_id, tenant_id, slug, name, asset_type, common_attributes, specific_attributes)
		VALUES (:asset_id, :tenant_id, :slug, :name, :asset_type, :common_attributes, :specific_attributes)
	`
	_, err := s.db.NamedExecContext(ctx, query, asset)
	return err
}

// GetAsset retrieves an asset by ID
func (s *Service) GetAsset(ctx context.Context, assetID uuid.UUID) (*AlternativeAsset, error) {
	var asset AlternativeAsset
	query := `SELECT * FROM alternative_assets WHERE asset_id = $1`
	err := s.db.GetContext(ctx, &asset, query, assetID)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// ListAssets retrieves assets for a tenant, optionally filtering by type
func (s *Service) ListAssets(ctx context.Context, tenantID uuid.UUID, assetType string) ([]AlternativeAsset, error) {
	var assets []AlternativeAsset
	query := `SELECT * FROM alternative_assets WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if assetType != "" {
		query += ` AND asset_type = $2`
		args = append(args, assetType)
	}

	err := s.db.SelectContext(ctx, &assets, query, args...)
	return assets, err
}

// RecordValuation adds a new valuation event
func (s *Service) RecordValuation(ctx context.Context, event *ValuationEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	query := `
		INSERT INTO valuation_events (event_id, asset_id, event_date, event_type, amount, currency, source, metadata)
		VALUES (:event_id, :asset_id, :event_date, :event_type, :amount, :currency, :source, :metadata)
	`
	_, err := s.db.NamedExecContext(ctx, query, event)
	return err
}

// GetValuationHistory retrieves raw events for an asset
func (s *Service) GetValuationHistory(ctx context.Context, assetID uuid.UUID) ([]ValuationEvent, error) {
	var events []ValuationEvent
	query := `SELECT * FROM valuation_events WHERE asset_id = $1 ORDER BY event_date DESC`
	err := s.db.SelectContext(ctx, &events, query, assetID)
	return events, err
}

// GetDailyNAV calculates the NAV for a range of dates using LOCF logic
// Note: This uses the view defined in the migration for simplicity.
func (s *Service) GetDailyNAV(ctx context.Context, assetID uuid.UUID, startDate, endDate time.Time) ([]DailyNAV, error) {
	var navs []DailyNAV
	query := `
		SELECT asset_id, report_date, nav
		FROM view_alternative_assets_daily_nav
		WHERE asset_id = $1 AND report_date >= $2 AND report_date <= $3
		ORDER BY report_date ASC
	`
	err := s.db.SelectContext(ctx, &navs, query, assetID, startDate, endDate)
	return navs, err
}

// SearchByAttribute finds assets where specific_attributes matches a JSON query
// Example: SearchByAttribute(ctx, tenantID, "vintage_year", 2020)
func (s *Service) SearchByAttribute(ctx context.Context, tenantID uuid.UUID, key string, value interface{}) ([]AlternativeAsset, error) {
	var assets []AlternativeAsset
	// Construct JSONB containment query
	// specific_attributes @> '{"key": value}'
	
	jsonQuery := map[string]interface{}{key: value}
	jsonBytes, err := json.Marshal(jsonQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	query := `SELECT * FROM alternative_assets WHERE tenant_id = $1 AND specific_attributes @> $2`
	err = s.db.SelectContext(ctx, &assets, query, tenantID, jsonBytes)
	return assets, err
}
