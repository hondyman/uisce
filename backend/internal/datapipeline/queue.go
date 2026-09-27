package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// QueueBroker polls and publishes messages for queue_source / queue_sink.
// Implementations must not log secret values.
type QueueBroker interface {
	Poll(ctx context.Context, cfg QueueSourceConfig) ([]map[string]any, error)
	Publish(ctx context.Context, cfg QueueSinkConfig, rows []map[string]any) (int, error)
}

// EnvQueueBroker resolves Kafka/Redpanda from env, and delegates SQS/Azure
// when those clients are wired (see queue_aws.go / queue_azure.go).
type EnvQueueBroker struct {
	SQS   QueueBroker // optional
	Azure QueueBroker // optional
}

func (b EnvQueueBroker) forBroker(name string) (QueueBroker, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "kafka", "redpanda":
		return kafkaQueueBroker{}, nil
	case "aws_sqs":
		if b.SQS == nil {
			return nil, fmt.Errorf("aws_sqs is not configured in this environment")
		}
		return b.SQS, nil
	case "azure_servicebus":
		if b.Azure == nil {
			return nil, fmt.Errorf("azure_servicebus is not configured in this environment")
		}
		return b.Azure, nil
	default:
		return nil, fmt.Errorf("unknown broker %q", name)
	}
}

func (b EnvQueueBroker) Poll(ctx context.Context, cfg QueueSourceConfig) ([]map[string]any, error) {
	impl, err := b.forBroker(cfg.Broker)
	if err != nil {
		return nil, err
	}
	return impl.Poll(ctx, cfg)
}

func (b EnvQueueBroker) Publish(ctx context.Context, cfg QueueSinkConfig, rows []map[string]any) (int, error) {
	impl, err := b.forBroker(cfg.Broker)
	if err != nil {
		return 0, err
	}
	return impl.Publish(ctx, cfg, rows)
}

type kafkaQueueBroker struct{}

func (kafkaQueueBroker) Poll(ctx context.Context, cfg QueueSourceConfig) ([]map[string]any, error) {
	brokers := resolveBrokers(cfg.BrokersEnv)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers not set (env %s)", firstNonEmpty(cfg.BrokersEnv, "KAFKA_BROKERS"))
	}
	topic := strings.TrimSpace(cfg.TopicOrQueue)
	if topic == "" {
		return nil, fmt.Errorf("topic_or_queue is required")
	}
	group := cfg.ConsumerGroup
	if group == "" {
		group = "uisce-data-pipeline"
	}
	max := cfg.MaxMessages
	if max <= 0 {
		max = 1000
	}
	idle := time.Duration(cfg.IdleTimeoutMS) * time.Millisecond
	if idle <= 0 {
		idle = 3 * time.Second
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        group,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        idle,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer r.Close()

	deadline := time.Now().Add(idle * 3)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	out := make([]map[string]any, 0, max)
	for len(out) < max {
		if time.Now().After(deadline) {
			break
		}
		cctx, cancel := context.WithTimeout(ctx, idle)
		msg, err := r.FetchMessage(cctx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			// idle / timeout → done with this bounded poll
			break
		}
		row, err := decodeQueueJSON(msg.Value)
		if err != nil {
			_ = r.CommitMessages(ctx, msg)
			continue
		}
		out = append(out, row)
		if err := r.CommitMessages(ctx, msg); err != nil {
			return out, fmt.Errorf("commit kafka offset: %w", err)
		}
	}
	return out, nil
}

