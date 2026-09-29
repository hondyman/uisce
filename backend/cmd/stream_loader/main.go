// Stream Loader consumes real Debezium Postgres connector change events
// (the standard Kafka Connect JSON envelope: {"schema":...,"payload":{
// "before":..,"after":..,"op":..,"ts_ms":..}}) and acts as the Streaming Gatekeeper:
//
// 1. Decodes Debezium decimal & payload fields
// 2. Asserts Layer 2 Tenant Identity (assigned tenant vs event.tenant_id)
// 3. Evaluates in-line active Validation Rules (RuleSnapshot / EmbeddedEngine)
// 4. Routes blocked/invalid records to Dead Letter Queue (DLQ topic + violation table)
// 5. Stream-loads verified records into StarRocks via HTTP stream load API
// 6. Exposes real-time HTTP metrics and health monitoring (/health, /metrics, /stats)
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
	StartTime             time.Time
	lastEventUnixNano     atomic.Int64
}

type GatekeeperStats struct {
	TotalConsumed         int64     `json:"total_consumed"`
	TotalLoaded           int64     `json:"total_loaded"`
	SkippedTombstones     int64     `json:"skipped_tombstones"`
	TenantMismatches      int64     `json:"tenant_mismatches"`
	RuleViolationsBlocked int64     `json:"rule_violations_blocked"`
	RuleViolationsWarned  int64     `json:"rule_violations_warned"`
	StreamLoadErrors      int64     `json:"stream_load_errors"`
	DLQEmitted            int64     `json:"dlq_emitted"`
	UptimeSeconds         float64   `json:"uptime_seconds"`
	LastEventTime         *time.Time `json:"last_event_time,omitempty"`
	Topic                 string    `json:"topic"`
	StarRocksTable        string    `json:"starrocks_table"`
	AssignedTenantID      string    `json:"assigned_tenant_id,omitempty"`
	ValidationEnabled     bool      `json:"validation_enabled"`
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
	return cfg
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
	log.Printf("Starting Streaming Gatekeeper Service [Topic=%s -> StarRocks=%s.%s]...", cfg.Topic, cfg.StarRocksDB, cfg.StarRocksTable)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := &GatekeeperMetrics{
		StartTime: time.Now(),
	}

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

	// Start Telemetry / Health HTTP Server
	go gk.startHTTPServer()

	// Handle graceful shutdown
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
		stats := gk.GetStats()
		json.NewEncoder(w).Encode(stats)
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := gk.GetStats()
		json.NewEncoder(w).Encode(stats)
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
		UptimeSeconds:         time.Since(gk.metrics.StartTime).Seconds(),
		LastEventTime:         lastTime,
		Topic:                 gk.cfg.Topic,
		StarRocksTable:        gk.cfg.StarRocksTable,
		AssignedTenantID:      gk.cfg.AssignedTenantID,
		ValidationEnabled:     gk.cfg.ValidationEnabled,
	}
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

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error fetching message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		gk.metrics.TotalConsumed.Add(1)
		gk.metrics.lastEventUnixNano.Store(time.Now().UnixNano())

		rowJSON, afterMap, skip, err := decodeRecord(m.Value)
		if err != nil {
			log.Printf("Error decoding Debezium event: %v", err)
			_ = r.CommitMessages(ctx, m)
			continue
		}
		if skip {
			gk.metrics.SkippedTombstones.Add(1)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		// Layer 2: Assert Tenant Identity
		eventTenant := extractTenantID(afterMap)
		if gk.cfg.AssignedTenantID != "" && eventTenant != "" && eventTenant != gk.cfg.AssignedTenantID {
			gk.metrics.TenantMismatches.Add(1)
			log.Printf("[SECURITY] Tenant Mismatch rejected: event tenant=%s != assigned tenant=%s", eventTenant, gk.cfg.AssignedTenantID)
			gk.emitDLQ(ctx, "ERR_TENANT_MISMATCH", fmt.Sprintf("Event tenant %s does not match assigned %s", eventTenant, gk.cfg.AssignedTenantID), afterMap, m.Value)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		// Layer 3: Embedded Validation Rule Engine
		if gk.cfg.ValidationEnabled && gk.ruleEngine != nil {
			boKey := gk.cfg.BusinessObject
			tenantForEval := gk.cfg.AssignedTenantID
			if tenantForEval == "" {
				tenantForEval = eventTenant
			}

			evalResult, evalErr := gk.ruleEngine.ValidateRecord(ctx, boKey, "", "", afterMap)
			if evalErr != nil {
				log.Printf("Validation evaluation error: %v", evalErr)
			} else if evalResult != nil {
				if evalResult.Blocked {
					gk.metrics.RuleViolationsBlocked.Add(1)
					log.Printf("[BLOCK] Validation Rule violation blocked stream load on %s: %d violations", boKey, len(evalResult.Violations))
					gk.persistViolations(ctx, tenantForEval, boKey, evalResult, true)
					gk.emitDLQ(ctx, "ERR_RULE_BLOCK_VIOLATION", fmt.Sprintf("Rule violations: %d blocking rules failed", len(evalResult.Violations)), afterMap, m.Value)
					_ = r.CommitMessages(ctx, m)
					continue
				} else if len(evalResult.Violations) > 0 {
					gk.metrics.RuleViolationsWarned.Add(1)
					gk.persistViolations(ctx, tenantForEval, boKey, evalResult, false)
				}
			}
		}

		// StarRocks Stream Load
		if err := streamLoad(gk.cfg, rowJSON); err != nil {
			gk.metrics.StreamLoadErrors.Add(1)
			log.Printf("Stream Load failed: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		gk.metrics.TotalLoaded.Add(1)
		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("Failed to commit message: %v", err)
		}
	}
}

func extractTenantID(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	if v, ok := m["tenant_id"]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	if v, ok := m["tenantId"]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	if v, ok := m["TenantID"]; ok && v != nil {
		return fmt.Sprintf("%v", v)
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

// decodeRecord parses a Debezium change-event envelope and returns the
// "after" row as flat JSON, the map representation, skip flag, and any error.
func decodeRecord(raw []byte) (json.RawMessage, map[string]interface{}, bool, error) {
	var env debeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, false, err
	}
	if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
		return nil, nil, true, nil
	}

	var after map[string]interface{}
	if err := json.Unmarshal(env.Payload.After, &after); err != nil {
		return nil, nil, false, err
	}

	if len(env.Schema) > 0 {
		var vs valueSchema
		if err := json.Unmarshal(env.Schema, &vs); err == nil {
			for _, f := range vs.Fields {
				if f.Field != "after" {
					continue
				}
				for _, cf := range f.Fields {
					if cf.Name != "org.apache.kafka.connect.data.Decimal" {
						continue
					}
					rawVal, ok := after[cf.Field]
					if !ok || rawVal == nil {
						continue
					}
					b64, ok := rawVal.(string)
					if !ok {
						continue
					}
					decoded, err := decodeDebeziumDecimal(b64, cf.Parameters["scale"])
					if err == nil {
						after[cf.Field] = decoded
					}
				}
			}
		}
	}

	rowJSON, err := json.Marshal(after)
	if err != nil {
		return nil, nil, false, err
	}
	return rowJSON, after, false, nil
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

func streamLoad(cfg Config, row json.RawMessage) error {
	url := fmt.Sprintf("%s/api/%s/%s/_stream_load", cfg.StarRocksHTTP, cfg.StarRocksDB, cfg.StarRocksTable)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(row))
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "false")

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	return nil
}
