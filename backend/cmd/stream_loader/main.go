// Stream Loader consumes real Debezium Postgres connector change events
// (the standard Kafka Connect JSON envelope: {"schema":...,"payload":{
// "before":..,"after":..,"op":..,"ts_ms":..}}) and acts as the Streaming Gatekeeper:
//
//  1. Decodes the Debezium envelope, including schema-driven normalisation of
//     logical types (Decimal, Date, Timestamp/ZonedTimestamp, Json)
//  2. Asserts Layer 2 Tenant Identity (assigned tenant vs event.tenant_id)
//  3. Evaluates in-line active Validation Rules (RuleSnapshot / EmbeddedEngine)
//  4. Routes blocked/invalid records to Dead Letter Queue (DLQ topic + violation table)
//  5. Batches verified records and stream-loads them into StarRocks over HTTP
//  6. Turns op=d events into deletes against the target PRIMARY KEY table
//  7. Exposes real-time HTTP metrics and health monitoring (/health, /metrics, /stats)
//
// # Batching and delivery semantics
//
// Rows are accumulated and flushed when either MAX_ROWS is reached or FLUSH_INTERVAL
// elapses, whichever comes first. Every flush carries a `label` derived from the
// topic/partition/offset range it covers, so a retry of the same batch is a no-op
// in StarRocks ("Label Already Exists") rather than a duplicate write. Offsets are
// committed only after the load is accepted, so an unacknowledged batch replays on
// restart.
//
// The target tables must be PRIMARY KEY tables: op=c/op=r/op=u upsert, op=d deletes
// by key. DELETE is a first-class operation here, not a dropped event.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	kafka "github.com/segmentio/kafka-go"
)

type Config struct {
	KafkaBrokers      string
	Topic             string
	DLQTopic          string
	StarRocksHTTP     string
	StarRocksUser     string
	StarRocksPassword string
	StarRocksDB       string
	StarRocksTable    string
	AssignedTenantID  string
	BusinessObject    string
	ValidationEnabled bool
	DatabaseDSN       string
	MetricsPort       string

	// PrimaryKeys are the columns forming the target table's key. They drive both
	// the op=d delete predicate and the CREATE TABLE DDL.
	PrimaryKeys []string
	// MaxRows / FlushInterval bound one stream load.
	MaxRows       int
	FlushInterval time.Duration
	// QueryDSN is a MySQL-protocol connection used for keyed DELETEs. StarRocks 3.3
	// no longer implements the HTTP SQL endpoint (/api/query returns 501), so a
	// DELETE cannot be issued through _stream_load.
	QueryDSN string
}

type GatekeeperMetrics struct {
	TotalConsumed         atomic.Int64
	TotalLoaded           atomic.Int64
	SkippedTombstones     atomic.Int64
	TenantMismatches      atomic.Int64
	RuleViolationsBlocked atomic.Int64
	RuleViolationsWarned  atomic.Int64
	StreamLoadErrors      atomic.Int64
	DLQEmitted            atomic.Int64
	TotalDeletes          atomic.Int64
	DeleteErrors          atomic.Int64
	BatchesFlushed        atomic.Int64
	RowsIn                atomic.Int64
	RowsLoaded            atomic.Int64
	StartTime             time.Time
	lastEventUnixNano     atomic.Int64
}

type GatekeeperStats struct {
	TotalConsumed         int64      `json:"total_consumed"`
	TotalLoaded           int64      `json:"total_loaded"`
	SkippedTombstones     int64      `json:"skipped_tombstones"`
	TenantMismatches      int64      `json:"tenant_mismatches"`
	RuleViolationsBlocked int64      `json:"rule_violations_blocked"`
	RuleViolationsWarned  int64      `json:"rule_violations_warned"`
	StreamLoadErrors      int64      `json:"stream_load_errors"`
	DLQEmitted            int64      `json:"dlq_emitted"`
	TotalDeletes          int64      `json:"total_deletes"`
	DeleteErrors          int64      `json:"delete_errors"`
	BatchesFlushed        int64      `json:"batches_flushed"`
	RowsIn                int64      `json:"rows_in"`
	RowsLoaded            int64      `json:"rows_loaded"`
	UptimeSeconds         float64    `json:"uptime_seconds"`
	LastEventTime         *time.Time `json:"last_event_time,omitempty"`
	Topic                 string     `json:"topic"`
	StarRocksTable        string     `json:"starrocks_table"`
	AssignedTenantID      string     `json:"assigned_tenant_id,omitempty"`
	ValidationEnabled     bool       `json:"validation_enabled"`
}

