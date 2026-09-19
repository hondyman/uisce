// Package driftread is the read-only choke point for pending schema-drift
// proposals (catalog_drift.schema_drift_proposals).
//
// Read-only by design: SELECT only. Do not delegate to
// metadata.BusinessObjectService.DetectSchemaDrift — that method returns
// synthetic sentinel proposals and is not a reader of this table.
// Writes/repairs belong on ApplyDriftRepairPatch (separate path).
package driftread

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// PendingProposal is one PENDING drift proposal joined to its BO name.
type PendingProposal struct {
	ProposalID     uuid.UUID `db:"proposal_id"`
	BOName         string    `db:"bo_name"`
	FieldName      string    `db:"field_name"`
	ProposedColumn string    `db:"proposed_column_name"`
	Confidence     float64   `db:"confidence_score"`
}

// ListPending returns PENDING proposals for the tenant, highest confidence first.
//
// Predicate comparison (SL extract from MCP inspect_schema_drift):
//
//	OLD: WHERE p.tenant_id = $1 AND p.status = 'PENDING' (+ JOIN business_objects)
//	NEW: identical
//	DELTA: none
//
// Optional boId from the MCP tool schema remains unused (same as Tier 0).
func (s *Service) ListPending(ctx context.Context, tenantID uuid.UUID) ([]PendingProposal, error) {
	if s == nil || s.db == nil {
		return []PendingProposal{}, nil
	}
	var proposals []PendingProposal
	err := s.db.SelectContext(ctx, &proposals, `
		SELECT p.proposal_id, bo.bo_name, p.field_name, p.proposed_column_name, p.confidence_score
		FROM catalog_drift.schema_drift_proposals p
		JOIN public.business_objects bo ON bo.id = p.bo_id
		WHERE p.tenant_id = $1 AND p.status = 'PENDING'
		ORDER BY p.confidence_score DESC;
	`, tenantID)
	if err != nil {
		return nil, err
	}
	if proposals == nil {
		proposals = []PendingProposal{}
	}
	return proposals, nil
}
