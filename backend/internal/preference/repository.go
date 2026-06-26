package preference

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository handles all database operations for source preferences and exceptions
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// --- Source Preferences ---

func (r *Repository) CreatePreference(ctx context.Context, p *SourcePreference) error {
	impactJSON, _ := p.ImpactAnalysis.JSON()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.source_preferences
			(id, tenant_id, business_object, semantic_term, region, priority, source_system,
			 confidence, status, version, core_id, override_reason, valid_from, valid_to,
			 impact_analysis, created_at, updated_at, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, p.ID, p.TenantID, p.BusinessObject, p.SemanticTerm, p.Region, p.Priority, p.SourceSystem,
		p.Confidence, p.Status, p.Version, p.CoreID, p.OverrideReason, p.ValidFrom, p.ValidTo,
		impactJSON, p.CreatedAt, p.UpdatedAt, p.CreatedBy)
	return err
}

func (r *Repository) GetPreference(ctx context.Context, id uuid.UUID) (*SourcePreference, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, business_object, semantic_term, region, priority, source_system,
		       confidence, status, version, core_id, override_reason, valid_from, valid_to,
		       impact_analysis, created_at, updated_at, created_by, updated_by
		FROM edm.source_preferences WHERE id = $1`, id)
	return scanPreference(row)
}

func (r *Repository) ListPreferences(ctx context.Context, tenantID uuid.UUID, bo, term, region string) ([]*SourcePreference, error) {
	query := `
		SELECT id, tenant_id, business_object, semantic_term, region, priority, source_system,
		       confidence, status, version, core_id, override_reason, valid_from, valid_to,
		       impact_analysis, created_at, updated_at, created_by, updated_by
		FROM edm.source_preferences WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	n := 2
	if bo != "" {
		query += fmt.Sprintf(" AND business_object = $%d", n)
		args = append(args, bo)
		n++
	}
	if term != "" {
		query += fmt.Sprintf(" AND semantic_term = $%d", n)
		args = append(args, term)
		n++
	}
	if region != "" {
		query += fmt.Sprintf(" AND region = $%d", n)
		args = append(args, region)
	}
	query += " ORDER BY priority ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prefs []*SourcePreference
	for rows.Next() {
		p, err := scanPreference(rows)
		if err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, nil
}

func (r *Repository) UpdatePreference(ctx context.Context, p *SourcePreference) error {
	impactJSON, _ := p.ImpactAnalysis.JSON()
	_, err := r.db.ExecContext(ctx, `
		UPDATE edm.source_preferences
		SET status=$1, version=$2, impact_analysis=$3, updated_at=$4, updated_by=$5
		WHERE id=$6`,
		p.Status, p.Version, impactJSON, time.Now(), p.UpdatedBy, p.ID)
	return err
}

func (r *Repository) AppendVersion(ctx context.Context, prefID uuid.UUID, status, reason string, impactJSON []byte, createdBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.preference_versions
			(id, preference_id, version, status, reason, impact_analysis, created_at, created_by)
		SELECT gen_random_uuid(), $1,
		       COALESCE((SELECT MAX(version) FROM edm.preference_versions WHERE preference_id=$1), 0) + 1,
		       $2, $3, $4, NOW(), $5
		`, prefID, status, reason, impactJSON, createdBy)
	return err
}

// --- Source Analytics ---

func (r *Repository) UpsertAnalytics(ctx context.Context, tenantID uuid.UUID, bo, term, region, source string, first, second, third, other int, avgConf float64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.source_analytics
			(id, tenant_id, business_object, semantic_term, region, source_system,
			 first_preference_count, second_preference_count, third_preference_count, other_preference_count,
			 total_selections, avg_confidence, generated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT DO NOTHING
	`, tenantID, bo, term, region, source, first, second, third, other, first+second+third+other, avgConf)
	return err
}

