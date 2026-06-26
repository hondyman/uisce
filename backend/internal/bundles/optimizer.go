package bundles

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// EnsureOptimizationSchema creates tables required for optimization proposals.
func EnsureOptimizationSchema(db *sqlx.DB) error {
	if db == nil {
		return nil
	}
	// Create bundle catalog and proposal tables (simple PoC schema)
	_, _ = db.Exec(`
        CREATE TABLE IF NOT EXISTS claim_bundle (
            id UUID PRIMARY KEY,
            name TEXT,
            version INT,
            domain TEXT,
            description TEXT,
            created_by TEXT,
            created_at TIMESTAMP WITH TIME ZONE,
            status TEXT,
            risk_level TEXT
        )
    `)
	_, _ = db.Exec(`
        CREATE TABLE IF NOT EXISTS claim_bundle_item (
            id UUID PRIMARY KEY,
            bundle_id UUID,
            model_id UUID,
            permission TEXT,
            scope JSONB
        )
    `)
	_, _ = db.Exec(`
        CREATE TABLE IF NOT EXISTS bundle_usage_stat (
            id UUID PRIMARY KEY,
            bundle_id UUID,
            window TEXT,
            assigned_users INT,
            active_users INT,
            utilization DOUBLE PRECISION,
            last_calculated TIMESTAMP WITH TIME ZONE
        )
    `)
	_, _ = db.Exec(`
        CREATE TABLE IF NOT EXISTS bundle_change_proposal (
            id UUID PRIMARY KEY,
            bundle_id UUID,
            proposed_version INT,
            change_type TEXT,
            details JSONB,
            fitness_score DOUBLE PRECISION,
            risk_score DOUBLE PRECISION,
            impact JSONB,
            status TEXT,
            created_at TIMESTAMP WITH TIME ZONE,
            decided_at TIMESTAMP WITH TIME ZONE,
            decided_by TEXT
        )
    `)
	// guardrail rules table for configurable SoD / certified rules
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS guardrail_rules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type TEXT,
			data JSONB
		)
	`)
	return nil
}

// AnalyzeAndPropose runs a simple analysis over candidate_bundles and creates proposals
// PoC logic: if a persisted candidate bundle does not map to claim_bundle, create a 'propose_add' proposal.
func AnalyzeAndPropose(db *sqlx.DB, tenantID string) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	// Ensure schema
	_ = EnsureOptimizationSchema(db)

	// Load persisted candidate_bundles for tenant
	candidates, err := ListPersistedCandidates(db, tenantID, 1000)
	if err != nil {
		return nil, err
	}
	created := []string{}
	for _, c := range candidates {
		// Check if there is an existing claim_bundle with same name
		var existingId sql.NullString
		err := db.Get(&existingId, `SELECT id FROM claim_bundle WHERE name=$1 LIMIT 1`, c.Name)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if existingId.Valid {
			// skip if exists
			continue
		}

		// create proposal to add this bundle to catalog
		details := map[string]interface{}{
			"claims":      c.Claims,
			"description": c.Description,
		}
		detailsB, _ := json.Marshal(details)
		id := uuid.New().String()
		now := time.Now()
		// simple fitness/risk heuristics for PoC
		fitness := c.Score
		risk := c.Risk
		_, err = db.Exec(`INSERT INTO bundle_change_proposal (id, bundle_id, proposed_version, change_type, details, fitness_score, risk_score, impact, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			id, nil, 1, "add", detailsB, fitness, risk, json.RawMessage(`{}`), "pending", now)
		if err != nil {
			return nil, fmt.Errorf("failed to insert proposal: %w", err)
		}
		created = append(created, id)
	}
	return created, nil
}

// ListProposals lists proposals with optional status filter
func ListProposals(db *sqlx.DB, status string, limit int) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Queryx(`SELECT id, bundle_id, proposed_version, change_type, details, fitness_score, risk_score, impact, status, created_at FROM bundle_change_proposal `+
		` `+"WHERE status=$1 LIMIT $2", status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		var bundleID sql.NullString
		var proposedVersion sql.NullInt64
		var changeType sql.NullString
		var detailsB []byte
		var fitness sql.NullFloat64
		var risk sql.NullFloat64
		var impactB []byte
		var stat sql.NullString
		var created time.Time
		if err := rows.Scan(&id, &bundleID, &proposedVersion, &changeType, &detailsB, &fitness, &risk, &impactB, &stat, &created); err != nil {
			return nil, err
		}
		entry := map[string]interface{}{
			"id":               id,
			"bundle_id":        nil,
			"proposed_version": proposedVersion.Int64,
			"change_type":      changeType.String,
			"details":          json.RawMessage(detailsB),
			"fitness_score":    fitness.Float64,
			"risk_score":       risk.Float64,
			"impact":           json.RawMessage(impactB),
			"status":           stat.String,
			"created_at":       created,
		}
		if bundleID.Valid {
			entry["bundle_id"] = bundleID.String
		}
		out = append(out, entry)
	}
	return out, nil
}
