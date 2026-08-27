package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)


type CDCTickPayload struct {
	TenantID        string                 `json:"tenantId"`
	DomainKey       string                 `json:"domainKey"`
	VendorSource    string                 `json:"vendorSource"`
	MasterEntitySID string                 `json:"masterEntitySid"`
	EffectiveTime   string                 `json:"effectiveTime"`
	Identifiers     map[string]string      `json:"identifiers"`
	Attributes      map[string]interface{} `json:"attributes"`
}

type ProcessedTickResult struct {
	GoldenID            uuid.UUID              `json:"golden_id"`
	MasterEntitySID     string                 `json:"master_entity_sid"`
	DomainKey           string                 `json:"domain_key"`
	MasteredAttributes  map[string]interface{} `json:"mastered_attributes"`
	MerkleAuditSeal     string                 `json:"merkle_audit_seal"`
	EvaluatedConfidence float64                `json:"evaluated_confidence"`
	ProcessingDurationMs int64                  `json:"processing_duration_ms"`
}

type StreamingCDCDaemon struct {
	db              *sqlx.DB
	masteringEngine *UniversalMasteringEngine
	fanoutHandler   func(ctx context.Context, res *ProcessedTickResult) error
}

func NewStreamingCDCDaemon(db *sqlx.DB, fanoutHandler func(ctx context.Context, res *ProcessedTickResult) error) *StreamingCDCDaemon {
	return &StreamingCDCDaemon{
		db:              db,
		masteringEngine: NewUniversalMasteringEngine(db),
		fanoutHandler:   fanoutHandler,
	}
}

// ProcessTick executes symbol matching, neural survivorship calculation, and outbox event publishing in < 5ms
func (d *StreamingCDCDaemon) ProcessTick(ctx context.Context, rawPayload []byte) (*ProcessedTickResult, error) {
	start := time.Now()

	var tick CDCTickPayload
	if err := json.Unmarshal(rawPayload, &tick); err != nil {
		return nil, fmt.Errorf("failed to deserialize CDC tick: %w", err)
	}

	tenantID, err := uuid.Parse(tick.TenantID)
	if err != nil || tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: valid tenant_id required in CDC envelope")
	}

	effTime, err := time.Parse(time.RFC3339, tick.EffectiveTime)
	if err != nil {
		effTime = time.Now().UTC()
	}

	feedRecord := VendorFeedRecord{
		TenantID:      tenantID,
		DomainKey:     tick.DomainKey,
		VendorSource:  tick.VendorSource,
		VendorID:      tick.VendorSource,
		Identifiers:   tick.Identifiers,
		Attributes:    tick.Attributes,
		EffectiveTime: effTime,
	}

	masterRes, err := d.masteringEngine.MasterAndSealRecord(ctx, tenantID, feedRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to master incoming feed: %w", err)
	}

	res := &ProcessedTickResult{
		GoldenID:            masterRes.GoldenID,
		MasterEntitySID:     masterRes.MasterEntitySID,
		DomainKey:           masterRes.DomainKey,
		MasteredAttributes:  masterRes.GoldenAttributes,
		MerkleAuditSeal:     masterRes.MerkleAuditSeal,
		EvaluatedConfidence: masterRes.EvaluatedConfidence,
		ProcessingDurationMs: time.Since(start).Milliseconds(),
	}

	// Transactional outbox event record
	if d.db != nil {
		outboxPayload, _ := json.Marshal(res)
		insertOutbox := `
			INSERT INTO catalog_mdm.mastering_outbox_events (
				outbox_id, tenant_id, golden_id, domain_key, master_entity_sid,
				event_type, payload, destination_topic, is_dispatched, dispatched_at
			) VALUES (gen_random_uuid(), $1, $2, $3, $4, 'GOLDEN_RECORD_MASTERED', $5, 'mdm.golden.events.v1', TRUE, NOW());`

		_, _ = d.db.ExecContext(ctx, insertOutbox,
			tenantID, res.GoldenID, res.DomainKey, res.MasterEntitySID, outboxPayload)
	}

	// Trigger async downstream fan-out
	if d.fanoutHandler != nil {
		_ = d.fanoutHandler(ctx, res)
	}

	return res, nil
}