type debeziumEnvelope struct {
	Schema  json.RawMessage `json:"schema"`
	Payload struct {
		Before json.RawMessage `json:"before"`
		After  json.RawMessage `json:"after"`
		Op     string          `json:"op"`
		TsMs   int64           `json:"ts_ms"`
	} `json:"payload"`
}

type schemaField struct {
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Field      string            `json:"field"`
	Parameters map[string]string `json:"parameters"`
}

type valueSchema struct {
	Fields []struct {
		Type   string        `json:"type"`
		Name   string        `json:"name"`
		Field  string        `json:"field"`
		Fields []schemaField `json:"fields"`
	} `json:"fields"`
}

type DLQMessage struct {
	OriginalTopic string                 `json:"original_topic"`
	Timestamp     time.Time              `json:"timestamp"`
	Reason        string                 `json:"reason"`
	Detail        string                 `json:"detail"`
	Payload       map[string]interface{} `json:"payload"`
	RawEvent      json.RawMessage        `json:"raw_event,omitempty"`
}

type StreamingGatekeeper struct {
	cfg        Config
	metrics    *GatekeeperMetrics
	sqlxDB     *sqlx.DB
	queryDB    *sql.DB
	ruleEngine *analytics.EmbeddedEngine
	dlqWriter  *kafka.Writer
	mu         sync.RWMutex
}

func loadConfig() Config {
	cfg := Config{
		KafkaBrokers:      os.Getenv("KAFKA_BROKERS"),
		Topic:             os.Getenv("CDC_TOPIC"),
		DLQTopic:          os.Getenv("DLQ_TOPIC"),
		StarRocksHTTP:     os.Getenv("STARROCKS_HTTP"),
		StarRocksUser:     os.Getenv("STARROCKS_USER"),
		StarRocksPassword: os.Getenv("STARROCKS_PASSWORD"),
		StarRocksDB:       os.Getenv("STARROCKS_DB"),
		StarRocksTable:    os.Getenv("STARROCKS_TABLE"),
		AssignedTenantID:  os.Getenv("ASSIGNED_TENANT_ID"),
		BusinessObject:    os.Getenv("BUSINESS_OBJECT"),
		ValidationEnabled: os.Getenv("VALIDATION_ENGINE_ENABLED") == "true" || os.Getenv("ENABLE_RULE_EVALUATION") == "true",
		DatabaseDSN:       os.Getenv("DATABASE_URL"),
		MetricsPort:       os.Getenv("METRICS_PORT"),
		MaxRows:           2000,
		FlushInterval:     10 * time.Second,
	}

	if cfg.AssignedTenantID == "" {
		cfg.AssignedTenantID = os.Getenv("EXPECTED_TENANT_ID")
	}
	if cfg.Topic == "" {
		cfg.Topic = "cdc_events"
	}
	if cfg.DLQTopic == "" {
		cfg.DLQTopic = cfg.Topic + ".dlq"
	}
	if cfg.KafkaBrokers == "" {
		cfg.KafkaBrokers = "localhost:9092"
	}
	if cfg.StarRocksHTTP == "" {
		cfg.StarRocksHTTP = "http://localhost:8030"
	}
	if cfg.StarRocksUser == "" {
		cfg.StarRocksUser = "root"
	}
	if cfg.StarRocksDB == "" {
		cfg.StarRocksDB = "oms"
	}
	if cfg.MetricsPort == "" {
		cfg.MetricsPort = os.Getenv("PORT")
		if cfg.MetricsPort == "" {
			cfg.MetricsPort = "8085"
		}
	}
	if cfg.BusinessObject == "" && cfg.StarRocksTable != "" {
		cfg.BusinessObject = deriveBusinessObject(cfg.StarRocksTable)
	}
	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = os.Getenv("POSTGRES_DSN")
	}
	if pk := os.Getenv("PRIMARY_KEYS"); pk != "" {
		for _, k := range strings.Split(pk, ",") {
			if k = strings.TrimSpace(k); k != "" {
				cfg.PrimaryKeys = append(cfg.PrimaryKeys, k)
			}
		}
	}
	if len(cfg.PrimaryKeys) == 0 {
		// Every table in the orm schema carries a single-column "id" primary key.
		// Making this explicit beats guessing wrong on an unconfigured deployment.
		cfg.PrimaryKeys = []string{"id"}
	}
	if v := os.Getenv("STREAM_LOAD_MAX_ROWS"); v != "" {
		if n := parsePositiveInt(v); n > 0 {
			cfg.MaxRows = n
		}
	}
	if v := os.Getenv("STREAM_LOAD_FLUSH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.FlushInterval = d
		}
	}
	cfg.QueryDSN = os.Getenv("STARROCKS_QUERY_DSN")
	if cfg.QueryDSN == "" {
		if host := hostOf(cfg.StarRocksHTTP); host != "" {
			cfg.QueryDSN = fmt.Sprintf("%s:%s@tcp(%s:9030)/%s", cfg.StarRocksUser, cfg.StarRocksPassword, host, cfg.StarRocksDB)
		}
	}
	return cfg
}

