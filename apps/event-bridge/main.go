package main

import (
	"context"
	"log"
	"time"

	"go.temporal.io/sdk/client"
)

type MarketEvent struct {
	TenantID string
	Symbol   string
	Price    float64
}

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	log.Println("Event Bridge started...")

	// Mock event consumption loop
	for {
		// Simulate receiving an event
		event := MarketEvent{
			TenantID: "tenant-1",
			Symbol:   "AAPL",
			Price:    150.0,
		}

		processEvent(c, event)
		time.Sleep(10 * time.Second)
	}
}

func processEvent(c client.Client, event MarketEvent) {
	// In a real scenario, we would look up the relevant workflow ID based on the event
	// For this mock, we assume a fixed workflow ID or lookup logic
	workflowID := "some-running-workflow-id" 

	log.Printf("Processing event for %s: %s at %f", event.TenantID, event.Symbol, event.Price)

	// Signal the workflow
	// We ignore errors here for the mock loop as the workflow might not exist
	_ = c.SignalWorkflow(context.Background(), workflowID, "", "market_signal", event)
}
