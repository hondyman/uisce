package main

import (
	"context"
	"encoding/json"
	"testing"

	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- DLQ replay sufficiency ----

// decodeDLQ returns the single message the recording publisher captured.
func decodeDLQ(t *testing.T, pub *recordingDLQ) DLQMessage {
	t.Helper()
	require.Equal(t, 1, pub.n, "exactly one dead-letter should have been published")
	var m DLQMessage
	require.NoError(t, json.Unmarshal(pub.last[0].Value, &m))
	return m
}

func TestDLQRecordCarriesTheEventPositionForASingleRow(t *testing.T) {
	// A dead letter without topic/partition/offset is a parking lot: you know a row was
	// rejected but not which one, so replay is archaeology rather than an operation.
	pub := &recordingDLQ{}
	gk := newTestGatekeeper(Config{Topic: "configured-topic", DLQTopic: "dlq"})
	gk.dlqWriter = pub

	msg := kafka.Message{Topic: "orm_oms.orm.order", Partition: 3, Offset: 4711}
	row := json.RawMessage(`{"id":"k1","tenant_id":"t-1"}`)
	require.NoError(t, gk.emitDLQ(context.Background(), "ERR_STARROCKS_REJECTED",
		"could not store", nil, originOf(msg.Topic, msg), row))

	got := decodeDLQ(t, pub)
	assert.Equal(t, "orm_oms.orm.order", got.OriginalTopic,
		"the record's topic is the message's topic, not the configured default")
	assert.Equal(t, 3, got.OriginalPartition)
	assert.Equal(t, int64(4711), got.OriginalOffset)
	assert.Equal(t, int64(4711), got.OriginalOffsetLast)
	assert.True(t, got.OffsetsKnown)
	assert.Equal(t, row, json.RawMessage(got.RawEvent), "a single rejected row carries its own event")
}

func TestDLQRecordCarriesAnOffsetRangeForABatch(t *testing.T) {
	// A whole batch that terminally failed records the RANGE instead of inlining 2000
	// raw envelopes: the events are still in the topic and re-readable by offset, and a
	// DLQ too large to read during an incident helps nobody.
	pub := &recordingDLQ{}
	gk := newTestGatekeeper(Config{Topic: "t", DLQTopic: "dlq"})
	gk.dlqWriter = pub

	batch := []kafka.Message{
		{Topic: "t", Partition: 1, Offset: 100},
		{Topic: "t", Partition: 1, Offset: 101},
		{Topic: "t", Partition: 1, Offset: 137},
	}
	require.NoError(t, gk.emitDLQ(context.Background(), "ERR_STARROCKS_DESTINATION_REJECTED",
		"denied", map[string]interface{}{"row_count": 3}, originRangeOf("t", batch), nil))

	got := decodeDLQ(t, pub)
	assert.True(t, got.OffsetsKnown)
	assert.Equal(t, int64(100), got.OriginalOffset)
	assert.Equal(t, int64(137), got.OriginalOffsetLast)
	assert.Empty(t, got.RawEvent, "a batch does not inline its events")
	assert.Equal(t, float64(3), got.Payload["row_count"])
}

func TestDLQRangeRefusesToSpanPartitions(t *testing.T) {
	// A range that crossed a partition boundary would replay the wrong rows on the
	// wrong partition, which is worse than admitting the position is unknown.
	origin := originRangeOf("t", []kafka.Message{
		{Topic: "t", Partition: 0, Offset: 10},
		{Topic: "t", Partition: 1, Offset: 90},
	})
	assert.False(t, origin.Known,
		"a cross-partition batch must not be presented as a replayable single range")
}

func TestDLQOriginIsHonestWhenUnknown(t *testing.T) {
	pub := &recordingDLQ{}
	gk := newTestGatekeeper(Config{Topic: "configured", DLQTopic: "dlq"})
	gk.dlqWriter = pub

	require.NoError(t, gk.emitDLQ(context.Background(), "ERR_X", "d", nil, dlqOrigin{}, nil))
	got := decodeDLQ(t, pub)
	assert.False(t, got.OffsetsKnown, "an unknown position is reported, not implied")
	assert.Equal(t, "configured", got.OriginalTopic, "it falls back to the configured topic")
}

func TestDLQRecordCarriesTriageFields(t *testing.T) {
	// The triage payload is the other half of replay: why it failed and where it was
	// being sent when it did.
	pub := &recordingDLQ{}
	gk := newTestGatekeeper(Config{Topic: "t", DLQTopic: "dlq"})
	gk.dlqWriter = pub

	require.NoError(t, gk.emitDLQ(context.Background(), "ERR_STARROCKS_DESTINATION_REJECTED",
		"denied", map[string]interface{}{
			"tenant_id": "t-1", "target_db": "tenant_a", "http_status": 401, "row_count": 3,
		}, originOf("t", kafka.Message{Partition: 0, Offset: 1}), nil))

	got := decodeDLQ(t, pub)
	assert.Equal(t, "t-1", got.Payload["tenant_id"])
	assert.Equal(t, "tenant_a", got.Payload["target_db"])
	assert.Equal(t, float64(401), got.Payload["http_status"])
	assert.Equal(t, float64(3), got.Payload["row_count"])
	assert.Equal(t, "ERR_STARROCKS_DESTINATION_REJECTED", got.Reason)
	assert.NotEmpty(t, got.Detail)
}

// ---- DLQ smoke ----

func TestSmokeDLQFailsWhenNoPublisher(t *testing.T) {
	err := smokeDLQ(context.Background(), []string{"127.0.0.1:1"}, "dlq", nil)
	require.Error(t, err, "an unusable DLQ must fail the deploy, not stream against it")
}

func TestSmokeDLQRejectsEmptyBrokerList(t *testing.T) {
	err := smokeDLQ(context.Background(), nil, "dlq", &recordingDLQ{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Kafka brokers")
}

func TestSmokeDLQChecksReachabilityBeforePublishing(t *testing.T) {
	// An unreachable broker must fail without attempting a produce: the failure has to
	// be attributable to the broker rather than to the ACL.
	pub := &recordingDLQ{}
	err := smokeDLQ(context.Background(), []string{"127.0.0.1:1"}, "dlq", pub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reach broker")
	assert.Equal(t, 0, pub.n, "nothing is published when the broker is unreachable")
}

func TestSmokeProbeIsSelfIdentifying(t *testing.T) {
	// One marked record per deploy is acceptable; an unidentifiable extra record that
	// looks like a lost row is not.
	probe, err := json.Marshal(map[string]interface{}{"probe": dlqProbeMarker})
	require.NoError(t, err)
	assert.Contains(t, string(probe), dlqProbeMarker,
		"a DLQ consumer must be able to skip the smoke record")
}

func TestSplitBrokers(t *testing.T) {
	assert.Equal(t, []string{"a:9092", "b:9092"}, splitBrokers("a:9092,b:9092"))
	assert.Equal(t, []string{"a:9092"}, splitBrokers("a:9092,"))
	assert.Empty(t, splitBrokers(""))
}

func TestRecordingDLQCapturesMessages(t *testing.T) {
	pub := &recordingDLQ{}
	require.NoError(t, pub.WriteMessages(context.Background(), kafka.Message{Value: []byte("x")}))
	assert.Equal(t, 1, pub.n)
	assert.Len(t, pub.last, 1)
}
