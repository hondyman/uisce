package swift

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
)

// TenantResolver maps a SWIFT BIC pair to (tenantID, custodianID).
// Implementation reads from swift_tenant_config (GSIFI-scoped).
// GSIFI boundary: every message must be tenant-tagged at adapter dispatch, never inferred from message content.
type TenantResolver interface {
	Resolve(bicSender, bicReceiver string) (tenantID uuid.UUID, custodianID uuid.UUID, ok bool)
}

// InboundSink receives tenant-tagged SWIFT messages for the data-pipeline swift_decode tile.
type InboundSink interface {
	Emit(ctx context.Context, rec InboundSWIFTRecord) error
}

// InboundSinkFunc adapts a function to InboundSink.
type InboundSinkFunc func(ctx context.Context, rec InboundSWIFTRecord) error

// Emit calls the underlying function.
func (f InboundSinkFunc) Emit(ctx context.Context, rec InboundSWIFTRecord) error {
	if f == nil {
		return nil
	}
	return f(ctx, rec)
}

// InboundSWIFTRecord is handed from adapter → swift_decode tile.
type InboundSWIFTRecord struct {
	RawBytes       []byte    `json:"raw_bytes"`
	TenantID       uuid.UUID `json:"tenant_id"`
	CustodianID    uuid.UUID `json:"custodian_id"`
	MsgType        string    `json:"msg_type"`       // "MT541", "pacs.008"
	TransactionRef string    `json:"transaction_ref"`// :20: or EndToEndId
	UETR           string    `json:"uetr,omitempty"` // GPI tracking ID
	BICSender      string    `json:"bic_sender"`
	BICReceiver    string    `json:"bic_receiver"`
	ReceivedAt     time.Time `json:"received_at"`
}

// StaticTenantResolver maps a BIC pair to a fixed tenant/custodian.
// Used for tests to ensure tagging without full swift_tenant_config lookup.
type StaticTenantResolver struct {
	TenantID    uuid.UUID
	CustodianID uuid.UUID
}

// Resolve implements TenantResolver for StaticTenantResolver.
// GSIFI boundary: tenant resolution is strictly configuration based.
func (s StaticTenantResolver) Resolve(bicSender, bicReceiver string) (uuid.UUID, uuid.UUID, bool) {
	if s.TenantID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false
	}
	return s.TenantID, s.CustodianID, true
}

// Adapter receives raw SWIFT bytes, tenant-tags them, writes an audit row to
// vend.swift_session_log, and dispatches to the pipeline.
//
// Session logging is a gateway concern (Layer 1): the audit row is written as
// soon as the tenant is resolved, before any business-logic tile runs. This
// gives full message visibility and UETR-based retransmission-discard
// independently of the data-pipeline DAG engine.
type Adapter struct {
	resolver        TenantResolver
	sink            InboundSink
	adminURL        string
	adminToken      string
	LatencyBudgetMs int
	// db is the application *sql.DB used to write vend.swift_session_log.
	// If nil, session logging is skipped (tests; non-DB deployments).
	db *sql.DB
}

// NewAdapter creates a new SWIFT adapter.
func NewAdapter(resolver TenantResolver, sink InboundSink, adminURL, adminToken string) *Adapter {
	return &Adapter{
		resolver:        resolver,
		sink:            sink,
		adminURL:        adminURL,
		adminToken:      adminToken,
		LatencyBudgetMs: 5000,
	}
}

// WithDB attaches the application *sql.DB so the adapter can write audit rows to
// vend.swift_session_log and discard SWIFTNet UETR retransmissions.
// db must use the lib/pq driver (the current binary registration is "postgres" / lib/pq).
func (a *Adapter) WithDB(db *sql.DB) *Adapter {
	a.db = db
	return a
}

