package surveillance

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestRedpanda_ConsumerLagBuildAndDrainBackpressure(t *testing.T) {
	brokerAddr := os.Getenv("KAFKA_BROKERS")
	if brokerAddr == "" {
		brokerAddr = "100.84.50.65:9092"
	}

	conn, err := net.DialTimeout("tcp", brokerAddr, 2*time.Second)
	if err != nil {
		t.Skipf("Redpanda at %s not reachable, skipping lag soak test", brokerAddr)
		return
	}
	conn.Close()

	testID := uuid.New().String()[:8]
	topicName := fmt.Sprintf("compliance_lag_test_%s", testID)
	groupID := fmt.Sprintf("compliance_lag_group_%s", testID)

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

	// Create test topic
	kConn, err := dialer.DialContext(context.Background(), "tcp", brokerAddr)
	require.NoError(t, err)
	defer kConn.Close()

	err = kConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)
	defer func() {
		_ = kConn.DeleteTopics(topicName)
	}()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        topicName,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Transport: &kafka.Transport{
			Dial: dialer.DialFunc,
		},
	}
	defer writer.Close()

	totalBurst := 200
	tenantID := uuid.New().String()
	orderID := uuid.New().String()

	t.Logf("Producing %d burst messages to topic %s...", totalBurst, topicName)

	burstMsgs := make([]kafka.Message, totalBurst)
	for i := 0; i < totalBurst; i++ {
		execID := uuid.New().String()
		payload := map[string]interface{}{
			"payload": map[string]interface{}{
				"op": "c",
				"after": map[string]interface{}{
					"id":         execID,
					"tenant_id":  tenantID,
					"order_id":   orderID,
					"exec_qty":   "100.0000",
					"exec_price": "150.2500",
					"status":     "FILLED",
					"exec_time":  time.Now().UTC().Format(time.RFC3339Nano),
				},
				"ts_ms": time.Now().UnixMilli(),
			},
			"seq_num": i,
		}
		valBytes, _ := json.Marshal(payload)
		burstMsgs[i] = kafka.Message{
			Key:   []byte(execID),
			Value: valBytes,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, burstMsgs...)
	require.NoError(t, err)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddr},
		GroupID: groupID,
		Topic:   topicName,
		Dialer:  dialer,
	})
	defer reader.Close()

	var slowThrottle atomic.Bool
	slowThrottle.Store(true)

	var maxObservedLag int64
	var maxStalenessMs int64
	consumedCount := 0

	engine := NewPostTradeSurveillanceEngine()

	// 1. Phase 1: Slow consumption simulating detector backpressure (first 50 messages)
	t.Logf("Simulating slow detector backpressure with throttled consumption...")
	for consumedCount < 50 {
		msg, err := reader.FetchMessage(ctx)
		require.NoError(t, err)

		// Check and export lag metrics
		stats := reader.Stats()
		if stats.Lag > maxObservedLag {
			maxObservedLag = stats.Lag
		}

		// Calculate message staleness delta
		msgTime := msg.Time
		if !msgTime.IsZero() {
			staleness := time.Since(msgTime).Milliseconds()
			if staleness > maxStalenessMs {
				maxStalenessMs = staleness
			}
		}

		time.Sleep(15 * time.Millisecond) // Artificial processing delay
		err = engine.ProcessExecutionCDC(ctx, msg.Value)
		require.NoError(t, err)

		err = reader.CommitMessages(ctx, msg)
		require.NoError(t, err)
		consumedCount++
	}

	t.Logf("Phase 1 Complete: Consumed 50 messages under backpressure. Peak Observed Lag = %d messages, Peak Staleness = %dms",
		maxObservedLag, maxStalenessMs)
	require.Greater(t, maxObservedLag, int64(10), "Lag should build up under throttled processing")

	// 2. Phase 2: Release throttle and fast-drain the remaining backlog
	t.Logf("Phase 2: Releasing throttle; measuring drain velocity to 0 lag...")
	drainStart := time.Now()

	for consumedCount < totalBurst {
		msg, err := reader.FetchMessage(ctx)
		require.NoError(t, err)

		err = engine.ProcessExecutionCDC(ctx, msg.Value)
		require.NoError(t, err)

		err = reader.CommitMessages(ctx, msg)
		require.NoError(t, err)
		consumedCount++
	}

	drainDuration := time.Since(drainStart)
	finalStats := reader.Stats()

	t.Logf("Phase 2 Complete: Drained 150 messages in %v (Throughput: %.2f msg/sec). Final Lag = %d",
		drainDuration, float64(150)/drainDuration.Seconds(), finalStats.Lag)

	require.Equal(t, totalBurst, consumedCount)
	require.Equal(t, int64(totalBurst), engine.Metrics().TotalProcessed.Load())
	require.Equal(t, int64(0), finalStats.Lag, "Final consumer lag must reach 0")

	t.Logf("Consumer Lag & Backpressure Characterization PASSED: Peak Lag=%d, DrainedTo=0, Zero Data Loss!",
		maxObservedLag)
}
