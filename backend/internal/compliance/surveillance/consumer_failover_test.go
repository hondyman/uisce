package surveillance

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type failoverMessagePayload struct {
	SeqNum   int    `json:"seq_num"`
	ExecID   string `json:"exec_id"`
	TenantID string `json:"tenant_id"`
	OrderID  string `json:"order_id"`
	Qty      string `json:"qty"`
	Price    string `json:"price"`
	Time     string `json:"time"`
}

func TestRedpanda_ConsumerFailoverAndBackpressureSoak(t *testing.T) {
	brokerAddr := os.Getenv("KAFKA_BROKERS")
	if brokerAddr == "" {
		brokerAddr = "100.84.50.65:9092"
	}

	conn, err := net.DialTimeout("tcp", brokerAddr, 2*time.Second)
	if err != nil {
		t.Skipf("Redpanda at %s not reachable, skipping consumer failover test", brokerAddr)
		return
	}
	conn.Close()

	testID := uuid.New().String()[:8]
	topicName := fmt.Sprintf("compliance_failover_test_%s", testID)
	groupID := fmt.Sprintf("compliance_failover_group_%s", testID)

	dialer := &kafka.Dialer{
		Timeout: 10 * time.Second,
		DialFunc: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err == nil && (host == "uisce-redpanda" || host == "redpanda") && brokerAddr != "" {
				brokerHost, _, _ := net.SplitHostPort(brokerAddr)
				if brokerHost != "" {
					address = net.JoinHostPort(brokerHost, port)
				}
			}
			d := &net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, network, address)
		},
	}

	// Explicitly create test topic
	kConn, err := dialer.DialContext(context.Background(), "tcp", brokerAddr)
	require.NoError(t, err)
	defer kConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}
	err = kConn.CreateTopics(topicConfig)
	require.NoError(t, err, "Create topic failed")
	defer func() {
		_ = kConn.DeleteTopics(topicName)
	}()

	// 1. Produce 500 messages to the dedicated test topic
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        topicName,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll, // acks=all
		Transport: &kafka.Transport{
			Dial: dialer.DialFunc,
		},
	}
	defer writer.Close()

	totalMessages := 500
	failoverPoint := 250
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	t.Logf("Producing %d messages to topic %s with acks=all...", totalMessages, topicName)

	kafkaMsgs := make([]kafka.Message, totalMessages)
	for i := 0; i < totalMessages; i++ {
		execID := uuid.New().String()
		payload := map[string]interface{}{
			"payload": map[string]interface{}{
				"op": "c",
				"after": map[string]interface{}{
					"id":          execID,
					"tenant_id":   tenantID,
					"order_id":    orderID,
					"exec_qty":    "100.0000",
					"exec_price":  "150.2500",
					"status":      "FILLED",
					"exec_time":   time.Now().UTC().Format(time.RFC3339Nano),
				},
				"ts_ms": time.Now().UnixMilli(),
			},
			"seq_num": i,
		}
		valBytes, _ := json.Marshal(payload)
		kafkaMsgs[i] = kafka.Message{
			Key:   []byte(execID),
			Value: valBytes,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, kafkaMsgs...)
	require.NoError(t, err, "Producing 500 messages to Redpanda must succeed")
	t.Logf("Successfully produced %d messages to %s", totalMessages, topicName)

	// 2. Start Consumer A on groupID
	engine := NewPostTradeSurveillanceEngine()
	var mu sync.Mutex
	receivedSeqNums := make(map[int]bool)

	consumerA := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddr},
		GroupID: groupID,
		Topic:   topicName,
		Dialer:  dialer,
	})

	t.Logf("Starting Consumer A on group %s, consuming up to %d messages...", groupID, failoverPoint)

	for i := 0; i < failoverPoint; i++ {
		msg, err := consumerA.FetchMessage(ctx)
		require.NoError(t, err, "Consumer A failed to fetch message")

		var env struct {
			SeqNum int `json:"seq_num"`
		}
		_ = json.Unmarshal(msg.Value, &env)

		mu.Lock()
		receivedSeqNums[env.SeqNum] = true
		mu.Unlock()

		err = engine.ProcessExecutionCDC(ctx, msg.Value)
		require.NoError(t, err)

		err = consumerA.CommitMessages(ctx, msg)
		require.NoError(t, err, "Consumer A commit failed")
	}

	mu.Lock()
	consumerACount := len(receivedSeqNums)
	mu.Unlock()
	require.Equal(t, failoverPoint, consumerACount, "Consumer A should have consumed exactly 250 messages")
	t.Logf("Consumer A successfully processed and committed %d messages. Simulating node crash/failover...", consumerACount)

	// 3. Kill / Close Consumer A
	err = consumerA.Close()
	require.NoError(t, err)

	// 4. Start Consumer B on the exact same groupID to simulate failover recovery
	t.Logf("Starting Consumer B on group %s to resume consumption...", groupID)

	consumerB := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddr},
		GroupID: groupID,
		Topic:   topicName,
		Dialer:  dialer,
	})
	defer consumerB.Close()

	remainingExpected := totalMessages - failoverPoint
	for i := 0; i < remainingExpected; i++ {
		msg, err := consumerB.FetchMessage(ctx)
		require.NoError(t, err, "Consumer B failed to fetch message after failover")

		var env struct {
			SeqNum int `json:"seq_num"`
		}
		_ = json.Unmarshal(msg.Value, &env)

		mu.Lock()
		receivedSeqNums[env.SeqNum] = true
		mu.Unlock()

		err = engine.ProcessExecutionCDC(ctx, msg.Value)
		require.NoError(t, err)

		err = consumerB.CommitMessages(ctx, msg)
		require.NoError(t, err, "Consumer B commit failed")
	}

	// 5. Assert zero-loss and exact sequence completion
	mu.Lock()
	totalReceived := len(receivedSeqNums)
	mu.Unlock()

	require.Equal(t, totalMessages, totalReceived, "Must receive exactly 500 unique messages across failover")
	for seq := 0; seq < totalMessages; seq++ {
		require.True(t, receivedSeqNums[seq], "Message seq %d must be delivered and processed", seq)
	}

	require.Equal(t, int64(totalMessages), engine.Metrics().TotalProcessed.Load(), "Engine must have processed exactly 500 events")

	t.Logf("Consumer Failover Soak Test Passed: Total=%d, FailoverAt=%d, ResumedTo=%d, Zero Loss Verified!",
		totalMessages, failoverPoint, totalReceived)
}