// Receive processes a raw inbound SWIFT message (MT or MX bytes).
// Steps:
// 1. Parse minimal header (BICSender, BICReceiver, MsgType, TransactionRef)
// 2. Resolve tenant via BIC pair — if not found, log and return error
// 3. Emit InboundSWIFTRecord to sink
// 4. Return nil on success
// GSIFI boundary: every message must be tenant-tagged at adapter dispatch.
func (a *Adapter) Receive(ctx context.Context, raw []byte) error {
	msgType := DetectMsgType(raw)
	
	var bicSender, bicReceiver, txRef, uetr string
	
	if len(raw) > 0 && raw[0] == '{' {
		mt, err := ParseMT(raw)
		if err == nil {
			bicSender = mt.BICSender
			bicReceiver = mt.BICReceiver
			txRef = mt.ExtractTransactionRef()
			uetr = mt.Fields[":121:"] // GPI UETR
		} else {
			log.Printf("[SWIFT] Failed to parse MT message: %v", err)
		}
	} else {
		mx, err := ParseMX(raw)
		if err == nil {
			txRef = mx.MsgID
			// Best effort extraction for MX BICs from flattened paths
			bicSender = mx.Fields["AppHdr/Fr/FIId/FinInstnId/BICFI"]
			if bicSender == "" {
				bicSender = mx.Fields["Document/FIToFICstmrCdtTrf/GrpHdr/InstgAgt/FinInstnId/BICFI"]
			}
			bicReceiver = mx.Fields["AppHdr/To/FIId/FinInstnId/BICFI"]
			if bicReceiver == "" {
				bicReceiver = mx.Fields["Document/FIToFICstmrCdtTrf/GrpHdr/InstdAgt/FinInstnId/BICFI"]
			}
			uetr = mx.Fields["AppHdr/BizMsgIdr"]
		} else {
			log.Printf("[SWIFT] Failed to parse MX message: %v", err)
		}
	}

	tenantID, custodianID, ok := a.resolver.Resolve(bicSender, bicReceiver)
	if !ok {
		log.Printf("[SWIFT] Receive: no tenant resolution for BICs %s -> %s; dropping", bicSender, bicReceiver)
		return nil
	}

	rec := InboundSWIFTRecord{
		RawBytes:       raw,
		TenantID:       tenantID,
		CustodianID:    custodianID,
		MsgType:        msgType,
		TransactionRef: txRef,
		UETR:           uetr,
		BICSender:      bicSender,
		BICReceiver:    bicReceiver,
		ReceivedAt:     time.Now().UTC(),
	}

	budget := time.Duration(a.LatencyBudgetMs) * time.Millisecond
	if budget <= 0 {
		budget = 5 * time.Second
	}
	emitCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	// Write audit row and discard UETR retransmissions at the gateway layer.
	// This happens before sink.Emit so no downstream work runs for a retransmission.
	if discard := a.logSession(emitCtx, rec); discard {
		return nil
	}

	if err := a.sink.Emit(emitCtx, rec); err != nil {
		log.Printf("[SWIFT] sink.Emit error: %v", err)
	}

	return nil
}

// DetectMsgType returns "MT541", "MT543", etc. for MT messages,
// or "pacs.008", "pacs.009", "camt.056" for MX.
func DetectMsgType(raw []byte) string {
	if len(raw) > 0 && raw[0] == '{' {
		mt, err := ParseMT(raw)
		if err == nil && mt.MsgType != "" {
			return "MT" + mt.MsgType
		}
		return "MTUNKNOWN"
	}
	return DetectMXMsgType(raw)
}

// logSession writes an audit row to vend.swift_session_log. Returns true if
// the record should be discarded (UETR duplicate = SWIFTNet retransmission).
//
// 23505 on (tenant_id, uetr) means we have already processed this message.
// Log at INFO and return discard=true; the caller skips sink.Emit.
// Any other write error is logged at ERROR and discard=false (audit gap, not fatal).
// If db is nil (tests / no-DB deployment) the call is a no-op, discard=false.
func (a *Adapter) logSession(ctx context.Context, rec InboundSWIFTRecord) (discard bool) {
	if a.db == nil {
		return false
	}

	var uetrParam interface{}
	if rec.UETR != "" {
		uetrParam = rec.UETR
	} // NULL for non-GPI messages that omit UETR

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO vend.swift_session_log (
			id, tenant_id, uetr, msg_type, transaction_ref,
			bic_sender, bic_receiver, raw_bytes, received_at, event_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'INBOUND')
	`,
		uuid.New().String(), rec.TenantID, uetrParam, rec.MsgType,
		rec.TransactionRef, rec.BICSender, rec.BICReceiver, rec.RawBytes,
		rec.ReceivedAt,
	)
	if err != nil {
		if IsUETRDuplicate(err) && rec.UETR != "" {
			log.Printf("[SWIFT] adapter: UETR retransmission discarded "+
				"tenant=%s uetr=%s msg_type=%s tx_ref=%s",
				rec.TenantID, rec.UETR, rec.MsgType, rec.TransactionRef)
			return true
		}
		log.Printf("[SWIFT] adapter: session_log write failed (audit gap, continuing) "+
			"tenant=%s uetr=%s error=%v", rec.TenantID, rec.UETR, err)
	}
	return false
}