func (r *Repository) GetRankings(ctx context.Context, tenantID uuid.UUID, bo, term, region string) ([]SourceRanking, error) {
	query := `
		SELECT source_system,
		       SUM(first_preference_count),
		       SUM(second_preference_count),
		       SUM(third_preference_count),
		       SUM(other_preference_count),
		       AVG(avg_confidence)
		FROM edm.source_analytics
		WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	n := 2
	if bo != "" {
		query += fmt.Sprintf(" AND business_object = $%d", n)
		args = append(args, bo)
		n++
	}
	if term != "" {
		query += fmt.Sprintf(" AND semantic_term = $%d", n)
		args = append(args, term)
		n++
	}
	if region != "" {
		query += fmt.Sprintf(" AND region = $%d", n)
		args = append(args, region)
	}
	query += " GROUP BY source_system ORDER BY SUM(first_preference_count) DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rankings []SourceRanking
	for rows.Next() {
		var sr SourceRanking
		if err := rows.Scan(&sr.SourceSystem, &sr.FirstPreferenceCount, &sr.SecondPreferenceCount, &sr.ThirdPreferenceCount, &sr.OtherPreferenceCount, &sr.AvgConfidence); err != nil {
			return nil, err
		}
		sr.TotalSelections = sr.FirstPreferenceCount + sr.SecondPreferenceCount + sr.ThirdPreferenceCount + sr.OtherPreferenceCount
		if sr.TotalSelections > 0 {
			sr.FirstPreferencePercent = float64(sr.FirstPreferenceCount) / float64(sr.TotalSelections) * 100
		}
		rankings = append(rankings, sr)
	}
	return rankings, nil
}

// --- Source Exceptions ---

func (r *Repository) CreateException(ctx context.Context, e *SourceException) error {
	metaJSON, _ := json.Marshal(e.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.source_exceptions
			(id, tenant_id, business_object, semantic_term, region, source_system, exception_type,
			 description, impact_level, critical_path, status, metadata, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, e.ID, e.TenantID, e.BusinessObject, e.SemanticTerm, e.Region, e.SourceSystem, e.ExceptionType,
		e.Description, e.ImpactLevel, e.CriticalPath, e.Status, metaJSON, e.CreatedAt)
	return err
}

func (r *Repository) ListExceptions(ctx context.Context, tenantID uuid.UUID, status string) ([]*SourceException, error) {
	query := `SELECT id, tenant_id, business_object, semantic_term, region, source_system,
	                 exception_type, description, impact_level, critical_path, status, metadata, created_at
	          FROM edm.source_exceptions WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exceptions []*SourceException
	for rows.Next() {
		var e SourceException
		var metaJSON []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.BusinessObject, &e.SemanticTerm, &e.Region, &e.SourceSystem,
			&e.ExceptionType, &e.Description, &e.ImpactLevel, &e.CriticalPath, &e.Status, &metaJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaJSON, &e.Metadata)
		exceptions = append(exceptions, &e)
	}
	return exceptions, nil
}

func (r *Repository) ResolveException(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE edm.source_exceptions
		SET status = 'resolved', resolved_at = $1, resolved_by = $2
		WHERE id = $3`, now, resolvedBy, id)
	return err
}

func (r *Repository) AppendExceptionHistory(ctx context.Context, exceptionID uuid.UUID, status, description string, createdBy *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.exception_history (id, exception_id, status, description, created_at, created_by)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW(), $4)
	`, exceptionID, status, description, createdBy)
	return err
}

// --- Helpers ---

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPreference(row rowScanner) (*SourcePreference, error) {
	var p SourcePreference
	var impactJSON []byte
	err := row.Scan(
		&p.ID, &p.TenantID, &p.BusinessObject, &p.SemanticTerm, &p.Region, &p.Priority, &p.SourceSystem,
		&p.Confidence, &p.Status, &p.Version, &p.CoreID, &p.OverrideReason, &p.ValidFrom, &p.ValidTo,
		&impactJSON, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(impactJSON, &p.ImpactAnalysis)
	return &p, nil
}
