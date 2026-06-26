//go:build legacy_amqp
// +build legacy_amqp

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	temporalclientlib "github.com/hondyman/semlayer/libs/temporal-client"
	st "github.com/streadway/amqp"
	sdkclient "go.temporal.io/sdk/client"

	"github.com/hondyman/semlayer/backend/internal/workflows"
	workerpkg "github.com/hondyman/semlayer/backend/temporal/worker"
)

func main() {
	var (
		amqpURL = getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	)
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

	// Prepare AMQP consumer to observe the published event
	conn, err := st.Dial(amqpURL)
	if err != nil {
		log.Fatalf("failed to dial rabbit: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("events", "topic", true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}
	routingKey := "test.workflow"
	if err := ch.QueueBind(q.Name, routingKey, "events", false, nil); err != nil {
		log.Fatalf("failed to bind queue: %v", err)
	}
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	// Start the workflow which runs the activity that publishes to RabbitMQ
	wid := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	opts := sdkclient.StartWorkflowOptions{ID: wid, TaskQueue: "e2e_test_queue"}

	payload := map[string]interface{}{"wid": wid}
	we, err := tc.ExecuteWorkflow(context.Background(), opts, workflows.TestWorkflow, amqpURL, routingKey, payload)
	if err != nil {
		log.Fatalf("failed to execute workflow: %v", err)
	}
	log.Printf("workflow started: %s", we.GetID())

	// Wait for message published by activity
	select {
	case d := <-msgs:
		var got map[string]interface{}
		if err := json.Unmarshal(d.Body, &got); err != nil {
			log.Fatalf("failed to unmarshal body: %v", err)
		}
		if got["wid"] == wid {
			log.Println("E2E PASS: workflow produced event")
			os.Exit(0)
		}
		log.Fatalf("unexpected payload: %v", got)
	case <-time.After(15 * time.Second):
		log.Fatalf("timed out waiting for event")
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
