package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func main() {
	brokers := flag.String("brokers", "localhost:9092", "comma-separated broker list")
	topic := flag.String("topic", "", "topic name")
	timeout := flag.Int("timeout", 10, "timeout seconds")
	flag.Parse()

	if *topic == "" {
		log.Fatalf("topic is required")
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(*brokers, ","),
		Topic:       *topic,
		GroupID:     "",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer r.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	m, err := r.FetchMessage(ctx)
	if err != nil {
		log.Fatalf("failed to fetch message: %v", err)
	}

	fmt.Printf("%s %s\n", string(m.Key), string(m.Value))
	// commit not necessary for one-off
}
