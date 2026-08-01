package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/segmentio/kafka-go"
)

const overrideAuditTopic = "compliance-overrides"

func (h *ExternalComplianceHandler) recordCryptographicOverride(
	ctx context.Context,
	tenantID uuid.UUID,
	userID, tradeID, reason string,
	violations []rules.RuleViolation,
) error {
	record := OverrideRecord{
		TenantID:            tenantID,
		TradeID:            tradeID,
		OverrideByUser:     userID,
		JustificationReason: reason,
		ViolationsBypassed: violations,
		Timestamp:          time.Now().UTC(),
	}

	payloadJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal override record: %w", err)
	}

	if h.db != nil {
		if err := h.insertHashChain(ctx, tenantID, tradeID, userID, payloadJSON); err != nil {
			return fmt.Errorf("hash chain insert: %w", err)
		}
	}

	if h.kafkaWriter != nil {
		msg := kafka.Message{
			Key:   []byte(tenantID.String()),
			Value: payloadJSON,
			Headers: []kafka.Header{
				{Key: "entity_type", Value: []byte("PRE_TRADE_OVERRIDE")},
				{Key: "tenant_id", Value: []byte(tenantID.String())},
				{Key: "event_type", Value: []byte("compliance.override")},
			},
		}
		if err := h.kafkaWriter.WriteMessages(ctx, msg); err != nil {
			return fmt.Errorf("kafka publish: %w", err)
		}
	}

	return nil
}

func (h *ExternalComplianceHandler) insertHashChain(
	ctx context.Context,
	tenantID uuid.UUID,
	tradeID, performedBy string,
	payloadJSON []byte,
) error {
	var previousHash string
	err := h.db.QueryRowContext(ctx,
		`SELECT current_hash FROM public.cryptographic_audit_ledger
		 WHERE tenant_id = $1 AND entity_type = 'PRE_TRADE_OVERRIDE'
		 ORDER BY sequence_id DESC LIMIT 1`,
		tenantID.String(),
	).Scan(&previousHash)
	if err != nil {
		previousHash = "GENESIS_BLOCK"
	}

	dataToHash := fmt.Sprintf("%s|%s|%s|%s", previousHash, tenantID.String(), tradeID, string(payloadJSON))
	hash := sha256.Sum256([]byte(dataToHash))
	currentHash := hex.EncodeToString(hash[:])

	_, err = h.db.ExecContext(ctx,
		`INSERT INTO public.cryptographic_audit_ledger
		 (tenant_id, entity_type, entity_id, action_type, payload_snapshot, performed_by, previous_hash, current_hash)
		 VALUES ($1, 'PRE_TRADE_OVERRIDE', $2, 'COMPLIANCE_OVERRIDE', $3, $4, $5, $6)`,
		tenantID.String(), tradeID, payloadJSON, performedBy, previousHash, currentHash,
	)
	return err
}