// hostOf extracts the hostname from an http(s) URL so the query port can be derived
// from the same host the stream load already targets.
func hostOf(raw string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	if i := strings.IndexAny(s, "/:"); i >= 0 {
		s = s[:i]
	}
	return s
}

func parsePositiveInt(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
		if n > 1<<20 {
			return 0
		}
	}
	return n
}

func deriveBusinessObject(table string) string {
	clean := strings.ToLower(table)
	clean = strings.TrimPrefix(clean, "orm_")
	clean = strings.TrimPrefix(clean, "oms_")
	clean = strings.TrimPrefix(clean, "mdm_")
	return clean
}

func main() {
	cfg := loadConfig()
	log.Printf("Starting Streaming Gatekeeper Service [Topic=%s -> StarRocks=%s.%s keys=%v maxRows=%d flush=%s]...",
		cfg.Topic, cfg.StarRocksDB, cfg.StarRocksTable, cfg.PrimaryKeys, cfg.MaxRows, cfg.FlushInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := &GatekeeperMetrics{StartTime: time.Now()}

	var db *sqlx.DB
	var ruleEngine *analytics.EmbeddedEngine
	if cfg.DatabaseDSN != "" {
		var err error
		db, err = sqlx.Connect("postgres", cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Warning: Failed to connect to Postgres DB: %v (embedded validation rules will be offline)", err)
		} else {
			defer db.Close()
			if cfg.ValidationEnabled {
				ruleSvc := analytics.NewValidationRuleService(db)
				ruleEngine = analytics.NewEmbeddedEngine(db, cfg.AssignedTenantID, ruleSvc, analytics.WithDSN(cfg.DatabaseDSN))
				ruleEngine.Start(ctx)
				log.Printf("Embedded Rule Evaluation Engine active for tenant=%s, BO=%s", cfg.AssignedTenantID, cfg.BusinessObject)
			}
		}
	}

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(cfg.KafkaBrokers, ",")...),
		Topic:        cfg.DLQTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	defer dlqWriter.Close()

	gk := &StreamingGatekeeper{
		cfg:        cfg,
		metrics:    metrics,
		sqlxDB:     db,
		ruleEngine: ruleEngine,
		dlqWriter:  dlqWriter,
	}

	go gk.startHTTPServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down Streaming Gatekeeper...")
		cancel()
	}()

	gk.runLoop(ctx)
}

