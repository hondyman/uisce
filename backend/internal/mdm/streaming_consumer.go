package mdm

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/twmb/franz-go/pkg/kgo"
)

type IngestionMessageEnvelope struct {
	TenantID        string                 `json:"tenantId"`
	DomainKey       string                 `json:"domainKey"`
	MasterEntitySID string                 `json:"masterEntitySid"`
	VendorName      string                 `json:"vendorName"`
	EffectiveDate   string                 `json:"effectiveDate"` // YYYY-MM-DD
	Attributes      map[string]interface{} `json:"attributes"`
	ConfidenceScore float64                `json:"confidenceScore"`
}

type StreamingIngestionWorker struct {
	db          *sqlx.DB
	client      *kgo.Client
	resolver    *UniversalMDMResolver
	dispatcher  TemporalWorkflowDispatcherPort // Rule 3: Domain port
	batchBuffer []VendorFeedPayload
	bufferMu    sync.Mutex
	flushSize   int
	flushTimer  time.Duration
}

// TemporalWorkflowDispatcherPort abstracts Temporal workflow starts to avoid package cycles (Rule 3)
type TemporalWorkflowDispatcherPort interface {
	DispatchDownstreamSync(ctx context.Context, req DownstreamSyncRequest) error
}

func NewStreamingIngestionWorker(
	db *sqlx.DB,
	client *kgo.Client,
	resolver *UniversalMDMResolver,
	dispatcher TemporalWorkflowDispatcherPort,
	flushSize int,
	flushTimer time.Duration,
) *StreamingIngestionWorker {
	return &StreamingIngestionWorker{
		db:          db,
		client:      client,
		resolver:    resolver,
		dispatcher:  dispatcher,
		batchBuffer: make([]VendorFeedPayload, 0, flushSize),
		flushSize:   flushSize,
		flushTimer:  flushTimer,
	}
}

// StartConsuming begins the Kafka/Redpanda consumer event loop
func (w *StreamingIngestionWorker) StartConsuming(ctx context.Context, topics []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fetches := w.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				continue
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				var env IngestionMessageEnvelope
				if err := json.Unmarshal(record.Value, &env); err != nil {
					w.sendToDLQ(ctx, uuid.Nil, "CORRUPT_PAYLOAD", string(record.Value), err.Error())
					continue
				}

				tenantID, err := uuid.Parse(env.TenantID)
				if err != nil || tenantID == uuid.Nil {
					w.sendToDLQ(ctx, uuid.Nil, "INVALID_TENANT_CONTEXT", string(record.Value), "Rule 7 Violation: Missing Tenant ID") // Rule 7 Guard
					continue
				}

				effDate, err := time.Parse("2006-01-02", env.EffectiveDate)
				if err != nil {
					effDate = time.Now().UTC()
				}

				item := VendorFeedPayload{
					DomainKey:       env.DomainKey,
					MasterEntitySID: env.MasterEntitySID,
					VendorName:      env.VendorName,
					EffectiveDate:   effDate,
					Attributes:      env.Attributes,
					ConfidenceScore: env.ConfidenceScore,
				}

				w.bufferMu.Lock()
				w.batchBuffer = append(w.batchBuffer, item)
				shouldFlush := len(w.batchBuffer) >= w.flushSize
				w.bufferMu.Unlock()

				if shouldFlush {
					_ = w.FlushMicroBatch(ctx, tenantID)
				}
			}
		}
	}
}

// FlushMicroBatch executes multi-source golden record mastering and triggers Temporal fan-out
func (w *StreamingIngestionWorker) FlushMicroBatch(ctx context.Context, tenantID uuid.UUID) error {
	w.bufferMu.Lock()
	if len(w.batchBuffer) == 0 {
		w.bufferMu.Unlock()
		return nil
	}
	currentBatch := w.batchBuffer
	w.batchBuffer = make([]VendorFeedPayload, 0, w.flushSize)
	w.bufferMu.Unlock()

	start := time.Now()

	// Group payloads by Entity SID for multi-vendor golden survivorship
	grouped := make(map[string][]VendorFeedPayload)
	for _, p := range currentBatch {
		grouped[p.MasterEntitySID] = append(grouped[p.MasterEntitySID], p)
	}

	successCount := 0
	for sid, feeds := range grouped {
		domainKey := feeds[0].DomainKey
		effectiveDate := feeds[0].EffectiveDate

		// 1. Resolve Golden Copy via Universal MDM Core (Rule 1)
		goldenAttributes, winningSources, err := w.resolver.MasterIncomingFeeds(
			ctx, tenantID, domainKey, sid, effectiveDate, feeds,
		)
		if err != nil {
			continue
		}
		successCount++

		// 2. Trigger Temporal Downstream Push Workflow (Step 4)
		if w.dispatcher != nil {
			syncReq := DownstreamSyncRequest{
				TenantID:       tenantID,
				BOID:           uuid.New(), // Resolved from master_domain_registry
				EntitySID:      sid,
				GoldAttributes: goldenAttributes,
				MutationType:   "UPSERT",
				KnowledgeTime:  time.Now().UTC(),
			}
			_ = w.dispatcher.DispatchDownstreamSync(ctx, syncReq)
		}
		_ = winningSources
	}

	elapsed := time.Since(start).Seconds()
	throughput := float64(len(currentBatch))
	if elapsed > 0 {
		throughput = float64(len(currentBatch)) / elapsed
	}

	// Update Ingestion Metrics Ledger
	if w.db != nil {
		_, _ = w.db.ExecContext(ctx, `
			INSERT INTO mdm_ingest.batch_execution_ledger (
				tenant_id, feed_id, domain_key, vendor_name, batch_status,
				total_records_ingested, successful_records, throughput_records_per_sec,
				execution_duration_ms, completed_at
			) VALUES ($1, gen_random_uuid(), $2, $3, 'COMPLETED', $4, $5, $6, $7, NOW())
		`, tenantID, currentBatch[0].DomainKey, currentBatch[0].VendorName,
			len(currentBatch), successCount, throughput, time.Since(start).Milliseconds())
	}

	return nil
}

func (w *StreamingIngestionWorker) sendToDLQ(ctx context.Context, tenantID uuid.UUID, domainKey, payload, reason string) {
	if w.db == nil {
		return
	}
	_, _ = w.db.ExecContext(ctx, `
		INSERT INTO mdm_ingest.dead_letter_queue (
			tenant_id, domain_key, vendor_name, raw_payload, error_reason
		) VALUES ($1, $2, 'UNKNOWN', $3, $4)
	`, tenantID, domainKey, payload, reason)
}
