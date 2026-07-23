package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type Config struct {
	KafkaBrokers      string
	Topic             string
	StarRocksHTTP     string
	StarRocksUser     string
	StarRocksPassword string
	StarRocksDB       string
	StarRocksTable    string
}

type CDCEvent struct {
	ID        string          `json:"id"`
	Operation string          `json:"op"`
	Table     string          `json:"table"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"ts"`
}

func main() {
	log.Println("Starting Stream Loader Service (Kafka)...")

	config := Config{
		KafkaBrokers:      os.Getenv("KAFKA_BROKERS"),
		Topic:             os.Getenv("CDC_TOPIC"),
		StarRocksHTTP:     os.Getenv("STARROCKS_HTTP"),
		StarRocksUser:     os.Getenv("STARROCKS_USER"),
		StarRocksPassword: os.Getenv("STARROCKS_PASSWORD"),
		StarRocksDB:       os.Getenv("STARROCKS_DB"),
		StarRocksTable:    os.Getenv("STARROCKS_TABLE"),
	}

	if config.Topic == "" {
		config.Topic = "cdc_events"
	}

	if config.KafkaBrokers == "" {
		config.KafkaBrokers = "localhost:9092"
	}

	brokers := []string{config.KafkaBrokers}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "stream-loader-group",
		Topic:    config.Topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Printf("Listening for CDC events on topic %s", config.Topic)

	for {
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Received message: %s", string(m.Value))

		var event CDCEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error decoding JSON: %v", err)
			// commit to skip bad message
			r.CommitMessages(context.Background(), m)
			continue
		}

		// Perform Stream Load
		if err := streamLoad(config, event.Data); err != nil {
			log.Printf("Stream Load failed: %v", err)
			// do not commit so message can be retried; add backoff
			time.Sleep(1 * time.Second)
			continue
		} else {
			if err := r.CommitMessages(context.Background(), m); err != nil {
				log.Printf("Failed to commit message: %v", err)
			}
		}
	}
}

func streamLoad(cfg Config, data json.RawMessage) error {
	url := fmt.Sprintf("%s/api/%s/%s/_stream_load", cfg.StarRocksHTTP, cfg.StarRocksDB, cfg.StarRocksTable)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// In production, parse response body to check "Status": "Success"
	return nil
}
