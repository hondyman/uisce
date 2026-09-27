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

	bad, _ := json.Marshal(QueueSourceConfig{Broker: "rabbitmq", TopicOrQueue: "x"})
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
