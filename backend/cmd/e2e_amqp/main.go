//go:build legacy_amqp
// +build legacy_amqp

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	api "github.com/hondyman/semlayer/backend/internal/api"
	st "github.com/streadway/amqp"
)

func main() {
	// NOTE: This E2E AMQP test is legacy. Prefer using `scripts/redpanda_smoke_test.sh` for Kafka/Redpanda checks.
	if kb := os.Getenv("KAFKA_BROKERS"); kb != "" {
		log.Printf("KAFKA_BROKERS is set (%s) — consider using scripts/redpanda_smoke_test.sh instead of e2e_amqp", kb)
	}

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	// Create a consumer connection to verify message arrives
	consumerConn, err := st.Dial(amqpURL)
	if err != nil {
		log.Fatalf("failed to dial rabbit for consumer: %v", err)
	}
	defer consumerConn.Close()

	consumerCh, err := consumerConn.Channel()
	if err != nil {
		log.Fatalf("failed to open channel for consumer: %v", err)
	}
	defer consumerCh.Close()

	if err := consumerCh.ExchangeDeclare("events", "topic", true, false, false, false, nil); err != nil {
		log.Fatalf("failed to declare events exchange: %v", err)
	}

	q, err := consumerCh.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		log.Fatalf("failed to declare temp queue: %v", err)
	}

	routingKey := "e2e.script.key"
	if err := consumerCh.QueueBind(q.Name, routingKey, "events", false, nil); err != nil {
		log.Fatalf("failed to bind queue: %v", err)
	}

	msgs, err := consumerCh.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	// Create the AMQPEventBus and publish
	bus, err := api.NewAMQPEventBus(amqpURL, "events")
	if err != nil {
		log.Fatalf("failed to create AMQPEventBus: %v", err)
	}
	defer bus.Close()

	payload := map[string]interface{}{"msg": "hello e2e", "ts": time.Now().UnixNano()}
	if err := bus.Emit(context.Background(), routingKey, payload); err != nil {
		log.Fatalf("failed to emit event: %v", err)
	}

	// wait for message arrival
	select {
	case d := <-msgs:
		var got map[string]interface{}
		if err := json.Unmarshal(d.Body, &got); err != nil {
			log.Fatalf("failed to unmarshal received body: %v", err)
		}
		fmt.Printf("received payload: %+v\n", got)
	case <-time.After(15 * time.Second):
		log.Fatalf("timed out waiting for message")
	}

	fmt.Println("E2E AMQP check succeeded")
}