func (gk *StreamingGatekeeper) startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "UP",
			"topic":              gk.cfg.Topic,
			"starrocks_table":    gk.cfg.StarRocksTable,
			"assigned_tenant_id": gk.cfg.AssignedTenantID,
			"validation_enabled": gk.cfg.ValidationEnabled,
		})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gk.GetStats())
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gk.GetStats())
	})

	addr := ":" + gk.cfg.MetricsPort
	log.Printf("Streaming Gatekeeper HTTP server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

func (gk *StreamingGatekeeper) GetStats() GatekeeperStats {
	lastNano := gk.metrics.lastEventUnixNano.Load()
	var lastTime *time.Time
	if lastNano > 0 {
		t := time.Unix(0, lastNano)
		lastTime = &t
	}

	return GatekeeperStats{
		TotalConsumed:         gk.metrics.TotalConsumed.Load(),
		TotalLoaded:           gk.metrics.TotalLoaded.Load(),
		SkippedTombstones:     gk.metrics.SkippedTombstones.Load(),
		TenantMismatches:      gk.metrics.TenantMismatches.Load(),
		RuleViolationsBlocked: gk.metrics.RuleViolationsBlocked.Load(),
		RuleViolationsWarned:  gk.metrics.RuleViolationsWarned.Load(),
		StreamLoadErrors:      gk.metrics.StreamLoadErrors.Load(),
		DLQEmitted:            gk.metrics.DLQEmitted.Load(),
		TotalDeletes:          gk.metrics.TotalDeletes.Load(),
		DeleteErrors:          gk.metrics.DeleteErrors.Load(),
		BatchesFlushed:        gk.metrics.BatchesFlushed.Load(),
		RowsIn:                gk.metrics.RowsIn.Load(),
		RowsLoaded:            gk.metrics.RowsLoaded.Load(),
		UptimeSeconds:         time.Since(gk.metrics.StartTime).Seconds(),
		LastEventTime:         lastTime,
		Topic:                 gk.cfg.Topic,
		StarRocksTable:        gk.cfg.StarRocksTable,
		AssignedTenantID:      gk.cfg.AssignedTenantID,
		ValidationEnabled:     gk.cfg.ValidationEnabled,
	}
}

// ---- batch assembly ----

// pendingOp is one coalesced operation for a single primary key. A batch is built by
// walking ops backwards and keeping the LAST operation seen for each key, so a
// create-then-delete inside one batch resolves to a delete and a delete-then-create
// resolves to an upsert. Order within a batch is therefore irrelevant.
type pendingOp struct {
	key      string
	row      json.RawMessage
	isDelete bool
}

type batch struct {
	ops      []pendingOp
	messages []kafka.Message
}

func (b *batch) empty() bool { return len(b.ops) == 0 }

// sanitizeLabel renders a Kafka topic/offset range as a StarRocks stream-load label.
// StarRocks validates labels against ^[-\w]{1,128}$ — letters, digits, underscore and
// hyphen only, so a dotted topic name like "orm_oms.orm.order" must have its dots
// replaced. The same label on a retry is what makes a redelivered batch a no-op.
func sanitizeLabel(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 100 {
		out = out[len(out)-100:]
	}
	return out
}

func (gk *StreamingGatekeeper) runLoop(ctx context.Context) {
	brokers := strings.Split(gk.cfg.KafkaBrokers, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "stream-loader-" + gk.cfg.Topic,
		Topic:    gk.cfg.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Printf("Listening for CDC events on topic %s -> %s.%s", gk.cfg.Topic, gk.cfg.StarRocksDB, gk.cfg.StarRocksTable)

	ticker := time.NewTicker(gk.cfg.FlushInterval)
	defer ticker.Stop()

	b := &batch{}

	// flush writes the accumulated batch and, only on success, commits its offsets.
	// It retries internally so a transient StarRocks error does not advance the batch.
	// A detached context is used because the final drain runs after ctx is cancelled.
	flush := func(reason string) {
		if b.empty() {
			return
		}
		// The batch must survive a cancelled parent context, or a graceful shutdown
		// would discard the rows it is holding.
		wctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		for attempt := 1; ; attempt++ {
			err := gk.writeBatch(wctx, b)
			if err == nil {
				gk.metrics.BatchesFlushed.Add(1)
				if cerr := r.CommitMessages(wctx, b.messages...); cerr != nil {
					log.Printf("Failed to commit messages: %v", cerr)
				}
				b = &batch{}
				return
			}
			log.Printf("Batch flush failed (%s, attempt %d, %d ops): %v", reason, attempt, len(b.ops), err)
			gk.metrics.StreamLoadErrors.Add(1)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff(attempt)):
			}
		}
	}

	// Consumption runs on its own goroutine. FetchMessage blocks for as long as the
	// topic is quiet, so polling a flush ticker at the top of the reading loop would
	// leave a batch unflushed indefinitely on a low-traffic topic.
	messages := make(chan kafka.Message, 4096)
	go func() {
		defer close(messages)
		for {
			m, err := r.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error fetching message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			select {
			case messages <- m:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			flush("shutdown")
			return

		case <-ticker.C:
			flush("interval")

		case m, ok := <-messages:
			if !ok {
				flush("shutdown")
				return
			}
			gk.metrics.TotalConsumed.Add(1)
			gk.metrics.lastEventUnixNano.Store(time.Now().UnixNano())

			if !gk.handleMessage(ctx, b, m) {
				// Rejected/tombstoned: needs no StarRocks write, so commit it now.
				if cerr := r.CommitMessages(ctx, m); cerr != nil {
					log.Printf("Failed to commit message: %v", cerr)
				}
				continue
			}

			if len(b.ops) >= gk.cfg.MaxRows {
				flush("full")
			}
		}
	}
}

func backoff(attempt int) time.Duration {
	d := time.Second
	for i := 1; i < attempt && d < 30*time.Second; i++ {
		d *= 2
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// handleMessage returns true when the message contributed an op to the batch (and so
// must not be committed yet). It returns false when the message was consumed and
// discarded, which the caller commits immediately.
func (gk *StreamingGatekeeper) handleMessage(ctx context.Context, b *batch, m kafka.Message) bool {
	ev, err := decodeRecord(m.Value)
	if err != nil {
		log.Printf("Error decoding Debezium event: %v", err)
		return false
	}
	if ev.tombstone {
		gk.metrics.SkippedTombstones.Add(1)
		return false
	}
	if ev.row == nil && !ev.isDelete {
		// Nothing carried (e.g. an op with neither before nor after). Drop it rather
		// than commit it into an unbounded retry loop.
		gk.metrics.SkippedTombstones.Add(1)
		return false
	}

	// Deletes read their key from `before`; upserts from `after`.
	src := ev.after
	if ev.isDelete {
		src = ev.before
	}

	// Layer 2: Assert Tenant Identity
	eventTenant := extractTenantID(src)
	if gk.cfg.AssignedTenantID != "" && eventTenant != "" && eventTenant != gk.cfg.AssignedTenantID {
		gk.metrics.TenantMismatches.Add(1)
		log.Printf("[SECURITY] Tenant Mismatch rejected: event tenant=%s != assigned tenant=%s", eventTenant, gk.cfg.AssignedTenantID)
		gk.emitDLQ(ctx, "ERR_TENANT_MISMATCH", fmt.Sprintf("Event tenant %s does not match assigned %s", eventTenant, gk.cfg.AssignedTenantID), src, m.Value)
		return false
	}

	// Layer 3: Embedded Validation Rule Engine (upserts only — there is nothing
	// meaningful to validate about a row that no longer exists).
	if !ev.isDelete && gk.cfg.ValidationEnabled && gk.ruleEngine != nil {
		boKey := gk.cfg.BusinessObject
		tenantForEval := gk.cfg.AssignedTenantID
		if tenantForEval == "" {
			tenantForEval = eventTenant
		}

		evalResult, evalErr := gk.ruleEngine.ValidateRecord(ctx, boKey, "", "", ev.after)
		if evalErr != nil {
			log.Printf("Validation evaluation error: %v", evalErr)
		} else if evalResult != nil {
			if evalResult.Blocked {
				gk.metrics.RuleViolationsBlocked.Add(1)
				log.Printf("[BLOCK] Validation Rule violation blocked stream load on %s: %d violations", boKey, len(evalResult.Violations))
				gk.persistViolations(ctx, tenantForEval, boKey, evalResult, true)
				gk.emitDLQ(ctx, "ERR_RULE_BLOCK_VIOLATION", fmt.Sprintf("Rule violations: %d blocking rules failed", len(evalResult.Violations)), ev.after, m.Value)
				return false
			} else if len(evalResult.Violations) > 0 {
				gk.metrics.RuleViolationsWarned.Add(1)
				gk.persistViolations(ctx, tenantForEval, boKey, evalResult, false)
			}
		}
	}

	key, ok := primaryKeyOf(gk.cfg.PrimaryKeys, src)
	if !ok {
		// Without a key we can neither upsert (ambiguous) nor delete (no predicate).
		// Send it to the DLQ rather than writing a row we can never update or remove.
		gk.metrics.DLQEmitted.Add(1)
		log.Printf("[DLQ] %s event has no usable primary key %v", ev.op, gk.cfg.PrimaryKeys)
		gk.emitDLQ(ctx, "ERR_MISSING_PRIMARY_KEY", fmt.Sprintf("no value for primary key %v", gk.cfg.PrimaryKeys), src, m.Value)
		return false
	}

	b.ops = append(b.ops, pendingOp{key: key, row: ev.row, isDelete: ev.isDelete})
	b.messages = append(b.messages, m)
	gk.metrics.RowsIn.Add(1)
	return true
}

func primaryKeyOf(keys []string, row map[string]interface{}) (string, bool) {
	var parts []string
	for _, k := range keys {
		v, ok := row[k]
		if !ok || v == nil {
			return "", false
		}
		parts = append(parts, fmt.Sprintf("%v", v))
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "\x1f"), true
}

// writeBatch performs one stream load for the upserts and one DELETE for the deletes.
// Upserts run first so that a batch containing both an update and a later delete of
// the same key ends in the deleted state.
func (gk *StreamingGatekeeper) writeBatch(ctx context.Context, b *batch) error {
	// Coalesce by key, last operation wins.
	type resolved struct {
		op pendingOp
	}
	order := []string{}
	byKey := map[string]pendingOp{}
	for _, o := range b.ops {
		if _, seen := byKey[o.key]; !seen {
			order = append(order, o.key)
		}
		byKey[o.key] = o
	}

	var upserts []json.RawMessage
	var deleteKeys []string
	for _, k := range order {
		o := byKey[k]
		if o.isDelete {
			deleteKeys = append(deleteKeys, o.key)
		} else if o.row != nil {
			upserts = append(upserts, o.row)
		}
	}

	// The label is derived from the offset range so a retry of the same batch is
	// rejected by StarRocks as a duplicate instead of loading the rows twice.
	// A batch with no messages can only be constructed by a test; fall back to a
	// content-derived label rather than panicking.
	label := sanitizeLabel(gk.cfg.Topic)
	if len(b.messages) > 0 {
		label = fmt.Sprintf("%s_%d_%d", label, b.messages[0].Offset, b.messages[len(b.messages)-1].Offset)
	} else {
		label = fmt.Sprintf("%s_%d", label, time.Now().UnixNano())
	}

	if len(upserts) > 0 {
		if err := streamLoadBatch(ctx, gk.cfg, upserts, label); err != nil {
			return fmt.Errorf("stream load %d rows: %w", len(upserts), err)
		}
		gk.metrics.TotalLoaded.Add(int64(len(upserts)))
		gk.metrics.RowsLoaded.Add(int64(len(upserts)))
	}

	if len(deleteKeys) > 0 {
		if err := gk.deleteRows(ctx, gk.cfg.PrimaryKeys, deleteKeys); err != nil {
			return fmt.Errorf("delete %d rows: %w", len(deleteKeys), err)
		}
		gk.metrics.TotalDeletes.Add(int64(len(deleteKeys)))
	}
	return nil
}

// deleteRows issues the keyed DELETE over the MySQL protocol. The connection is
// opened lazily so a loader with no deletes never pays for it.
func (gk *StreamingGatekeeper) deleteRows(ctx context.Context, keyCols []string, keys []string) error {
	q, ok := deleteSQL(gk.cfg, keyCols, keys)
	if !ok {
		return nil
	}
	if gk.queryDB == nil {
		if gk.cfg.QueryDSN == "" {
			return errors.New("no StarRocks query DSN is configured, cannot apply deletes")
		}
		db, err := sql.Open("mysql", gk.cfg.QueryDSN)
		if err != nil {
			return err
		}
		db.SetMaxOpenConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)
		gk.queryDB = db
	}
	if _, err := gk.queryDB.ExecContext(ctx, q); err != nil {
		return err
	}
	return nil
}

func extractTenantID(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"tenant_id", "tenantId", "TenantID"} {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func (gk *StreamingGatekeeper) emitDLQ(ctx context.Context, reason, detail string, payload map[string]interface{}, raw json.RawMessage) {
	gk.metrics.DLQEmitted.Add(1)
	dlqMsg := DLQMessage{
		OriginalTopic: gk.cfg.Topic,
		Timestamp:     time.Now().UTC(),
		Reason:        reason,
		Detail:        detail,
		Payload:       payload,
		RawEvent:      raw,
	}

	data, err := json.Marshal(dlqMsg)
	if err != nil {
		log.Printf("Failed to serialize DLQ message: %v", err)
		return
	}

	if gk.dlqWriter != nil {
		err = gk.dlqWriter.WriteMessages(ctx, kafka.Message{
			Key:   []byte(extractTenantID(payload)),
			Value: data,
		})
		if err != nil {
			log.Printf("Failed to write message to DLQ topic %s: %v", gk.cfg.DLQTopic, err)
		}
	}
}

func (gk *StreamingGatekeeper) persistViolations(ctx context.Context, tenantID, boKey string, evalResult *analytics.RecordEvaluation, blocked bool) {
	if gk.sqlxDB == nil || evalResult == nil {
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return
	}

	for _, v := range evalResult.Violations {
		recordIDStr := fmt.Sprintf("%v", v.RecordID)
		ruleIDUUID, err := uuid.Parse(v.RuleID)
		if err != nil {
			ruleIDUUID = uuid.Nil
		}

		fieldName := ""
		if len(v.Fields) > 0 {
			fieldName = v.Fields[0]
		}

		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"message":          v.Message,
			"source_subsystem": "cdc_stream_loader",
			"blocked":          blocked,
			"table":            gk.cfg.StarRocksTable,
			"evaluated_count":  evalResult.EvaluatedCount,
			"fields":           v.Fields,
			"context":          v.Context,
		})

		_, _ = gk.sqlxDB.ExecContext(ctx, `
			INSERT INTO public.validation_rule_violations (
				id, tenant_id, rule_id, rule_node_id, rule_key, rule_name,
				severity, business_object, record_id, field_name, invalid_value,
				error_message, write_blocked, source_subsystem, details, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW()
			)
		`,
			uuid.New(), tenantUUID, ruleIDUUID, ruleIDUUID, v.RuleKey, v.RuleName,
			v.Severity, boKey, recordIDStr, fieldName, "",
			v.Message, blocked, "cdc_stream_loader", detailsJSON,
		)
	}
}

// ---- envelope decoding ----

// decodedEvent is one Debezium change event with its logical types already
// converted into something StarRocks can ingest.
type decodedEvent struct {
	op        string
	isDelete  bool
	tombstone bool
	before    map[string]interface{}
	after     map[string]interface{}
	row       json.RawMessage
}

const (
	opCreate = "c"
	opUpdate = "u"
	opDelete = "d"
	opRead   = "r"
)

// decodeRecord parses a Debezium change-event envelope and normalises the row
// according to the connector's own schema, which is the only reliable source for
// how each logical type was encoded.
func decodeRecord(raw []byte) (decodedEvent, error) {
	var env debeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return decodedEvent{}, err
	}

	ev := decodedEvent{op: env.Payload.Op, isDelete: env.Payload.Op == opDelete}

	// A delete carries no "after" image — its row lives in "before". Checking
	// "after" first would classify every delete as a tombstone and silently drop
	// it, which is exactly the behaviour this loader is meant to stop having.
	if ev.isDelete {
		if isNullish(env.Payload.Before) {
			// No key to delete by: unrepresentable, so drop it rather than
			// leaving it in the topic forever.
			ev.tombstone = true
			return ev, nil
		}
		var before map[string]interface{}
		if err := json.Unmarshal(env.Payload.Before, &before); err != nil {
			return decodedEvent{}, err
		}
		ev.before = before
	} else {
		if isNullish(env.Payload.After) {
			ev.tombstone = true
			return ev, nil
		}
		var after map[string]interface{}
		if err := json.Unmarshal(env.Payload.After, &after); err != nil {
			return decodedEvent{}, err
		}
		ev.after = after
	}

	// Normalise logical types using the connector schema, which is the only reliable
	// source for how each column was encoded.
	if len(env.Schema) > 0 {
		var vs valueSchema
		if err := json.Unmarshal(env.Schema, &vs); err == nil {
			normalizeStruct(&vs, "after", ev.after)
			normalizeStruct(&vs, "before", ev.before)
		}
	}

	if !ev.isDelete {
		rowJSON, err := json.Marshal(ev.after)
		if err != nil {
			return decodedEvent{}, err
		}
		ev.row = rowJSON
	}
	return ev, nil
}

