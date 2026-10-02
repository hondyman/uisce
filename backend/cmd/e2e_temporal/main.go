// Command e2e_temporal is an end-to-end check that a Temporal workflow can run
// an activity that publishes an event to Kafka/Redpanda: it starts a worker,
// starts workflows.TestWorkflow, and consumes the "events" topic until the
// published message arrives. Run it through scripts/e2e_temporal.sh.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	temporalclientlib "github.com/hondyman/uisce/libs/temporal-client"
	"github.com/segmentio/kafka-go"
	sdkclient "go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/workflows"
	workerpkg "github.com/hondyman/uisce/backend/temporal/worker"
)

const eventsTopic = "events" // the topic workflows.PublishEventActivity writes to

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	flag.Parse()

	// Start Temporal client (centralized helper with retries)
	tc, err := temporalclientlib.NewClientWithRetry()
	if err != nil {
		log.Fatalf("failed to create temporal client: %v", err)
	}
	defer tc.Close()

	// Start the worker in background using the dedicated worker package
	go func() {
		if err := workerpkg.Start(tc); err != nil {
			log.Fatalf("worker returned error: %v", err)
		}
	}()

	if err := ensureTopic(brokers, eventsTopic); err != nil {
		log.Fatalf("failed to ensure topic %q: %v", eventsTopic, err)
	}

	// Observe the topic from the end, before the workflow can publish.
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(brokers, ","),
		Topic:       eventsTopic,
		Partition:   0,
		StartOffset: kafka.LastOffset,
		MaxWait:     500 * time.Millisecond,
	})
	defer reader.Close()
	if err := reader.SetOffset(kafka.LastOffset); err != nil {
		log.Fatalf("failed to position reader: %v", err)
	}

	wid := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	routingKey := "test.workflow"
	opts := sdkclient.StartWorkflowOptions{ID: wid, TaskQueue: "e2e_test_queue"}
	payload := map[string]interface{}{"wid": wid}
	we, err := tc.ExecuteWorkflow(context.Background(), opts, workflows.TestWorkflow, brokers, routingKey, payload)
	if err != nil {
		log.Fatalf("failed to execute workflow: %v", err)
	}
	log.Printf("workflow started: %s", we.GetID())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Fatalf("timed out waiting for the published event: %v", err)
		}
		if string(m.Key) != routingKey {
			continue
		}
		var got map[string]interface{}
		if err := json.Unmarshal(m.Value, &got); err != nil {
			log.Fatalf("failed to unmarshal body: %v", err)
		}
		if got["wid"] == wid {
			log.Println("E2E PASS: workflow produced event")
			os.Exit(0)
		}
		// Another run's message on the shared topic: keep waiting for ours.
	}
}

func ensureTopic(brokers, topic string) error {
	conn, err := kafka.Dial("tcp", strings.Split(brokers, ",")[0])
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.CreateTopics(kafka.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1})
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
