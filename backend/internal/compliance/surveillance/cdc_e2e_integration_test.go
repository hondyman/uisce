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

	// Fetch valid placement_id, order_id, order_allocation_id, and tenant_id from alpha
	var orderID, placementID, orderAllocID, tenantID uuid.UUID
	err = db.QueryRow("SELECT p.id, p.order_id, oa.id, p.tenant_id FROM orm.placement p JOIN orm.order_allocation oa ON p.order_id = oa.order_id LIMIT 1").Scan(&placementID, &orderID, &orderAllocID, &tenantID)
	if err != nil {
		t.Fatalf("Query placement & order_allocation failed: %v", err)
	}

	engine := NewPostTradeSurveillanceEngine()
	ownerID := uuid.New()
	engine.RegisterAccountOwner(orderID, ownerID)

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

	// 2. Start Kafka Readers on both Debezium topics: 'orm_oms.orm.execution' and 'orm_oms.orm.execution_allocation'
	execTopic := "orm_oms.orm.execution"
	allocTopic := "orm_oms.orm.execution_allocation"

	execReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddr},
		Topic:     execTopic,
		Partition: 0,
		MaxBytes:  10e6,
		Dialer:    dialer,
	})
	defer execReader.Close()
	_ = execReader.SetOffset(kafka.FirstOffset)

	allocReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddr},
		Topic:     allocTopic,
		Partition: 0,
		MaxBytes:  10e6,
		Dialer:    dialer,
	})
	defer allocReader.Close()
	_ = allocReader.SetOffset(kafka.FirstOffset)

	// 3. Insert Execution & Allocation into alpha PostgreSQL to trigger Debezium CDC
	execID := uuid.New()
	allocID := uuid.New()

	t.Logf("Inserting execution %s and allocation %s into alpha...", execID, allocID)

	execQuery := `
		INSERT INTO orm.execution (
			id, placement_id, order_id, exec_qty, exec_price, 
			broker_id, exec_time, transact_time, status, tenant_id
		) VALUES (
			$1, $2, $3, 200.0000, 155.5000,
			'BRK_DEBEZIUM_TEST', NOW(), NOW(), 'FILLED', $4
		)
	`
	_, err = db.Exec(execQuery, execID, placementID, orderID, tenantID)
	if err != nil {
		t.Fatalf("Insert execution failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM orm.execution_allocation WHERE execution_id = $1", execID)
		_, _ = db.Exec("DELETE FROM orm.execution WHERE id = $1", execID)
	}()

	allocQuery := `
		INSERT INTO orm.execution_allocation (
			id, execution_id, order_allocation_id, alloc_exec_qty, alloc_exec_price,
			created_at, tenant_id
		) VALUES (
			$1, $2, $3, 200.0000, 155.5000,
			NOW(), $4
		)
	`
	_, err = db.Exec(allocQuery, allocID, execID, orderAllocID, tenantID)
	if err != nil {
		t.Fatalf("Insert allocation failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 4. Consume Execution and Allocation concurrently from Redpanda
	execCh := make(chan bool, 1)
	allocCh := make(chan bool, 1)
	errCh := make(chan error, 2)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- fmt.Errorf("timeout waiting for execution %s: %w", execID, ctx.Err())
				return
			default:
				msg, err := execReader.ReadMessage(ctx)
				if err != nil {
					errCh <- fmt.Errorf("execReader failed: %w", err)
					return
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
					t.Logf("Captured Debezium CDC event for execution %s! Processing through surveillance...", execID)
					if err := engine.ProcessExecutionCDC(context.Background(), msg.Value); err != nil {
						errCh <- fmt.Errorf("ProcessExecutionCDC failed: %w", err)
						return
					}
					execCh <- true
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- fmt.Errorf("timeout waiting for allocation %s: %w", allocID, ctx.Err())
				return
			default:
				msg, err := allocReader.ReadMessage(ctx)
				if err != nil {
					errCh <- fmt.Errorf("allocReader failed: %w", err)
					return
				}

				var env struct {
					Payload struct {
						After struct {
							ID string `json:"id"`
						} `json:"after"`
					} `json:"payload"`
				}
				_ = json.Unmarshal(msg.Value, &env)

				if env.Payload.After.ID == allocID.String() {
					t.Logf("Captured Debezium CDC event for allocation %s! Processing through surveillance...", allocID)
					if err := engine.ProcessExecutionAllocationCDC(context.Background(), msg.Value); err != nil {
						errCh <- fmt.Errorf("ProcessExecutionAllocationCDC failed: %w", err)
						return
					}
					allocCh <- true
					return
				}
			}
		}
	}()

	matchedExec := false
	matchedAlloc := false

	for !matchedExec || !matchedAlloc {
		select {
		case <-execCh:
			matchedExec = true
		case <-allocCh:
			matchedAlloc = true
		case err := <-errCh:
			t.Fatalf("CDC consumption error: %v", err)
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for CDC events (exec=%v, alloc=%v)", matchedExec, matchedAlloc)
		}
	}

	if engine.Metrics().TotalProcessed.Load() < 2 {
		t.Fatalf("Expected TotalProcessed >= 2 in surveillance metrics, got %d", engine.Metrics().TotalProcessed.Load())
	}

	t.Logf("Both Debezium CDC streams (execution + execution_allocation) verified end-to-end through Surveillance Engine!")
}