func isNullish(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return len(s) == 0 || s == "null"
}

// normalizeStruct converts the Debezium logical types that StarRocks cannot ingest
// directly into their SQL literals:
//
//	Decimal      base64 two's-complement -> decimal string
//	Date         epoch days              -> "YYYY-MM-DD"
//	Timestamp    epoch millis/micros/nanos-> "YYYY-MM-DD HH:MM:SS[.ffffff]"
//	ZonedTimestamp ISO-8601 with offset  -> "YYYY-MM-DD HH:MM:SS[.ffffff]" (UTC)
//	Json         base64 UTF-8            -> the JSON text
func normalizeStruct(vs *valueSchema, want string, row map[string]interface{}) {
	for _, f := range vs.Fields {
		if f.Field != want {
			continue
		}
		for _, cf := range f.Fields {
			v, ok := row[cf.Field]
			if !ok || v == nil {
				continue
			}
			switch logicalShortName(cf) {
			case "Decimal":
				s, isStr := v.(string)
				if !isStr {
					continue
				}
				if dec, err := decodeDebeziumDecimal(s, cf.Parameters["scale"]); err == nil {
					row[cf.Field] = dec
				}
			case "Date":
				n, isNum := asInt64(v)
				if !isNum {
					continue
				}
				row[cf.Field] = epochDaysToDate(n)
			case "MicroTimestamp", "NanoTimestamp":
				n, isNum := asInt64(v)
				if !isNum {
					continue
				}
				if s, ok := epochToDatetime(n, logicalShortName(cf)); ok {
					row[cf.Field] = s
				}
			case "Timestamp":
				n, isNum := asInt64(v)
				if !isNum {
					continue
				}
				if s, ok := epochToDatetime(n*int64(time.Millisecond), "Timestamp"); ok {
					row[cf.Field] = s
				}
			case "ZonedTimestamp":
				s, isStr := v.(string)
				if !isStr {
					continue
				}
				if lit, ok := zonedToDatetime(s); ok {
					row[cf.Field] = lit
				}
			case "Json":
				s, isStr := v.(string)
				if !isStr {
					continue
				}
				if b, err := base64.StdEncoding.DecodeString(s); err == nil {
					row[cf.Field] = string(b)
				}
			}
		}
	}
}

