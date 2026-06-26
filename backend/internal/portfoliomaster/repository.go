package portfoliomaster

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repository is the data-access layer for the portfolio master gold copy.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── Source Registry ──────────────────────────────────────────────────────────

// ListRegistrySources returns all active source registry entries for a tenant.
func (r *Repository) ListRegistrySources(ctx context.Context, tenantID uuid.UUID) ([]*SourceRegistry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, source_name, source_code, source_type,
		       COALESCE(endpoint_url,''), is_active, priority_score, confidence_base,
		       account_types, asset_classes, regions,
		       tenant_id, core_id, created_at, updated_at
		FROM edm.source_registry
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY priority_score DESC, source_name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SourceRegistry
	for rows.Next() {
		s, err := scanRegistry(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

// GetRegistrySourceByCode returns a single source registry entry by its short code.
func (r *Repository) GetRegistrySourceByCode(ctx context.Context, tenantID uuid.UUID, code string) (*SourceRegistry, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, source_name, source_code, source_type,
		       COALESCE(endpoint_url,''), is_active, priority_score, confidence_base,
		       account_types, asset_classes, regions,
		       tenant_id, core_id, created_at, updated_at
		FROM edm.source_registry
		WHERE tenant_id = $1 AND source_code = $2`, tenantID, code)
	return scanRegistry(row)
}

// ─── Portfolio Golden ─────────────────────────────────────────────────────────

// ListGoldenRecords returns current (valid_to IS NULL) golden records for a tenant.
// Optionally filtered by account_type and scoped to a point-in-time asOf date.
func (r *Repository) ListGoldenRecords(ctx context.Context, tenantID uuid.UUID, accountType string, asOf time.Time) ([]*PortfolioGolden, error) {
	query := `
		SELECT id, tenant_id, portfolio_id, account_type, security_id, security_name,
		       quantity, price, market_value, currency, asset_class, country, region,
		       confidence_score, source_systems, contributing_sources,
		       created_at, updated_at, created_by, updated_by, valid_from, valid_to
		FROM edm.portfolio_golden
		WHERE tenant_id = $1
		  AND valid_from <= $2
		  AND (valid_to IS NULL OR valid_to > $2)`
	args := []interface{}{tenantID, asOf}
	n := 3
	if accountType != "" {
		query += fmt.Sprintf(" AND account_type = $%d", n)
		args = append(args, accountType)
		n++ //nolint:ineffassign
	}
	query += " ORDER BY portfolio_id, security_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*PortfolioGolden
	for rows.Next() {
		g, err := scanGolden(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, g)
	}
	return records, nil
}

// UpsertGoldenRecord inserts or updates a golden record (by unique portfolio+security+valid_from).
func (r *Repository) UpsertGoldenRecord(ctx context.Context, g *PortfolioGolden) error {
	ssJSON, _ := json.Marshal(g.SourceSystems)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.portfolio_golden
			(id, tenant_id, portfolio_id, account_type, security_id, security_name,
			 quantity, price, market_value, currency, asset_class, country, region,
			 confidence_score, source_systems, contributing_sources,
			 created_at, updated_at, created_by, valid_from, valid_to)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		ON CONFLICT (tenant_id, portfolio_id, security_id, valid_from)
		DO UPDATE SET
			security_name       = EXCLUDED.security_name,
			quantity            = EXCLUDED.quantity,
			price               = EXCLUDED.price,
			market_value        = EXCLUDED.market_value,
			currency            = EXCLUDED.currency,
			asset_class         = EXCLUDED.asset_class,
			country             = EXCLUDED.country,
			region              = EXCLUDED.region,
			confidence_score    = EXCLUDED.confidence_score,
			source_systems      = EXCLUDED.source_systems,
			contributing_sources= EXCLUDED.contributing_sources,
			updated_at          = NOW()`,
		g.ID, g.TenantID, g.PortfolioID, g.AccountType, g.SecurityID, g.SecurityName,
		g.Quantity, g.Price, g.MarketValue, g.Currency, g.AssetClass, g.Country, g.Region,
		g.ConfidenceScore, ssJSON, pq.Array(g.ContributingSources),
		g.CreatedAt, g.UpdatedAt, g.CreatedBy, g.ValidFrom, g.ValidTo)
	return err
}

// ─── Source Preferences (portfolio scope) ─────────────────────────────────────

// ListPortfolioPreferences returns source preferences scoped to the Portfolio BO,
// optionally filtered by semantic_term and account_type.
func (r *Repository) ListPortfolioPreferences(ctx context.Context, tenantID uuid.UUID, semanticTerm, accountType string) ([]portPref, error) {
	query := `
		SELECT id, semantic_term, account_type, priority, source_system, confidence, status
		FROM edm.source_preferences
		WHERE tenant_id = $1 AND business_object = 'Portfolio'
		  AND status = 'production'
		  AND valid_from <= NOW() AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}
	n := 2
	if semanticTerm != "" {
		query += fmt.Sprintf(" AND semantic_term = $%d", n)
		args = append(args, semanticTerm)
		n++
	}
	if accountType != "" {
		query += fmt.Sprintf(" AND account_type = $%d", n)
		args = append(args, accountType)
		n++ //nolint:ineffassign
	}
	query += " ORDER BY priority ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []portPref
	for rows.Next() {
		var p portPref
		if err := rows.Scan(&p.ID, &p.SemanticTerm, &p.AccountType, &p.Priority, &p.SourceSystem, &p.Confidence, &p.Status); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, nil
}

// portPref is a lightweight projection used internally for preference resolution
type portPref struct {
	ID           uuid.UUID
	SemanticTerm string
	AccountType  string
	Priority     int
	SourceSystem string
	Confidence   int
	Status       string
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanRegistry(row rowScanner) (*SourceRegistry, error) {
	var s SourceRegistry
	var coreID sql.NullString
	err := row.Scan(
		&s.ID, &s.SourceName, &s.SourceCode, &s.SourceType,
		&s.EndpointURL, &s.IsActive, &s.PriorityScore, &s.ConfidenceBase,
		pq.Array(&s.AccountTypes), pq.Array(&s.AssetClasses), pq.Array(&s.Regions),
		&s.TenantID, &coreID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if coreID.Valid {
		id, _ := uuid.Parse(coreID.String)
		s.CoreID = &id
	}
	return &s, nil
}

func scanGolden(row rowScanner) (*PortfolioGolden, error) {
	var g PortfolioGolden
	var ssJSON []byte
	var updatedBy sql.NullString
	var validTo sql.NullTime
	var cs []string // contributing_sources scanned as text[]

	err := row.Scan(
		&g.ID, &g.TenantID, &g.PortfolioID, &g.AccountType, &g.SecurityID, &g.SecurityName,
		&g.Quantity, &g.Price, &g.MarketValue, &g.Currency, &g.AssetClass, &g.Country, &g.Region,
		&g.ConfidenceScore, &ssJSON, pq.Array(&cs),
		&g.CreatedAt, &g.UpdatedAt, &g.CreatedBy, &updatedBy, &g.ValidFrom, &validTo,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(ssJSON, &g.SourceSystems)
	if g.SourceSystems == nil {
		g.SourceSystems = make(map[string]string)
	}
	if validTo.Valid {
		g.ValidTo = &validTo.Time
	}
	if updatedBy.Valid {
		id, _ := uuid.Parse(updatedBy.String)
		g.UpdatedBy = &id
	}
	for _, s := range cs {
		s = strings.TrimSpace(s)
		if id, err := uuid.Parse(s); err == nil {
			g.ContributingSources = append(g.ContributingSources, id)
		}
	}
	return &g, nil
}
