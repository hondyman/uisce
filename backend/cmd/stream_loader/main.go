// Stream Loader consumes real Debezium Postgres connector change events
// (the standard Kafka Connect JSON envelope: {"schema":...,"payload":{
// "before":..,"after":..,"op":..,"ts_ms":..}}) and stream-loads the "after"
// row into a StarRocks table via the HTTP stream load API. One instance
// handles one topic -> one table (CDC_TOPIC / STARROCKS_TABLE env vars).
//
// Debezium encodes numeric(p,s) columns as base64-encoded big-endian
// two's-complement integers (org.apache.kafka.connect.data.Decimal) rather
// than plain JSON numbers; decodeRecord below resolves each field's scale
// from the embedded value schema and converts those back to real numbers
// before the row is forwarded to StarRocks.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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

type debeziumEnvelope struct {
	Schema  json.RawMessage `json:"schema"`
	Payload struct {
		Before json.RawMessage `json:"before"`
		After  json.RawMessage `json:"after"`
		Op     string          `json:"op"`
		TsMs   int64           `json:"ts_ms"`
	} `json:"payload"`
}

type schemaField struct {
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Field      string            `json:"field"`
	Parameters map[string]string `json:"parameters"`
}

type valueSchema struct {
	Fields []struct {
		Type   string        `json:"type"`
		Field  string        `json:"field"`
		Fields []schemaField `json:"fields"`
	} `json:"fields"`
}

func main() {
	log.Println("Starting Stream Loader Service (Kafka -> StarRocks)...")

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
	if config.StarRocksUser == "" {
		config.StarRocksUser = "root"
	}

	brokers := []string{config.KafkaBrokers}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "stream-loader-" + config.Topic,
		Topic:    config.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	log.Printf("Listening for CDC events on topic %s -> %s.%s", config.Topic, config.StarRocksDB, config.StarRocksTable)

	for {
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		row, skip, err := decodeRecord(m.Value)
		if err != nil {
			log.Printf("Error decoding Debezium event: %v", err)
			r.CommitMessages(context.Background(), m)
			continue
		}
		if skip {
			// Tombstone or delete event (no "after" row) - nothing to load.
			r.CommitMessages(context.Background(), m)
			continue
		}

		if err := streamLoad(config, row); err != nil {
			log.Printf("Stream Load failed: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if err := r.CommitMessages(context.Background(), m); err != nil {
			log.Printf("Failed to commit message: %v", err)
		}
	}
}

// decodeRecord parses a Debezium change-event envelope and returns the
// "after" row as flat JSON, with numeric(p,s) fields converted from
// Debezium's base64 Decimal encoding to plain JSON numbers.
func decodeRecord(raw []byte) (json.RawMessage, bool, error) {
	var env debeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, false, err
	}
	if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
		return nil, true, nil
	}

	var after map[string]interface{}
	if err := json.Unmarshal(env.Payload.After, &after); err != nil {
		return nil, false, err
	}

	if len(env.Schema) > 0 {
		var vs valueSchema
		if err := json.Unmarshal(env.Schema, &vs); err == nil {
			for _, f := range vs.Fields {
				if f.Field != "after" {
					continue
				}
				for _, cf := range f.Fields {
					if cf.Name != "org.apache.kafka.connect.data.Decimal" {
						continue
					}
					raw, ok := after[cf.Field]
					if !ok || raw == nil {
						continue
					}
					b64, ok := raw.(string)
					if !ok {
						continue
					}
					decoded, err := decodeDebeziumDecimal(b64, cf.Parameters["scale"])
					if err == nil {
						after[cf.Field] = decoded
					}
				}
			}
		}
	}

	rowJSON, err := json.Marshal(after)
	if err != nil {
		return nil, false, err
	}
	return rowJSON, false, nil
}

// decodeDebeziumDecimal converts Debezium's base64-encoded big-endian
// two's-complement Decimal representation into a decimal string.
func decodeDebeziumDecimal(b64, scaleStr string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	i := new(big.Int).SetBytes(raw)
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		// Negative: two's complement.
		full := new(big.Int).Lsh(big.NewInt(1), uint(len(raw)*8))
		i.Sub(i, full)
	}
	scale := 0
	fmt.Sscanf(scaleStr, "%d", &scale)
	f := new(big.Float).SetInt(i)
	if scale > 0 {
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
		f.Quo(f, divisor)
	}
	return f.Text('f', scale), nil
}

func streamLoad(cfg Config, row json.RawMessage) error {
	url := fmt.Sprintf("%s/api/%s/%s/_stream_load", cfg.StarRocksHTTP, cfg.StarRocksDB, cfg.StarRocksTable)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(row))
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "false")

	// StarRocks stream load answers with a 307 redirect from the FE to the
	// owning BE node. Go's default redirect policy strips Authorization on
	// cross-host redirects, which breaks stream load auth - preserve it.
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	return nil
}