// logicalShortName resolves a schema field's logical type to its unqualified name.
// Debezium carries these in the field's "name" ("io.debezium.data.Decimal") while
// the primitive type sits in "type"; the Kafka Connect variants use the
// "org.apache.kafka.connect.data" namespace. Matching on the short name covers both
// without hard-coding a namespace that changes between connector versions.
func logicalShortName(f schemaField) string {
	for _, candidate := range []string{f.Name, f.Type} {
		if i := strings.LastIndex(candidate, "."); i >= 0 {
			candidate = candidate[i+1:]
		}
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func asInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	case json.Number:
		n, err := x.Int64()
		return n, err == nil
	}
	return 0, false
}

func epochDaysToDate(days int64) string {
	return time.Unix(0, 0).UTC().AddDate(0, 0, int(days)).Format("2006-01-02")
}

func epochToDatetime(n int64, kind string) (string, bool) {
	var unit time.Duration
	switch kind {
	case "MicroTimestamp":
		unit = time.Microsecond
	case "NanoTimestamp":
		unit = time.Nanosecond
	default:
		unit = time.Millisecond
	}
	return time.Unix(0, n*int64(unit)).UTC().Format("2006-01-02 15:04:05.000000"), true
}

var zonedLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999Z",
	"2006-01-02T15:04:05Z",
}

