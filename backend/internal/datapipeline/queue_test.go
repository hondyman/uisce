package datapipeline

import (
	"context"
	"encoding/json"
	"testing"
)

func TestQueueSourceConfigValidate(t *testing.T) {
	good, _ := json.Marshal(QueueSourceConfig{
		Broker: "redpanda", TopicOrQueue: "account.ingest.v1", Format: "json",
	})
	n := Node{ID: "q", Type: NodeQueueSource, Config: good}
	if errs := validateNodeConfig(&n); len(errs) != 0 {
		t.Fatalf("expected valid: %v", errs)
	}

	bad, _ := json.Marshal(QueueSourceConfig{Broker: "unknown-broker", TopicOrQueue: "x"})
	n.Config = bad
	if errs := validateNodeConfig(&n); len(errs) == 0 {
		t.Fatal("expected invalid broker")
	}
}

func TestQueueSinkInSpec(t *testing.T) {
	src, _ := json.Marshal(QueueSourceConfig{Broker: "kafka", TopicOrQueue: "in"})
	sink, _ := json.Marshal(QueueSinkConfig{Broker: "kafka", TopicOrQueue: "out"})
	spec := Spec{
		Version: SpecVersion,
		Nodes: []Node{
			{ID: "in", Type: NodeQueueSource, Config: src},
			{ID: "out", Type: NodeQueueSink, Config: sink},
		},
		Edges: []Edge{{From: "in", To: "out"}},
	}
	if errs := spec.Validate(); len(errs) != 0 {
		t.Fatalf("spec invalid: %v", errs)
	}
}

type memQueue struct {
	published []map[string]any
	poll      []map[string]any
}

func (m *memQueue) Poll(context.Context, QueueSourceConfig) ([]map[string]any, error) {
	return m.poll, nil
}
func (m *memQueue) Publish(_ context.Context, _ QueueSinkConfig, rows []map[string]any) (int, error) {
	m.published = append(m.published, rows...)
	return len(rows), nil
}

func TestQueueSinkProcess(t *testing.T) {
	cfg, _ := json.Marshal(QueueSinkConfig{Broker: "kafka", TopicOrQueue: "t"})
	mem := &memQueue{}
	p, err := newQueueSink(Node{ID: "o", Type: NodeQueueSink, Config: cfg}, mem)
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Open(context.Background(), &RunContext{TenantID: "t", RunID: "r"})
	res, err := p.Process(context.Background(), []Row{
		{Num: 1, Data: map[string]any{"account_cd": "A1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Out) != 1 || len(mem.published) != 1 {
		t.Fatalf("out=%d published=%d", len(res.Out), len(mem.published))
	}
}

func TestQueueSourceStream(t *testing.T) {
	cfg, _ := json.Marshal(QueueSourceConfig{Broker: "kafka", TopicOrQueue: "t"})
	mem := &memQueue{poll: []map[string]any{{"account_cd": "A1"}, {"account_cd": "A2"}}}
	src, err := newQueueSource(Node{ID: "i", Type: NodeQueueSource, Config: cfg}, mem)
	if err != nil {
		t.Fatal(err)
	}
	var got []Row
	err = src.Stream(context.Background(), &RunContext{}, 10, func(rows []Row, _ []Reject) error {
		got = append(got, rows...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
}

func TestQueueEnvNamesAreAllowListed(t *testing.T) {
	for _, ok := range []string{"KAFKA_BROKERS", "REDPANDA_BROKERS_EU", "AWS_SQS_QUEUE_URL", "AZURE_SERVICEBUS_CONNECTION_STRING", "DATAPIPELINE_QUEUE_X"} {
		if !queueEnvAllowed(ok) {
			t.Errorf("%s should be allowed", ok)
		}
	}
	for _, bad := range []string{"JWT_SECRET", "DATABASE_URL", "kafka_brokers", "KAFKA", "AWS_SECRET_ACCESS_KEY", "KAFKA_BROKERS; X", ""} {
		if queueEnvAllowed(bad) {
			t.Errorf("%q must not be allowed", bad)
		}
	}
}

func TestLookupQueueEnvRefusesOtherVariables(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("KAFKA_BROKERS", "broker-1:9092")
	if got := lookupQueueEnv("JWT_SECRET"); got != "" {
		t.Fatalf("read a non-queue variable: %q", got)
	}
	if got := resolveBrokers("JWT_SECRET"); len(got) != 0 {
		t.Fatalf("resolveBrokers read a non-queue variable: %v", got)
	}
	if got := lookupQueueEnv("KAFKA_BROKERS"); got != "broker-1:9092" {
		t.Fatalf("allowed variable not read: %q", got)
	}
}

func TestQueueNodesRejectNonQueueEnvNames(t *testing.T) {
	for _, tc := range []struct {
		typ string
		cfg any
	}{
		{NodeQueueSource, QueueSourceConfig{Broker: "kafka", TopicOrQueue: "t", BrokersEnv: "JWT_SECRET"}},
		{NodeQueueSource, QueueSourceConfig{Broker: "aws_sqs", QueueURLEnv: "DATABASE_URL"}},
		{NodeQueueSink, QueueSinkConfig{Broker: "azure_servicebus", TopicOrQueue: "q", ConnectionStringEnv: "JWT_SECRET"}},
	} {
		raw, _ := json.Marshal(tc.cfg)
		n := Node{ID: "q", Type: tc.typ, Config: raw}
		if errs := validateNodeConfig(&n); len(errs) == 0 {
			t.Errorf("%s with %s should be rejected", tc.typ, raw)
		}
	}
	ok, _ := json.Marshal(QueueSourceConfig{Broker: "kafka", TopicOrQueue: "t", BrokersEnv: "KAFKA_BROKERS_EU"})
	n := Node{ID: "q", Type: NodeQueueSource, Config: ok}
	if errs := validateNodeConfig(&n); len(errs) != 0 {
		t.Fatalf("allow-listed name rejected: %v", errs)
	}
}
