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

	// Try brokers one by one to find a responsive broker to query partitions
	var conn *kafka.Conn
	var err error
	for _, b := range strings.Split(*brokers, ",") {
		conn, err = kafka.DialContext(context.Background(), "tcp", b)
		if err == nil {
			defer conn.Close()
			break
		}
		log.Printf("failed to dial broker %s: %v", b, err)
	}
	if conn == nil {
		log.Fatalf("failed to connect to any broker from %s", *brokers)
	}

	parts, err := conn.ReadPartitions(*topic)
	if err != nil {
		log.Fatalf("failed to read partitions for %s: %v", *topic, err)
	}

	if len(parts) == 0 {
		log.Fatalf("no partitions found for topic %s", *topic)
	}

	// Pick partition 0 by default (most topics use partition 0 in test)
	partition := parts[0]
	leader := fmt.Sprintf("%s:%d", partition.Leader, 9092)
	log.Printf("Found partition leader: %s (partition %d)", leader, partition.ID)

	// Dial leader
	leaderConn, err := kafka.DialLeader(context.Background(), "tcp", leader, *topic, partition.ID)
	if err != nil {
		log.Fatalf("failed to dial leader %s: %v", leader, err)
	}
	defer leaderConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	// Set read deadline
	leaderConn.SetReadDeadline(time.Now().Add(time.Duration(*timeout) * time.Second))

	var b []byte
	n, err := leaderConn.Read(b)
	if err == nil && n > 0 {
		fmt.Println(string(b[:n]))
		return
	}

	// Use ReadMessage helper via kafka.Reader on the leader broker
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   strings.Split(*brokers, ","),
		Topic:     *topic,
		Partition: int(partition.ID),
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	defer r.Close()

	m, err := r.FetchMessage(ctx)
	if err != nil {
		log.Fatalf("failed to fetch message from leader: %v", err)
	}

	fmt.Printf("%s %s\n", string(m.Key), string(m.Value))
}