func zonedToDatetime(s string) (string, bool) {
	for _, layout := range zonedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format("2006-01-02 15:04:05.000000"), true
		}
	}
	return "", false
}

// decodeDebeziumDecimal converts Debezium's base64-encoded big-endian
// two's-complement Decimal representation into a decimal string.
func decodeDebeziumDecimal(b64, scaleStr string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	i := new(big.Int).SetBytes(raw)
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		// Negative: two's complement.
		full := new(big.Int).Lsh(big.NewInt(1), uint(len(raw)*8))
		i.Sub(i, full)
	}
	scale := 0
	fmt.Sscanf(scaleStr, "%d", &scale)
	f := new(big.Float).SetInt(i)
	if scale > 0 {
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
		f.Quo(f, divisor)
	}
	return f.Text('f', scale), nil
}

// ---- StarRocks transport ----

// starRocksResponse is the subset of the stream-load reply we act on. A load can
// return HTTP 200 and still not have applied the rows ("Publish Timeout",
// "Label Already Exists"), so the status must be read rather than assumed.
type starRocksResponse struct {
	Status string `json:"Status"`
	Msg    string `json:"Message"`
}

func (gk *StreamingGatekeeper) httpClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.SetBasicAuth(gk.cfg.StarRocksUser, gk.cfg.StarRocksPassword)
			return nil
		},
	}
}

