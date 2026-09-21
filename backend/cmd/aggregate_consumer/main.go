// Aggregate Consumer
//
// Consumes Debezium Postgres CDC events from the alpha-trg / crims-trg
// connectors and runs the same trigger logic as a per-event handler.
// Multiple topics per binary via HANDLERS_TOPICS env var (comma-separated).
//
// Each handler is idempotent: dedupe is keyed on (source_lsn, table, op),
// stored in a per-handler `consumer_dedupe` table in the target DB. A poison
// event on the main topic stalls the partition and freezes downstream
// aggregates, so unhandled failures go to a DLQ topic per connector
// (dlq.alpha_trg, dlq.crims_trg) — operator paged via standard alerting.
//
// Differences from the original triggers:
//   - Transactional protection is gone: triggers fired inside the source
//     transaction; consumers don't. Handlers tolerate at-least-once via
//     dedupe + idempotent UPSERT patterns.
//   - Ordering preserved per (topic, partition) since the consumer group
//     reads partitions in order; do NOT partition these topics by anything
//     other than the source PK.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/handlers"
)

type Config struct {
	KafkaBrokers    string
	DLQTopic        string
	DBURL           string
	ConsumerGroup   string
	Topics          []string
}

type debeziumEnvelope struct {
	Schema  json.RawMessage `json:"schema"`
	Payload struct {
		Before json.RawMessage `json:"before"`
		After  json.RawMessage `json:"after"`
		Op     string          `json:"op"`
		TsMs   int64           `json:"ts_ms"`
		Source struct {
			LSN    string `json:"lsn"`
			Schema string `json:"schema"`
			Table  string `json:"table"`
			TxID   int64  `json:"txId"`
		} `json:"source"`
	} `json:"payload"`
}

type Event = handlers.Event

type Handler interface {
	Topic() string
	Handle(ctx context.Context, evt handlers.Event) error
}

// MultiTopicHandler routes events from multiple source topics to one handler.
// Used by notify_events which reads two CDC topics and re-emits to two agg topics.
type MultiTopicHandler interface {
	Topics() []string
	Handle(ctx context.Context, evt handlers.Event) error
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Starting Aggregate Consumer...")

	cfg := Config{
		KafkaBrokers:  envOr("KAFKA_BROKERS", "uisce-redpanda:9092"),
		DLQTopic:      envOr("DLQ_TOPIC", "dlq.aggregate_consumer"),
		DBURL:         os.Getenv("DATABASE_URL"),
		ConsumerGroup: envOr("KAFKA_GROUP_ID", "aggregate-consumer"),
		Topics:        splitNonEmpty(envOr("HANDLERS_TOPICS", ""), ","),
	}

	if len(cfg.Topics) == 0 {
		log.Fatal("HANDLERS_TOPICS env var required (comma-separated list of topic names)")
	}
	if cfg.DBURL == "" {
		log.Fatal("DATABASE_URL env var required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("postgres connect failed: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping failed: %v", err)
	}

	handlers, multi := buildHandlers(ctx, pool, cfg)
	if len(handlers) == 0 && len(multi) == 0 {
		log.Fatalf("no handlers matched HANDLERS_TOPICS=%v", cfg.Topics)
	}
	for h := range handlers {
		log.Printf("registered single-topic handler for topic=%s", h)
	}
	for h := range multi {
		log.Printf("registered multi-topic handler covering topic=%s", h)
	}

	dlqWriter := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.KafkaBrokers),
		Topic:                  cfg.DLQTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer dlqWriter.Close()

	var wg sync.WaitGroup
	for topic, h := range handlers {
		wg.Add(1)
		go runConsumer(ctx, &wg, cfg, topic, h, dlqWriter)
	}
	for topic, h := range multi {
		wg.Add(1)
		go runMultiConsumer(ctx, &wg, cfg, topic, h, dlqWriter)
	}
	wg.Wait()
	log.Println("Aggregate Consumer stopped")
}

func runConsumer(ctx context.Context, wg *sync.WaitGroup, cfg Config, topic string, h Handler, dlq *kafka.Writer) {
	defer wg.Done()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(cfg.KafkaBrokers, ","),
		GroupID:     cfg.ConsumerGroup + "-" + topic,
		Topic:       topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer r.Close()

	log.Printf("[%s] consuming from topic=%s", h.Topic(), topic)
	consumeLoop(ctx, r, h, dlq)
}

func runMultiConsumer(ctx context.Context, wg *sync.WaitGroup, cfg Config, topic string, h MultiTopicHandler, dlq *kafka.Writer) {
	defer wg.Done()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(cfg.KafkaBrokers, ","),
		GroupID:     cfg.ConsumerGroup + "-" + topic,
		Topic:       topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer r.Close()

	log.Printf("[multi] consuming from topic=%s", topic)
	consumeLoopMulti(ctx, r, h, dlq)
}

func consumeLoop(ctx context.Context, r *kafka.Reader, h Handler, dlq *kafka.Writer) {
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
			log.Printf("[%s] fetch error: %v", h.Topic(), err)
			time.Sleep(time.Second)
			continue
		}

		evt, skip, err := decodeEnvelope(m.Value)
		if err != nil {
			log.Printf("[%s] decode error: %v; offset=%d", h.Topic(), err, m.Offset)
			sendDLQ(ctx, dlq, m.Value, "decode_error", err.Error())
			_ = r.CommitMessages(ctx, m)
			continue
		}
		if skip {
			_ = r.CommitMessages(ctx, m)
			continue
		}

		if err := h.Handle(ctx, evt); err != nil {
			log.Printf("[%s] handler error on table=%s op=%s lsn=%s: %v; offset=%d",
				h.Topic(), evt.Table, evt.Op, evt.LSN, err, m.Offset)
			sendDLQ(ctx, dlq, m.Value, "handler_error", err.Error())
			_ = r.CommitMessages(ctx, m)
			continue
		}
		_ = r.CommitMessages(ctx, m)
	}
}

