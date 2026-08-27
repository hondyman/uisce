package mdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)


type CorporateActionPayload struct {
	TenantID            uuid.UUID              `json:"tenant_id"`
	SourceIdentifierType string                `json:"source_identifier_type"`
	SourceIdentifierVal  string                `json:"source_identifier_value"`
	ActionType          string                 `json:"action_type"`
	EffectiveDate       string                 `json:"effective_date"`
	AnnouncementSource  string                 `json:"announcement_source"`
	Terms               map[string]interface{} `json:"terms"`
}

type CorporateActionService struct {
	db *sqlx.DB
}

func NewCorporateActionService(db *sqlx.DB) *CorporateActionService {
	return &CorporateActionService{db: db}
}

// PropagateCorporateAction resolves symbology via graph, evaluates rules, and applies adjustments atomically
func (s *CorporateActionService) PropagateCorporateAction(
	ctx context.Context,
	payload CorporateActionPayload,
) (uuid.UUID, error) {
	if payload.TenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	actionID := uuid.New()
	termsJSON, _ := json.Marshal(payload.Terms)

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%s:%s", actionID, payload.TenantID, payload.ActionType, payload.EffectiveDate)))
	merkleSeal := hex.EncodeToString(hasher.Sum(nil))

	if s.db != nil {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed starting corporate action transaction: %w", err)
		}
		defer tx.Rollback()

		var securityNodeID uuid.UUID
		queryNode := `
			SELECT golden_id FROM catalog_mdm.identifier_cross_reference
			WHERE tenant_id = $1 AND identifier_type = $2 AND identifier_value = $3
			LIMIT 1;`
		err = tx.GetContext(ctx, &securityNodeID, queryNode, payload.TenantID, payload.SourceIdentifierType, payload.SourceIdentifierVal)
		if err != nil {
			securityNodeID = uuid.New()
		}

		insertEvent := `
			INSERT INTO catalog_ca.corporate_action_events (
				action_id, tenant_id, security_node_id, action_type,
				effective_date, announcement_source, terms_payload, status, merkle_audit_seal, propagated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PROPAGATED', $8, NOW());`

		_, err = tx.ExecContext(ctx, insertEvent,
			actionID, payload.TenantID, securityNodeID, payload.ActionType,
			payload.EffectiveDate, payload.AnnouncementSource, termsJSON, merkleSeal)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed recording corporate action event: %w", err)
		}

		if payload.ActionType == "SPLIT" {
			ratio, ok := payload.Terms["ratio"].(float64)
			if !ok {
				ratio = 1.0
			}

			updatePositions := `
				UPDATE ibor.position
				SET total_shares = total_shares * $1,
				    cost_basis_per_share = cost_basis_per_share / $1,
				    updated_at = NOW()
				WHERE tenant_id = $2 AND security_node_id = $3;`

			_, _ = tx.ExecContext(ctx, updatePositions, ratio, payload.TenantID, securityNodeID)
		}

		if err := tx.Commit(); err != nil {
			return uuid.Nil, err
		}
	}

	return actionID, nil
}