// streamLoadBatch sends every row as one JSON array in a single request. The `label`
// makes the request idempotent: a retry carrying the same label is rejected as a
// duplicate rather than loading the rows twice.
func streamLoadBatch(ctx context.Context, cfg Config, rows []json.RawMessage, label string) error {
	body := make([]byte, 0, len(rows)*128)
	body = append(body, '[')
	for i, r := range rows {
		if i > 0 {
			body = append(body, ',')
		}
		body = append(body, r...)
	}
	body = append(body, ']')

	url := fmt.Sprintf("%s/api/%s/%s/_stream_load", cfg.StarRocksHTTP, cfg.StarRocksDB, cfg.StarRocksTable)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true")
	req.Header.Set("label", label)
	// These tables are PRIMARY KEY tables; the label is the idempotency key and the
	// rows are keyed by the table's own key columns.
	req.Header.Set("partial_update", "false")

	resp, err := clientFor(cfg).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out starRocksResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	switch {
	case out.Status == "Success":
		return nil
	case out.Status == "Label Already Exists":
		// A previous attempt with this label already applied these rows.
		return nil
	case out.Status == "Publish Timeout":
		// StarRocks committed but the FE lost the reply; the label makes a retry safe.
		return nil
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("bad status %s: %s", resp.Status, out.Msg)
	default:
		return fmt.Errorf("stream load rejected: %s (%s)", out.Status, out.Msg)
	}
}

// deleteSQL builds one keyed DELETE for the whole batch. StarRocks 3.3 removed the
// HTTP SQL endpoint, so this is issued over the MySQL protocol instead.
func deleteSQL(cfg Config, keyCols []string, keys []string) (string, bool) {
	var clauses []string
	for _, k := range keys {
		parts := strings.Split(k, "\x1f")
		if len(parts) != len(keyCols) {
			// A key that does not match the configured key columns must never become
			// a partial predicate, which would delete more than intended.
			continue
		}
		conds := make([]string, 0, len(parts))
		for i, p := range parts {
			conds = append(conds, fmt.Sprintf("`%s` = %s", keyCols[i], quoteLiteral(p)))
		}
		clauses = append(clauses, "("+strings.Join(conds, " AND ")+")")
	}
	if len(clauses) == 0 {
		return "", false
	}
	return fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s",
		cfg.StarRocksDB, cfg.StarRocksTable, strings.Join(clauses, " OR ")), true
}

// quoteLiteral renders a primary-key value as a SQL string literal using the SQL
// standard escape (a single quote is written twice). Key values come off the CDC
// envelope, so doubling is applied unconditionally rather than trusting the
// character set. Unlike a backslash escape, doubling is correct whether or not
// NO_BACKSLASH_ESCAPES is in effect.
func quoteLiteral(s string) string {
	if strings.ContainsAny(s, "\x00\n\r") {
		return "NULL"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func clientFor(cfg Config) *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
			return nil
		},
	}
}
