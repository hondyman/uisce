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
	// StrictMode is on by default. Without it StarRocks silently writes NULL into any
	// column it cannot convert and still reports Success with zero filtered rows --
	// verified 3.3.22 with a DECIMAL fed "NOT_A_NUMBER". loadRows bisects a rejected
	// batch so one bad row cannot block the rest.
	StrictMode bool
	// TenantRoutesDir points at the per-tenant credential files written by
	// scripts/provision_starrocks_tenants.sh. When it is set and contains routes, the
	// loader routes every event to its own tenant's StarRocks database and principal
	// instead of the single shared destination. Empty keeps the single-destination
	// behaviour, which is still correct for a genuinely single-tenant deployment.
	TenantRoutesDir string
	// SourceDatabase records which Postgres database the connector is capturing. It
	// is metadata, not a routing key: alpha is one shared database discriminated by
	// the tenant_id column, so a single replication slot serves every tenant. If
	// per-tenant Postgres databases are ever provisioned, the discriminator moves to
	// this value and the slot topology becomes one slot per tenant database.
	SourceDatabase string
}

type GatekeeperMetrics struct {
	TotalConsumed         atomic.Int64
	TotalLoaded           atomic.Int64
	SkippedTombstones     atomic.Int64
	SkippedHeartbeats     atomic.Int64
	SkippedSchemaChanges  atomic.Int64
	SkippedUnknown        atomic.Int64
	TenantMismatches      atomic.Int64
	TenantUnattributed    atomic.Int64
	TenantUnknown         atomic.Int64
	RowsRouted            atomic.Int64
	LoadsPerFlush         atomic.Int64
	Flushes               atomic.Int64
	FatalLoads            atomic.Int64
	RuleViolationsBlocked atomic.Int64
	RuleViolationsWarned  atomic.Int64
	StreamLoadErrors      atomic.Int64
	DLQEmitted            atomic.Int64
	DLQUnavailable        atomic.Int64
	TopologyTripwireFired atomic.Int64
	TripwireQueryErrors   atomic.Int64
	TotalDeletes          atomic.Int64
	DeleteErrors          atomic.Int64
	BatchesFlushed        atomic.Int64
	RowsIn                atomic.Int64
	RowsLoaded            atomic.Int64
	RowsRejected          atomic.Int64
	StartTime             time.Time
	lastEventUnixNano     atomic.Int64
}