func (kafkaQueueBroker) Publish(ctx context.Context, cfg QueueSinkConfig, rows []map[string]any) (int, error) {
	brokers := resolveBrokers(cfg.BrokersEnv)
	if len(brokers) == 0 {
		return 0, fmt.Errorf("kafka brokers not set (env %s)", firstNonEmpty(cfg.BrokersEnv, "KAFKA_BROKERS"))
	}
	topic := strings.TrimSpace(cfg.TopicOrQueue)
	if topic == "" {
		return 0, fmt.Errorf("topic_or_queue is required")
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	defer w.Close()

	msgs := make([]kafka.Message, 0, len(rows))
	for _, row := range rows {
		payload, err := json.Marshal(row)
		if err != nil {
			return 0, err
		}
		m := kafka.Message{Value: payload}
		if cfg.KeyField != "" {
			if v, ok := row[cfg.KeyField]; ok && v != nil {
				m.Key = []byte(fmt.Sprint(v))
			}
		}
		msgs = append(msgs, m)
	}
	if err := w.WriteMessages(ctx, msgs...); err != nil {
		return 0, err
	}
	return len(msgs), nil
}

func resolveBrokers(envName string) []string {
	name := firstNonEmpty(envName, "KAFKA_BROKERS")
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func decodeQueueJSON(b []byte) (map[string]any, error) {
	var row map[string]any
	if err := json.Unmarshal(b, &row); err != nil {
		return nil, err
	}
	if row == nil {
		row = map[string]any{}
	}
	return row, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func envOr(name, fallback string) string {
	if name == "" {
		name = fallback
	}
	return strings.TrimSpace(os.Getenv(name))
}

// --- queue_source ----------------------------------------------------------

type queueSource struct {
	cfg    QueueSourceConfig
	broker QueueBroker
}

func newQueueSource(n Node, b QueueBroker) (Source, error) {
	var c QueueSourceConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	c.Broker = strings.ToLower(strings.TrimSpace(c.Broker))
	if c.Format == "" {
		c.Format = "json"
	}
	if b == nil {
		return nil, fmt.Errorf("queue brokers are not configured for this environment")
	}
	return &queueSource{cfg: c, broker: b}, nil
}

func (s *queueSource) Stream(ctx context.Context, rc *RunContext, batchSize int, emit func(rows []Row, rejected []Reject) error) error {
	msgs, err := s.broker.Poll(ctx, s.cfg)
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}
	num := 0
	for i := 0; i < len(msgs); i += batchSize {
		end := i + batchSize
		if end > len(msgs) {
			end = len(msgs)
		}
		batch := make([]Row, 0, end-i)
		for _, m := range msgs[i:end] {
			num++
			batch = append(batch, Row{Num: num, Data: m})
		}
		if err := emit(batch, nil); err != nil {
			return err
		}
	}
	return nil
}

// --- queue_sink ------------------------------------------------------------

type queueSink struct {
	cfg    QueueSinkConfig
	broker QueueBroker
	dryRun bool
}

func newQueueSink(n Node, b QueueBroker) (Processor, error) {
	var c QueueSinkConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	c.Broker = strings.ToLower(strings.TrimSpace(c.Broker))
	if c.Format == "" {
		c.Format = "json"
	}
	if b == nil {
		return nil, fmt.Errorf("queue brokers are not configured for this environment")
	}
	return &queueSink{cfg: c, broker: b}, nil
}

func (p *queueSink) Open(_ context.Context, rc *RunContext) error {
	p.dryRun = p.cfg.DryRun || (rc != nil && rc.DryRun)
	return nil
}
func (p *queueSink) Close(context.Context, error) error { return nil }

func (p *queueSink) Process(ctx context.Context, rows []Row) (Result, error) {
	if len(rows) == 0 {
		return Result{}, nil
	}
	if p.dryRun {
		return Result{Out: rows, Warnings: []Reject{{
			Reason: fmt.Sprintf("queue_sink dry_run: would publish %d messages to %s/%s",
				len(rows), p.cfg.Broker, p.cfg.TopicOrQueue),
		}}}, nil
	}
	payloads := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		payloads = append(payloads, r.Data)
	}
	n, err := p.broker.Publish(ctx, p.cfg, payloads)
	if err != nil {
		return Result{}, err
	}
	_ = n
	return Result{Out: rows}, nil
}
