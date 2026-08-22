package drift

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PhysicalColumnMeta struct {
	ColumnName string `db:"column_name"`
	DataType   string `db:"data_type"`
}

type SchemaCrawler struct {
	db      *sqlx.DB
	matcher *DriftRemediationMatcher
}

func NewSchemaCrawler(db *sqlx.DB) *SchemaCrawler {
	return &SchemaCrawler{
		db:      db,
		matcher: NewDriftRemediationMatcher(db),
	}
}

// CrawlTableAndDetectDrift inspects physical table columns against catalog_node definitions
func (c *SchemaCrawler) CrawlTableAndDetectDrift(
	ctx context.Context,
	tenantID, backendID, tableNodeID uuid.UUID,
	schemaName, tableName string,
) ([]uuid.UUID, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var physicalCols []PhysicalColumnMeta
	infoQuery := `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_schema = $1 AND table_name = $2;
	`
	err := c.db.SelectContext(ctx, &physicalCols, infoQuery, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed fetching physical schema: %w", err)
	}

	physMap := make(map[string]string)
	for _, pc := range physicalCols {
		physMap[strings.ToLower(pc.ColumnName)] = pc.DataType
	}

	var catalogCols []struct {
		NodeID   uuid.UUID `db:"node_id"`
		NodeName string    `db:"node_name"`
		DataType string    `db:"data_type"`
	}
	catQuery := `
		SELECT node_id, node_name, COALESCE(properties->>'data_type', 'VARCHAR') AS data_type
		FROM public.catalog_node
		WHERE parent_node_id = $1 AND tenant_id = $2 AND node_type = 'COLUMN' AND is_active = TRUE;
	`
	err = c.db.SelectContext(ctx, &catalogCols, catQuery, tableNodeID, tenantID)
	if err != nil {
		return nil, err
	}

	detectedProposalIDs := make([]uuid.UUID, 0)

	for _, cc := range catalogCols {
		cleanName := strings.ToLower(cc.NodeName)
		_, exists := physMap[cleanName]

		if !exists {
			eventID := uuid.New()
			_, err = c.db.ExecContext(ctx, `
				INSERT INTO catalog_drift.schema_drift_events (
					event_id, tenant_id, backend_id, table_node_id,
					change_type, column_name, old_data_type
				) VALUES ($1, $2, $3, $4, 'COLUMN_DROPPED', $5, $6);
			`, eventID, tenantID, backendID, tableNodeID, cc.NodeName, cc.DataType)
			if err != nil {
				continue
			}

			candidates, matchErr := c.matcher.FindCandidateMatches(ctx, tenantID, tableNodeID, cc.NodeName)
			if matchErr == nil && len(candidates) > 0 {
				topCandidate := candidates[0]

				var affectedBindings []struct {
					BOID      uuid.UUID `db:"bo_id"`
					BindingID uuid.UUID `db:"binding_id"`
					FieldID   uuid.UUID `db:"field_id"`
					FieldName string    `db:"field_name"`
				}
				findBOQuery := `
					SELECT fb.bo_id, fb.binding_id, fb.field_id, bof.field_name
					FROM public.field_bindings fb
					JOIN public.business_object_fields bof ON bof.field_id = fb.field_id
					WHERE fb.source_node_id = $1 AND fb.tenant_id = $2 AND fb.is_active = TRUE;
				`
				_ = c.db.SelectContext(ctx, &affectedBindings, findBOQuery, cc.NodeID, tenantID)

				for _, ab := range affectedBindings {
					proposalID := uuid.New()
					_, insErr := c.db.ExecContext(ctx, `
						INSERT INTO catalog_drift.schema_drift_proposals (
							proposal_id, tenant_id, event_id, bo_id, binding_id, field_id, field_name,
							current_source_node_id, proposed_source_node_id, proposed_column_name,
							confidence_score, matching_strategy, affected_reports_count, status, remediation_rationale
						) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 1, 'PENDING', $13)
						ON CONFLICT (tenant_id, bo_id, binding_id, field_id, status) DO NOTHING;
					`, proposalID, tenantID, eventID, ab.BOID, ab.BindingID, ab.FieldID, ab.FieldName,
						cc.NodeID, topCandidate.ColumnNodeID, topCandidate.ColumnName,
						topCandidate.ConfidenceScore, topCandidate.Strategy, topCandidate.Rationale)

					if insErr == nil {
						detectedProposalIDs = append(detectedProposalIDs, proposalID)
					}
				}
			}
		}
	}

	return detectedProposalIDs, nil
}