type GatekeeperStats struct {
	TotalConsumed         int64 `json:"total_consumed"`
	TotalLoaded           int64 `json:"total_loaded"`
	SkippedTombstones     int64 `json:"skipped_tombstones"`
	SkippedHeartbeats     int64 `json:"skipped_heartbeats"`
	SkippedSchemaChanges  int64 `json:"skipped_schema_changes"`
	SkippedUnknown        int64 `json:"skipped_unknown"`
	TenantMismatches      int64 `json:"tenant_mismatches"`
	TenantUnattributed    int64 `json:"tenant_unattributed"`
	TenantUnknown         int64 `json:"tenant_unknown_route"`
	RowsRouted            int64 `json:"rows_routed"`
	LoadsPerFlush         int64 `json:"loads_per_flush_last"`
	Flushes               int64 `json:"flushes"`
	FatalLoads            int64 `json:"fatal_loads"`
	RuleViolationsBlocked int64 `json:"rule_violations_blocked"`
	RuleViolationsWarned  int64 `json:"rule_violations_warned"`
	StreamLoadErrors      int64 `json:"stream_load_errors"`
	DLQEmitted            int64 `json:"dlq_emitted"`
	DLQUnavailable        int64 `json:"dlq_unavailable"`
	TopologyTripwireFired int64 `json:"topology_tripwire_fired"`
	// OffsetFloorHoldSeconds is how long each topic-partition has been unable to
	// commit because some tenant's rows have not settled. A growing value means a
	// tenant is failing, and is otherwise indistinguishable from a hung consumer.
	OffsetFloorHoldSeconds map[string]float64 `json:"offset_floor_hold_seconds"`
	TotalDeletes           int64              `json:"total_deletes"`
	DeleteErrors           int64              `json:"delete_errors"`
	BatchesFlushed         int64              `json:"batches_flushed"`
	RowsIn                 int64              `json:"rows_in"`
	RowsLoaded             int64              `json:"rows_loaded"`
	RowsRejected           int64              `json:"rows_rejected_by_starrocks"`
	UptimeSeconds          float64            `json:"uptime_seconds"`
	LastEventTime          *time.Time         `json:"last_event_time,omitempty"`
	Topic                  string             `json:"topic"`
	StarRocksTable         string             `json:"starrocks_table"`
	AssignedTenantID       string             `json:"assigned_tenant_id,omitempty"`
	ValidationEnabled      bool               `json:"validation_enabled"`
	TenantRoutingEnabled   bool               `json:"tenant_routing_enabled"`
	TenantRoutesKnown      int                `json:"tenant_routes_known"`
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

// dlqPublisher is the slice of *kafka.Writer the loader needs. It exists so the
// dead-letter path can be exercised without a broker, which matters because a
// dead-letter write that fails is the difference between "this row is recorded
// somewhere" and "this row has vanished".
type dlqPublisher interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type StreamingGatekeeper struct {
	cfg        Config
	metrics    *GatekeeperMetrics
	sqlxDB     *sqlx.DB
	ruleEngine *analytics.EmbeddedEngine
	dlqWriter  dlqPublisher

	// pool holds the MySQL-protocol handles used for keyed DELETEs. Under tenant
	// routing each tenant's deletes must run as that tenant's own principal -- a single
	// shared handle would either cross tenant boundaries or be denied -- so it is keyed
	// by DSN and bounded, because one handle per tenant is one FE session per tenant.
	pool *queryDBPool

	// router resolves each event's tenant to a StarRocks database and principal. Nil
	// (or empty) means single-destination mode.
	router *routerCache

	// floor stops an offset commit from stepping over a tenant whose rows have not
	// landed yet. Required once a flush window can settle per tenant.
	floor *offsetFloor
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
	cfg.StrictMode = true
	if v := os.Getenv("STREAM_LOAD_STRICT_MODE"); v != "" {
		cfg.StrictMode = !strings.EqualFold(v, "false")
	}
	cfg.TenantRoutesDir = os.Getenv("STREAM_LOAD_TENANT_ROUTES_DIR")
	cfg.SourceDatabase = os.Getenv("CDC_SOURCE_DB")
	if cfg.SourceDatabase == "" {
		cfg.SourceDatabase = "alpha"
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

	// The DLQ is mandatory, and that is checked here rather than discovered mid-stream.
	//
	// A row that terminally fails -- a cross-database denial, a destination that does
	// not exist -- is settled by being recorded somewhere. With no DLQ there is nowhere
	// to record it, so the only honest choices are to hold the offsets (freezing the
	// partition) or advance them (losing the row). Refusing to start turns a deployment
	// mistake into a loud failure at deploy time, which is the cheap moment to find it.
	if cfg.DLQTopic == "" {
		log.Fatalf("DLQ_TOPIC is required: this loader dead-letters terminally-failed rows, " +
			"and without a DLQ those rows would exist in neither StarRocks nor any record")
	}
	if cfg.KafkaBrokers == "" {
		log.Fatalf("KAFKA_BROKERS is required to publish the DLQ")
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
		pool:       newQueryDBPool(maxTenantDBHandles),
		floor:      newOffsetFloor(),
	}

	// Per-tenant routing is opt-in by directory. A loader pointed at a populated
	// credential directory refuses to start on any malformed route rather than
	// silently falling back to the shared database -- a fallback here would put
	// every tenant's rows in one place, which is the failure this work exists to
	// prevent.
	if cfg.TenantRoutesDir != "" {
		routes, err := LoadTenantRoutes(cfg.TenantRoutesDir)
		if err != nil {
			log.Fatalf("Tenant routing is configured but unusable: %v", err)
		}
		if len(routes) == 0 {
			log.Fatalf("STREAM_LOAD_TENANT_ROUTES_DIR=%s contains no usable tenant routes; "+
				"refusing to start rather than writing to the shared database %s",
				cfg.TenantRoutesDir, cfg.StarRocksDB)
		}
		gk.router = &routerCache{router: NewTenantRouter(routes, cfg.SourceDatabase)}
		log.Printf("Tenant routing enabled: %d provisioned routes from %s (source database %q)",
			gk.router.get().Len(), cfg.TenantRoutesDir, cfg.SourceDatabase)
	}

	go gk.startHTTPServer()

	// The topology tripwire watches for the condition that invalidates this loader's
	// tenant_id discriminator. Cheap, slow, and deliberately not an outage alarm.
	if gk.sqlxDB != nil {
		go runTopologyTripwire(ctx, sqlBindingCounter{db: gk.sqlxDB}, cfg, metrics)
	}

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
			"status":                 "UP",
			"topic":                  gk.cfg.Topic,
			"starrocks_table":        gk.cfg.StarRocksTable,
			"assigned_tenant_id":     gk.cfg.AssignedTenantID,
			"validation_enabled":     gk.cfg.ValidationEnabled,
			"tenant_routing_enabled": gk.router.get() != nil && gk.router.get().Enabled(),
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

	var routingEnabled bool
	var routesKnown int
	if r := gk.router.get(); r != nil {
		routingEnabled = r.Enabled()
		routesKnown = r.Len()
	}

	return GatekeeperStats{
		TotalConsumed:          gk.metrics.TotalConsumed.Load(),
		TotalLoaded:            gk.metrics.TotalLoaded.Load(),
		SkippedTombstones:      gk.metrics.SkippedTombstones.Load(),
		SkippedHeartbeats:      gk.metrics.SkippedHeartbeats.Load(),
		SkippedSchemaChanges:   gk.metrics.SkippedSchemaChanges.Load(),
		SkippedUnknown:         gk.metrics.SkippedUnknown.Load(),
		TenantMismatches:       gk.metrics.TenantMismatches.Load(),
		TenantUnattributed:     gk.metrics.TenantUnattributed.Load(),
		TenantUnknown:          gk.metrics.TenantUnknown.Load(),
		RowsRouted:             gk.metrics.RowsRouted.Load(),
		LoadsPerFlush:          gk.metrics.LoadsPerFlush.Load(),
		Flushes:                gk.metrics.Flushes.Load(),
		FatalLoads:             gk.metrics.FatalLoads.Load(),
		RuleViolationsBlocked:  gk.metrics.RuleViolationsBlocked.Load(),
		RuleViolationsWarned:   gk.metrics.RuleViolationsWarned.Load(),
		StreamLoadErrors:       gk.metrics.StreamLoadErrors.Load(),
		DLQEmitted:             gk.metrics.DLQEmitted.Load(),
		DLQUnavailable:         gk.metrics.DLQUnavailable.Load(),
		TopologyTripwireFired:  gk.metrics.TopologyTripwireFired.Load(),
		OffsetFloorHoldSeconds: gk.floor.holds(),
		TotalDeletes:           gk.metrics.TotalDeletes.Load(),
		DeleteErrors:           gk.metrics.DeleteErrors.Load(),
		BatchesFlushed:         gk.metrics.BatchesFlushed.Load(),
		RowsIn:                 gk.metrics.RowsIn.Load(),
		RowsLoaded:             gk.metrics.RowsLoaded.Load(),
		RowsRejected:           gk.metrics.RowsRejected.Load(),
		UptimeSeconds:          time.Since(gk.metrics.StartTime).Seconds(),
		LastEventTime:          lastTime,
		Topic:                  gk.cfg.Topic,
		StarRocksTable:         gk.cfg.StarRocksTable,
		AssignedTenantID:       gk.cfg.AssignedTenantID,
		ValidationEnabled:      gk.cfg.ValidationEnabled,
		TenantRoutingEnabled:   routingEnabled,
		TenantRoutesKnown:      routesKnown,
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
	if gk.router.get() != nil {
		log.Printf("  routing: per-tenant databases under %s (no fallback destination exists)", gk.cfg.StarRocksDB)
	}

	ticker := time.NewTicker(gk.cfg.FlushInterval)
	defer ticker.Stop()

	sets := newBatchSet(gk.cfg.Topic)

	// flush writes every destination in the window and, only on success, commits the
	// offsets it covered. A detached context is used because the final drain runs
	// after ctx is cancelled.
	//
	// Destinations are independent: a tenant whose database was never provisioned
	// must not stop the tenants that are working. Each is resolved to its own
	// outcome, and only an outcome that can still succeed is retried.
	//
	// Offsets are committed per partition up to the lowest unsettled offset, never
	// past it. Several tenants share one partition, and kafka-go commits the highest
	// offset it is handed, so committing a healthy tenant's rows would otherwise step
	// straight over a neighbour's unlanded rows and lose them.
	flush := func(reason string) {
		if sets.empty() {
			return
		}
		// The batch must survive a cancelled parent context, or a graceful shutdown
		// would discard the rows it is holding.
		wctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		gk.metrics.LoadsPerFlush.Store(int64(len(sets.order)))
		gk.metrics.Flushes.Add(1)

		settled := make([]kafka.Message, 0, 64)
		for _, dest := range sets.batches() {
			if gk.flushDestination(wctx, dest, reason) {
				settled = append(settled, dest.messages...)
			} else {
				gk.floor.block(dest.messages)
			}
		}
		if commit := gk.floor.safe(settled); len(commit) > 0 {
			if cerr := r.CommitMessages(wctx, commit...); cerr != nil {
				log.Printf("Failed to commit messages: %v", cerr)
			} else {
				// Commits moved, so any partition we just committed past is no longer
				// stalled. Clearing here is what lets a recovered partition stop alerting
				// instead of reporting its old stall age forever.
				gk.floor.release(commit)
			}
		}
		sets = newBatchSet(gk.cfg.Topic)
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

			if !gk.handleMessage(ctx, sets, m) {
				// Rejected/tombstoned/skipped: needs no StarRocks write, so commit it
				// now -- unless a neighbouring tenant's rows at a lower offset are
				// still replayable, in which case committing would step over them.
				if !gk.floor.blocked(m) {
					if cerr := r.CommitMessages(ctx, m); cerr != nil {
						log.Printf("Failed to commit message: %v", cerr)
					}
				}
				continue
			}

			if sets.pending() >= gk.cfg.MaxRows {
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

// handleMessage returns true when the message contributed an op to some tenant's
// batch (and so must not be committed yet). It returns false when the message was
// consumed and discarded -- skipped, DLQ'd, or loaded immediately -- which the
// caller commits right away.
//
// The order of the two gates below is load-bearing. Classification runs first
// because heartbeats, schema changes and tombstones carry no tenant_id, and reading
// tenant_id first would put the first heartbeat after every quiet period into the
// DLQ. Routing runs next, and a data event with no usable tenant_id is a hard
// failure rather than a default destination.
func (gk *StreamingGatekeeper) handleMessage(ctx context.Context, sets *batchSet, m kafka.Message) bool {
	ev, err := decodeRecord(m.Value)
	if err != nil {
		// Undecodable frames are not evidence of a misroute, so they are counted and
		// skipped rather than DLQ'd.
		gk.metrics.SkippedUnknown.Add(1)
		log.Printf("Error decoding Debezium event: %v", err)
		return false
	}

	switch c := classify(ev, m.Value); c {
	case classData:
		// the only class that is routed
	case classHeartbeat:
		gk.metrics.SkippedHeartbeats.Add(1)
		return false
	case classSchemaChange:
		gk.metrics.SkippedSchemaChanges.Add(1)
		return false
	case classTombstone:
		gk.metrics.SkippedTombstones.Add(1)
		return false
	default:
		gk.metrics.SkippedUnknown.Add(1)
		log.Printf("Skipping unrecognised Debezium envelope on %s offset %d", m.Topic, m.Offset)
		return false
	}

	// Deletes read their key from `before`; upserts from `after`.
	src := ev.after
	if ev.isDelete {
		src = ev.before
	}

	route, ok := gk.route(ctx, src, m)
	if !ok {
		return false
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

	destCfg := route.ConfigFor(gk.cfg)
	key, ok := primaryKeyOf(destCfg.PrimaryKeys, src)
	if !ok {
		// Without a key we can neither upsert (ambiguous) nor delete (no predicate).
		// Send it to the DLQ rather than writing a row we can never update or remove.
		log.Printf("[DLQ] %s event has no usable primary key %v (tenant %s -> %s)",
			ev.op, destCfg.PrimaryKeys, route.TenantID, route.Target(gk.cfg.StarRocksTable))
		gk.emitDLQ(ctx, "ERR_MISSING_PRIMARY_KEY", fmt.Sprintf("no value for primary key %v", destCfg.PrimaryKeys), src, m.Value)
		return false
	}

	dest := sets.get(route)
	dest.ops = append(dest.ops, pendingOp{key: key, row: ev.row, isDelete: ev.isDelete})
	dest.messages = append(dest.messages, m)
	gk.metrics.RowsIn.Add(1)
	if gk.router.get() != nil {
		gk.metrics.RowsRouted.Add(1)
	}
	return true
}

// route resolves the destination for a data event. It returns false when the event
// has already been handled (DLQ'd) and the caller must not batch it.
//
// Fail-closed by construction: there is no default database, because a fallback
// destination is precisely the silent cross-tenant leak this work removes. A tenant
// with no route is DLQ'd whether it was never provisioned (provisioning lag) or was
// deliberately deprovisioned while its events were still in flight -- the same
// outcome either way, so history is preserved and the signal stays visible.
func (gk *StreamingGatekeeper) route(ctx context.Context, src map[string]interface{}, m kafka.Message) (TenantRoute, bool) {
	router := gk.router.get()
	if router == nil || !router.Enabled() {
		// Single-destination mode: the configured database is the only destination,
		// and AssignedTenantID (when set) is the only identity assertion.
		return TenantRoute{
			TenantID:   gk.cfg.AssignedTenantID,
			Database:   gk.cfg.StarRocksDB,
			User:       gk.cfg.StarRocksUser,
			Password:   gk.cfg.StarRocksPassword,
			KeyColumns: gk.cfg.PrimaryKeys,
		}, true
	}

	tenantID := extractTenantID(src)
	if tenantID == "" {
		gk.metrics.TenantUnattributed.Add(1)
		log.Printf("[DLQ] data event on %s offset %d has no tenant_id; failing closed rather than "+
			"guessing a destination", m.Topic, m.Offset)
		gk.emitDLQ(ctx, "ERR_TENANT_UNATTRIBUTED",
			"data event carries no tenant_id; no default database exists", src, m.Value)
		return TenantRoute{}, false
	}

	route, err := router.Route(tenantID)
	if err != nil {
		gk.metrics.TenantUnknown.Add(1)
		log.Printf("[DLQ] no StarRocks route for tenant %s (topic %s offset %d): %v",
			tenantID, m.Topic, m.Offset, err)
		gk.emitDLQ(ctx, "ERR_TENANT_UNKNOWN_ROUTE",
			fmt.Sprintf("tenant %s has no provisioned StarRocks database; either provisioning has not run for it or it was deprovisioned while events were in flight", tenantID),
			src, m.Value)
		return TenantRoute{}, false
	}
	return route, true
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

// flushDestination writes one tenant's pending work and reports whether that
// destination is settled, meaning its offsets may be committed.
//
// The settle decision is the point of this function. A destination is settled when
// the rows are known to be in StarRocks -- or known never to arrive, having gone to
// the DLQ. Only two cases leave it unsettled: a retryable transport failure, and the
// flush window running out while retries were still in flight. Those rows must be
// replayed, so the caller holds their offsets back.
//
// Everything else settles on purpose. A cross-database denial will never succeed on
// retry, so replaying it forever would stall the partition while producing no new
// information; the rows go to the DLQ with the tenant and target attached, which is
// where a human resolves it.
func (gk *StreamingGatekeeper) flushDestination(ctx context.Context, dest *tenantBatch, reason string) bool {
	route := dest.route
	target := route.Target(gk.cfg.StarRocksTable)

	if dest.empty() {
		return true
	}

	for attempt := 1; ; attempt++ {
		err := gk.writeBatch(ctx, dest)
		if err == nil {
			gk.metrics.BatchesFlushed.Add(1)
			return true
		}

		// A fatal load can never succeed as addressed: the tenant's grants do not
		// cover this database, or the database does not exist. DLQ the rows with the
		// tenant and target attached, commit, and move on -- retrying would either
		// hammer the FE or stall every other tenant behind this one.
		if le, fatal := FatalLoadError(err); fatal {
			gk.metrics.FatalLoads.Add(1)
			gk.metrics.StreamLoadErrors.Add(1)
			log.Printf("[DLQ] tenant %s -> %s: %v (reason %s, %d ops)",
				route.TenantID, target, le.err, le.outcome, len(dest.ops))
			if derr := gk.emitDLQ(ctx, fatalReason(le.outcome), le.err.Error(),
				map[string]interface{}{
					"tenant_id":    route.TenantID,
					"target_db":    route.Database,
					"target_table": gk.cfg.StarRocksTable,
					"http_status":  le.status,
					"row_count":    len(dest.ops),
					"routing":      gk.router.get() != nil,
					"source_db":    gk.cfg.SourceDatabase,
				}, nil); derr != nil {
				// The dead-letter is the only durable record of these rows. If it did not
				// land, the batch is not terminal and its offsets must hold the floor.
				log.Printf("[DLQ] tenant %s -> %s: dead-letter not acknowledged, holding offsets: %v",
					route.TenantID, target, derr)
				return false
			}
			return true
		}

		log.Printf("Batch flush failed (%s, tenant %s -> %s, attempt %d, %d ops): %v",
			reason, route.TenantID, target, attempt, len(dest.ops), err)
		gk.metrics.StreamLoadErrors.Add(1)
		if attempt >= maxFlushAttempts {
			// Give up holding the offsets back: the next window will replay these rows
			// under a fresh label, and the DLQ records what was lost rather than
			// letting the topic back up indefinitely.
			log.Printf("[DLQ] tenant %s -> %s: giving up after %d attempts, %d ops replayed later: %v",
				route.TenantID, target, attempt, len(dest.ops), err)
			if derr := gk.emitDLQ(ctx, "ERR_STARROCKS_UNREACHABLE", err.Error(),
				map[string]interface{}{
					"tenant_id":    route.TenantID,
					"target_db":    route.Database,
					"target_table": gk.cfg.StarRocksTable,
					"attempts":     attempt,
					"row_count":    len(dest.ops),
				}, nil); derr != nil {
				// The dead-letter is the only durable record of these rows. If it did not
				// land, the batch is not terminal and its offsets must hold the floor.
				log.Printf("[DLQ] tenant %s -> %s: dead-letter not acknowledged, holding offsets: %v",
					route.TenantID, target, derr)
				return false
			}
			return true
		}
		select {
		case <-ctx.Done():
			// Shutdown, or the flush window's own deadline. The caller must not commit
			// these offsets: the rows have to be replayed, not lost.
			return false
		case <-time.After(backoff(attempt)):
		}
	}
}

// maxFlushAttempts bounds retries of one destination within one flush window. Three
// attempts with exponential backoff covers a transient FE restart without turning a
// 30-second window into a 30-minute stall.
const maxFlushAttempts = 3

// fatalReason names the DLQ cause so an alert distinguishes "routing bug" from
// "credential expired" from "provisioning lag" without re-deriving the response.
func fatalReason(o loadOutcome) string {
	if o == outcomeFatal {
		return "ERR_STARROCKS_DESTINATION_REJECTED"
	}
	return "ERR_STARROCKS_UNREACHABLE"
}

// writeBatch performs one stream load for the upserts and one DELETE for the deletes.
// Upserts run first so that a batch containing both an update and a later delete of
// the same key ends in the deleted state.
func (gk *StreamingGatekeeper) writeBatch(ctx context.Context, b *tenantBatch) error {
	destCfg := b.route.ConfigFor(gk.cfg)

	// Coalesce by key, last operation wins.
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
	// rejected by StarRocks as a duplicate instead of loading the rows twice. StarRocks
	// scopes labels per database, so the same label is safe across tenants, but keeping
	// the offsets in means a re-processed batch after a rebalance stays idempotent at
	// each destination rather than by accident.
	label := sanitizeLabel(gk.cfg.Topic)
	if len(b.messages) > 0 {
		label = fmt.Sprintf("%s_%d_%d", label, b.messages[0].Offset, b.messages[len(b.messages)-1].Offset)
	} else {
		label = fmt.Sprintf("%s_%d", label, time.Now().UnixNano())
	}

	if len(upserts) > 0 {
		rejected, err := gk.loadRows(ctx, destCfg, b.route, upserts, label)
		if err != nil {
			return fmt.Errorf("stream load %d rows to %s: %w",
				len(upserts), b.route.Target(gk.cfg.StarRocksTable), err)
		}
		gk.metrics.TotalLoaded.Add(int64(len(upserts)))
		gk.metrics.RowsLoaded.Add(int64(len(upserts)))
		if rejected > 0 {
			gk.metrics.RowsRejected.Add(rejected)
		}
	}

	if len(deleteKeys) > 0 {
		if err := gk.deleteRows(ctx, destCfg, destCfg.PrimaryKeys, deleteKeys); err != nil {
			return fmt.Errorf("delete %d rows from %s: %w",
				len(deleteKeys), b.route.Target(gk.cfg.StarRocksTable), err)
		}
		gk.metrics.TotalDeletes.Add(int64(len(deleteKeys)))
	}
	return nil
}

// deleteRows issues the keyed DELETE over the MySQL protocol. The connection is
// opened lazily per destination DSN so a loader with no deletes never pays for it,
// and so each tenant's deletes run as that tenant's own principal.
func (gk *StreamingGatekeeper) deleteRows(ctx context.Context, cfg Config, keyCols []string, keys []string) error {
	q, ok := deleteSQL(cfg, keyCols, keys)
	if !ok {
		return nil
	}
	if cfg.QueryDSN == "" {
		return errors.New("no StarRocks query DSN is configured, cannot apply deletes")
	}

	// The pool is bounded: one handle per tenant would otherwise be one MySQL session
	// per tenant on the FE, and the projected tenant count is in the hundreds.
	db, release, err := gk.pool.acquire(cfg.QueryDSN, func(dsn string) (*sql.DB, error) {
		opened, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		opened.SetMaxOpenConns(2)
		opened.SetConnMaxLifetime(5 * time.Minute)
		return opened, nil
	})
	if err != nil {
		return err
	}
	defer release()

	if _, err := db.ExecContext(ctx, q); err != nil {
		return err
	}
	return nil
}

// loadRows loads a batch under strict_mode and, if StarRocks rejects it, bisects to
// isolate the offending rows instead of letting one bad row block the whole batch.
//
// This matters because the failure is silent without strict mode: StarRocks writes
// NULL into any column it cannot convert and still answers Status=Success with
// NumberFilteredRows=0. Verified 3.3.22 — a DECIMAL fed "NOT_A_NUMBER" produced a
// stored NULL, a Success status and zero filtered rows. Only strict_mode surfaces
// it ("too many filtered rows"), so it is the default.
//
// Bisection is applied only to row rejections. An authorisation failure or a
// missing destination is not a bad row, and splitting the batch would turn one
// clear error into log2(n) identical ones.
func (gk *StreamingGatekeeper) loadRows(ctx context.Context, destCfg Config, route TenantRoute, rows []json.RawMessage, label string) (int64, error) {
	rejected, err := streamLoadBatch(ctx, destCfg, route, rows, label)
	if err == nil {
		return rejected, nil
	}
	if !gk.cfg.StrictMode || OutcomeOf(err) != outcomeRowRejected {
		return 0, err
	}
	if len(rows) == 1 {
		// Counted by the caller from the returned total; incrementing here too would
		// double-count every rejected row.
		var row map[string]interface{}
		_ = json.Unmarshal(rows[0], &row)
		gk.emitDLQ(ctx, "ERR_STARROCKS_REJECTED",
			fmt.Sprintf("StarRocks could not store this row: %v", err), row, rows[0])
		log.Printf("[DATA-LOSS] row for tenant %s could not be stored in %s: %v",
			route.TenantID, route.Target(gk.cfg.StarRocksTable), err)
		return 1, nil
	}

	mid := len(rows) / 2
	left, lerr := gk.loadRows(ctx, destCfg, route, rows[:mid], label+"_a")
	right, rerr := gk.loadRows(ctx, destCfg, route, rows[mid:], label+"_b")
	if lerr != nil {
		return left, lerr
	}
	if rerr != nil {
		return right, rerr
	}
	return left + right, nil
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

// emitDLQ publishes a dead-letter record and reports whether it was actually
// acknowledged by the broker.
//
// The return value is load-bearing, not decoration. A dead-letter is what makes a
// failed batch terminal: the rows are in neither StarRocks nor the DLQ, and the only
// remaining copy is in Kafka. If the DLQ write itself fails, the batch is NOT
// terminal and its offsets must stay uncommitted -- otherwise the failure path of the
// safety mechanism reintroduces exactly the loss the offset floor exists to prevent.
//
// A nil writer means no DLQ is configured at all. That is a deployment choice, not a
// transient fault, so it is counted once and does not stall the stream; a configured
// writer that fails is a different thing and does hold the floor.
func (gk *StreamingGatekeeper) emitDLQ(ctx context.Context, reason, detail string, payload map[string]interface{}, raw json.RawMessage) error {
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
		return fmt.Errorf("serialize DLQ message: %w", err)
	}

	if gk.dlqWriter == nil {
		// A terminal outcome with nowhere to record it is not terminal. Returning nil
		// here would let the caller commit offsets for a row that exists in neither
		// StarRocks nor any dead-letter record -- the only path in this loader where a
		// row can vanish without a trace. main() refuses to start without a DLQ, so
		// reaching this is a programming error, and the safe response is to fail.
		gk.metrics.DLQUnavailable.Add(1)
		return errors.New("no DLQ publisher configured: a dead-letter cannot be recorded")
	}
	if err = gk.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(extractTenantID(payload)),
		Value: data,
	}); err != nil {
		log.Printf("Failed to write message to DLQ topic %s: %v", gk.cfg.DLQTopic, err)
		return fmt.Errorf("write to DLQ topic %s: %w", gk.cfg.DLQTopic, err)
	}
	return nil
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
//
// The row counts matter too: StarRocks filters rows it cannot convert and still
// answers Status=Success. Verified on 3.3.22 — a DECIMAL column fed the string
// "NOT_A_NUMBER" loaded as NULL, with no error anywhere. The only signal is
// NumberFilteredRows, which nothing reads unless we read it here.
type starRocksResponse struct {
	Status             string `json:"Status"`
	Msg                string `json:"Message"`
	NumberTotalRows    int64  `json:"NumberTotalRows"`
	NumberLoadedRows   int64  `json:"NumberLoadedRows"`
	NumberFilteredRows int64  `json:"NumberFilteredRows"`
	NumberErrorRows    int64  `json:"NumberErrorRows"`

	// Database and Table are not in the reply: a cross-database denial answers 401
	// with no body at all, so the destination has to come from the request. They are
	// attached at the call site so an alert can name the database that refused the
	// write without the operator re-deriving which tenant it belonged to.
	Database string `json:"-"`
	Table    string `json:"-"`
}

// rejected reports rows StarRocks refused to store, which become NULL columns.
func (r starRocksResponse) rejected() int64 {
	var v int64
	for _, n := range []int64{r.NumberFilteredRows, r.NumberErrorRows} {
		if n > v {
			v = n
		}
	}
	return v
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
//
// It returns the number of rows StarRocks refused to store. Those rows are
// written with NULL in any column it could not convert, so the caller must
// surface the count rather than treating Status=Success as a clean load.
//
// Failures are returned as a *loadError carrying the taxonomy outcome, because
// "retry", "bisect" and "dead-letter" are three different answers to the same
// HTTP response and the caller must not have to infer which it got from a string.
func streamLoadBatch(ctx context.Context, cfg Config, route TenantRoute, rows []json.RawMessage, label string) (int64, error) {
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
		return 0, &loadError{outcome: outcomeTransport, target: cfg.StarRocksDB + "." + cfg.StarRocksTable,
			tenant: route.TenantID, err: err}
	}
	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true")
	req.Header.Set("label", label)
	// strict_mode makes StarRocks reject the whole load rather than filtering the
	// rows it cannot convert. Off by default would mean one bad row silently becomes
	// a row of NULLs with a Success status and no filtered-row count; the loader
	// opts out only when an operator sets STREAM_LOAD_STRICT_MODE=false.
	if cfg.StrictMode {
		req.Header.Set("strict_mode", "true")
	}
	// These tables are PRIMARY KEY tables; the label is the idempotency key and the
	// rows are keyed by the table's own key columns.
	req.Header.Set("partial_update", "false")

	resp, err := clientFor(cfg).Do(req)
	if err != nil {
		// No response at all: connection refused, timeout, DNS. Always retryable,
		// because nothing was decided.
		return 0, &loadError{outcome: outcomeTransport, target: cfg.StarRocksDB + "." + cfg.StarRocksTable,
			tenant: route.TenantID, err: err}
	}
	defer resp.Body.Close()

	var out starRocksResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	out.Database = cfg.StarRocksDB
	out.Table = cfg.StarRocksTable

	outcome, cerr := classifyResponse(resp.StatusCode, out)
	if outcome != outcomeOK {
		return 0, &loadError{
			outcome: outcome,
			status:  resp.StatusCode,
			resp:    out,
			target:  cfg.StarRocksDB + "." + cfg.StarRocksTable,
			tenant:  route.TenantID,
			err:     cerr,
		}
	}
	// A Success that filtered rows is still a partial loss, and the count is the only
	// evidence of it -- see the starRocksResponse doc comment.
	return out.rejected(), nil
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
