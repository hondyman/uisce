// Package mdmread is the thin read choke point for MDM exception-queue lookups
// used by MCP adapters (triage_mdm_exception).
package mdmread

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// ExceptionRow is one pending MDM exception queue entry.
type ExceptionRow struct {
	DomainKey       string `db:"domain_key"`
	MasterEntitySID string `db:"master_entity_sid"`
	FieldName       string `db:"field_name"`
	CompetingValues []byte `db:"competing_values"`
}

// GetByID loads one exception for the calling tenant.
//
// Predicate comparison (SL extract from MCP triage_mdm_exception):
//
//	OLD: WHERE exception_id = $1 AND tenant_id = $2
//	NEW: identical
//	DELTA: none
func (s *Service) GetByID(ctx context.Context, tenantID, exceptionID uuid.UUID) (*ExceptionRow, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	var item ExceptionRow
	err := s.db.GetContext(ctx, &item, `
		SELECT domain_key, master_entity_sid, field_name, competing_values
		FROM mdm.universal_exception_queue
		WHERE exception_id = $1 AND tenant_id = $2;
	`, exceptionID, tenantID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
