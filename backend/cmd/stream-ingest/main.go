// 014_stream_ingest_skeleton.go
// Skeleton for the stream-ingest service.
// Consumes PriceTick from Redpanda, writes to streaming.price_tick.
//
// RLS: every pool connection runs
//
//	SELECT set_config('app.current_tenant', $1, false)
//
// so FORCE ROW LEVEL SECURITY on streaming.* evaluates against the dev
// tenant where mdm.source_systems rows live.
//
// Build:  go build -o bin/stream-ingest ./cmd/stream-ingest
// Run:    ./bin/stream-ingest --topic price.security.official.eod \
//                             --group stream-ingest-factset-eod \
//                             --source-id <uuid> \
//                             --tenant-id 99e99e99-99e9-49e9-89e9-99e99e99e999
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers     []string
	Topic       string
	Group       string
	SourceSysID string
	TenantID    string
	DSN         string
}

type PriceTick struct {
	SourceSystemCd  string            `json:"source_system_cd"`
	EntityType      string            `json:"entity_type"`
	EntityRef       string            `json:"entity_ref"`
	PriceTypeCd     string            `json:"price_type_cd"`
	ObservationType string            `json:"observation_type"`
	EventTimeMs     int64             `json:"event_time_ms"`
	Value           *float64          `json:"value,omitempty"`
	Bid             *float64          `json:"bid,omitempty"`
	Ask             *float64          `json:"ask,omitempty"`
	Mid             *float64          `json:"mid,omitempty"`
	Last            *float64          `json:"last,omitempty"`
	Volume          *float64          `json:"volume,omitempty"`
	Currency        string            `json:"currency,omitempty"`
	QualityTierCd   string            `json:"quality_tier_cd,omitempty"`
	IsCorrection    bool              `json:"is_correction,omitempty"`
	CorrectsTickID  string            `json:"corrects_tick_id,omitempty"`
	Attrs           map[string]string `json:"attrs,omitempty"`
}

func main() {
	var cfg Config
	var brokers string
	flag.StringVar(&brokers, "brokers", "localhost:9092", "redpanda brokers")
	flag.StringVar(&cfg.Topic, "topic", "", "topic to consume")
	flag.StringVar(&cfg.Group, "group", "", "consumer group")
	flag.StringVar(&cfg.SourceSysID, "source-id", "", "source_systems.uuid for this feed")
	flag.StringVar(&cfg.TenantID, "tenant-id",
		"99e99e99-99e9-49e9-89e9-99e99e99e999", "tenant uuid (dev default — where source_systems live)")
	flag.StringVar(&cfg.DSN, "dsn", os.Getenv("CRIMS_DSN"), "crims dsn")
	flag.Parse()

	if cfg.Topic == "" || cfg.Group == "" || cfg.SourceSysID == "" || cfg.DSN == "" {
		fmt.Fprintln(os.Stderr, "required: --topic --group --source-id --dsn (or CRIMS_DSN)")
		os.Exit(2)
	}
	cfg.Brokers = []string{brokers}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		log.Error("parse dsn", "err", err)
		os.Exit(1)
	}
	// Session-scoped GUC so FORCE RLS write/read policies match tenant_id.
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx,
			`SELECT set_config('app.current_tenant', $1, false)`, cfg.TenantID)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Error("db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Error("db ping", "err", err)
		os.Exit(1)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.Group,
		Topic:    cfg.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Info("stream-ingest started",
		"topic", cfg.Topic, "group", cfg.Group,
		"brokers", cfg.Brokers, "tenant", cfg.TenantID,
		"source_id", cfg.SourceSysID)

	for {
		if ctx.Err() != nil {
			return
		}
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error("fetch", "err", err)
			time.Sleep(time.Second)
			continue
		}
		if err := handleRecord(ctx, pool, log, &cfg, &m); err != nil {
			log.Error("handle", "offset", m.Offset, "partition", m.Partition, "err", err)
			continue
		}
		if err := r.CommitMessages(ctx, m); err != nil {
			log.Error("commit", "err", err)
		}
	}
}

func handleRecord(ctx context.Context, pool *pgxpool.Pool,
	log *slog.Logger, cfg *Config, m *kafka.Message) error {

	var tick PriceTick
	if err := json.Unmarshal(m.Value, &tick); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if tick.EventTimeMs == 0 {
		tick.EventTimeMs = m.Time.UnixMilli()
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO streaming.price_tick (
			kafka_topic, kafka_partition, kafka_offset, kafka_timestamp,
			source_system_id, price_entity_type, source_entity_ref,
			price_type_cd, observation_type,
			event_time, value, bid, ask, mid, last, volume, currency,
			quality_tier_cd, is_correction, corrects_tick_id,
			raw_payload, tenant_id
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			to_timestamp($10::bigint / 1000.0), $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20,
			$21, $22
		)
		ON CONFLICT DO NOTHING`,
		m.Topic, m.Partition, m.Offset, m.Time,
		cfg.SourceSysID, tick.EntityType, tick.EntityRef,
		tick.PriceTypeCd, tick.ObservationType,
		tick.EventTimeMs, tick.Value, tick.Bid, tick.Ask, tick.Mid,
		tick.Last, tick.Volume, tick.Currency,
		tick.QualityTierCd, tick.IsCorrection, nullIfEmpty(tick.CorrectsTickID),
		m.Value, cfg.TenantID,
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO streaming.watermark
			(source_system_id, topic, partition, high_watermark,
			 last_seen_at, status, tenant_id)
		VALUES ($1, $2, $3, $4, now(), 'HEALTHY', $5)
		ON CONFLICT (tenant_id, source_system_id, topic, partition)
		DO UPDATE SET
			high_watermark = GREATEST(streaming.watermark.high_watermark,
			                           EXCLUDED.high_watermark),
			last_seen_at = now()`,
		cfg.SourceSysID, m.Topic, m.Partition,
		time.UnixMilli(tick.EventTimeMs), cfg.TenantID,
	)
	if err != nil {
		return fmt.Errorf("watermark: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