func consumeLoopMulti(ctx context.Context, r *kafka.Reader, h MultiTopicHandler, dlq *kafka.Writer) {
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
			log.Printf("[multi] fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		evt, skip, err := decodeEnvelope(m.Value)
		if err != nil {
			log.Printf("[multi] decode error: %v; offset=%d", err, m.Offset)
			sendDLQ(ctx, dlq, m.Value, "decode_error", err.Error())
			_ = r.CommitMessages(ctx, m)
			continue
		}
		if skip {
			_ = r.CommitMessages(ctx, m)
			continue
		}

		if err := h.Handle(ctx, evt); err != nil {
			log.Printf("[multi] handler error on table=%s op=%s lsn=%s: %v; offset=%d",
				evt.Table, evt.Op, evt.LSN, err, m.Offset)
			sendDLQ(ctx, dlq, m.Value, "handler_error", err.Error())
			_ = r.CommitMessages(ctx, m)
			continue
		}
		_ = r.CommitMessages(ctx, m)
	}
}

func decodeEnvelope(raw []byte) (handlers.Event, bool, error) {
	var env debeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return handlers.Event{}, false, err
	}
	if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
		// Tombstone or delete — no after row. Consumers handle by topic; if
		// the handler doesn't care, return skip=true to commit and move on.
		return handlers.Event{
			Op:    env.Payload.Op,
			TS:    env.Payload.TsMs,
			Table: env.Payload.Source.Table,
			LSN:   env.Payload.Source.LSN,
		}, true, nil
	}

	var after map[string]any
	if err := json.Unmarshal(env.Payload.After, &after); err != nil {
		return handlers.Event{}, false, err
	}
	var before map[string]any
	if len(env.Payload.Before) > 0 && string(env.Payload.Before) != "null" {
		_ = json.Unmarshal(env.Payload.Before, &before)
	}

	var changed []string
	if before != nil {
		for k, av := range after {
			if bv, ok := before[k]; ok {
				if !handlers.ColEq(av, bv) {
					changed = append(changed, k)
				}
			} else {
				changed = append(changed, k)
			}
		}
		for k := range before {
			if _, ok := after[k]; !ok {
				changed = append(changed, k)
			}
		}
	}

	return handlers.Event{
		Op:             env.Payload.Op,
		TS:             env.Payload.TsMs,
		Table:          env.Payload.Source.Table,
		LSN:            env.Payload.Source.LSN,
		Before:         before,
		After:          after,
		ChangedColumns: changed,
	}, false, nil
}

func sendDLQ(ctx context.Context, w *kafka.Writer, raw []byte, reason, detail string) {
	msg := kafka.Message{
		Key:   []byte(reason),
		Value: raw,
		Headers: []kafka.Header{
			{Key: "x-dlq-reason", Value: []byte(reason)},
			{Key: "x-dlq-detail", Value: []byte(detail)},
			{Key: "x-dlq-time", Value: []byte(time.Now().UTC().Format(time.RFC3339Nano))},
		},
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		log.Printf("DLQ write failed: %v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// dedupeStore is a tiny indirection so handler constructors can accept the
// concrete store type without importing the dedupe package everywhere.
func dedupeStore(pool *pgxpool.Pool) *dedupe.Store { return dedupe.New(pool) }

// buildHandlers wires up the registered handlers for the configured HANDLERS_TOPICS.
// Single-topic handlers go into `single` (one Handler per topic); multi-topic
// handlers go into `multi` (one MultiTopicHandler per topic — each topic gets its
// own consumer goroutine pointing at the same handler).
func buildHandlers(ctx context.Context, pool *pgxpool.Pool, cfg Config) (single map[string]Handler, multi map[string]MultiTopicHandler) {
	single = make(map[string]Handler)
	multi = make(map[string]MultiTopicHandler)

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.KafkaBrokers),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	// Note: writer is intentionally not closed here; the main defer closes it
	// via the dlq writer if they share an underlying transport. In practice
	// the same broker is used; we close the dlq writer only.

	dedupe := dedupeStore(pool)
	if err := dedupe.EnsureSchema(ctx); err != nil {
		log.Fatalf("dedupe schema: %v", err)
	}

	for _, topic := range cfg.Topics {
		switch topic {
		case "alpha_trg.public.template_ratings":
			single[topic] = handlers.NewRatingStats(pool, dedupe)
		case "alpha_trg.public.cube_custom_models":
			single[topic] = handlers.NewCubeCustomModels(pool, dedupe)
		case "alpha_trg.public.cube_security_policies":
			single[topic] = handlers.NewCubeSecurityPolicies(pool, dedupe)
		case "alpha_trg.public.semantic_query_templates":
			single[topic] = handlers.NewTemplateRBAC(pool, dedupe)
		case "alpha_trg.public.investment_opportunities":
			single[topic] = handlers.NewInvestmentScreening(pool, dedupe)
		case "alpha_trg.public.security_user_fund_access",
			"alpha_trg.public.metrics_registry":
			// Multi-topic handler shared by both topics.
			notify := handlers.NewNotifyEvents(pool, writer)
			// Register the same instance under each topic — both consumer
			// goroutines dispatch to the same Handle() method.
			if _, exists := multi[topic]; !exists {
				multi[topic] = notify
			} else {
				multi[topic] = notify
			}
		case "crims_trg.orm.security_identifier":
			single[topic] = handlers.NewIdentifierCache(pool, dedupe)
		default:
			log.Printf("WARN: no handler registered for topic=%s", topic)
		}
	}

	return single, multi
}
