package surveillance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

func TestPostTradeSurveillance_LiveDebeziumCDCAndDetectorE2E(t *testing.T) {
	brokerAddr := os.Getenv("KAFKA_BROKERS")
	if brokerAddr == "" {
		brokerAddr = "100.84.50.65:9092"
	}

	conn, err := net.DialTimeout("tcp", brokerAddr, 2*time.Second)
	if err != nil {
		t.Skipf("Redpanda at %s not reachable, skipping live CDC integration test", brokerAddr)
		return
	}
	conn.Close()

	// 1. Connect to PostgreSQL alpha
	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres alpha not reachable, skipping: %v", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("Postgres alpha ping failed, skipping: %v", err)
		return
	}

	// Fetch valid placement_id, order_id, and tenant_id from alpha
	var orderID, placementID, tenantID uuid.UUID
	err = db.QueryRow("SELECT id, order_id, tenant_id FROM orm.placement LIMIT 1").Scan(&placementID, &orderID, &tenantID)
	if err != nil {
		t.Fatalf("Query placement failed: %v", err)
	}

	engine := NewPostTradeSurveillanceEngine()
	ownerID := uuid.New()
	engine.RegisterAccountOwner(orderID, ownerID)

	// 2. Start Kafka Consumer on Debezium topic 'orm_oms.orm.execution'
	topic := "orm_oms.orm.execution"
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

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddr},
		Topic:     topic,
		Partition: 0,
		MaxBytes:  10e6,
		Dialer:    dialer,
	})
	defer reader.Close()

	// Seek to end so we only consume new event
	_ = reader.SetOffset(kafka.LastOffset)

	// 3. Insert new execution into alpha.orm.execution to trigger Debezium CDC
	execID := uuid.New()
	t.Logf("Inserting execution %s into alpha.orm.execution for Debezium capture...", execID)

	query := `
		INSERT INTO orm.execution (
			id, placement_id, order_id, exec_qty, exec_price, 
			broker_id, exec_time, transact_time, status, tenant_id
		) VALUES (
			$1, $2, $3, 150.0000, 152.7500,
			'BRK_DEBEZIUM_TEST', NOW(), NOW(), 'FILLED', $4
		)
	`
	_, err = db.Exec(query, execID, placementID, orderID, tenantID)
	if err != nil {
		t.Fatalf("Insert execution failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM orm.execution WHERE id = $1", execID)
	}()

	// 4. Consume Debezium event from Redpanda
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	matched := false
	t.Logf("Waiting for Debezium CDC change event on topic %s...", topic)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("Failed to read Debezium message from Redpanda: %v", err)
		}

		var env struct {
			Payload struct {
				After struct {
					ID string `json:"id"`
				} `json:"after"`
			} `json:"payload"`
		}
		_ = json.Unmarshal(msg.Value, &env)

		if env.Payload.After.ID == execID.String() {
			t.Logf("Captured Debezium CDC event for execution %s! Processing through surveillance engine...", execID)
			if err := engine.ProcessExecutionCDC(context.Background(), msg.Value); err != nil {
				t.Fatalf("ProcessExecutionCDC failed: %v", err)
			}
			matched = true
			break
		}
	}

	if !matched {
		t.Fatalf("Did not capture Debezium event for execution %s", execID)
	}

	if engine.Metrics().TotalProcessed.Load() < 1 {
		t.Fatalf("Expected TotalProcessed >= 1 in surveillance metrics")
	}

	t.Logf("End-to-End Debezium CDC -> Redpanda -> Surveillance Engine Verified Successfully!")
}
