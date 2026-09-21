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
//
// Delete handling: op=d events carry a `before` row but no `after`. The
// loader emits a StarRocks stream-load delete via the `__op` column on a
// Primary Key model table. Each loader must declare its PK column name
// (PRIMARY_KEY_COLUMN env var, default "id").
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
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
	PrimaryKeyColumn  string
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

// decodeResult conveys the action to take on StarRocks for one CDC event.
// skip=true means nothing to do (tombstone or empty before-image for delete).
type decodeResult struct {
	op      string          // "u" upsert, "d" delete
	skip    bool
	payload json.RawMessage // the row body for stream load
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
		PrimaryKeyColumn:  envOr("PRIMARY_KEY_COLUMN", "id"),
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

	log.Printf("Listening for CDC events on topic %s -> %s.%s (pk col: %s)",
		config.Topic, config.StarRocksDB, config.StarRocksTable, config.PrimaryKeyColumn)

	for {
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		res, err := decodeRecord(m.Value)
		if err != nil {
			log.Printf("Error decoding Debezium event: %v", err)
			r.CommitMessages(context.Background(), m)
			continue
		}
		if res.skip {
			r.CommitMessages(context.Background(), m)
			continue
		}

		if err := streamLoad(config, res); err != nil {
			log.Printf("Stream Load failed: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if err := r.CommitMessages(context.Background(), m); err != nil {
			log.Printf("Failed to commit message: %v", err)
		}
	}
}

// decodeRecord parses a Debezium change-event envelope, decodes any
// numeric(p,s) base64-Debezium-Decimal columns, and returns the action to
// take on StarRocks: either an upsert payload (after row) or a delete
// payload (PK from before row) or skip (tombstone).
func decodeRecord(raw []byte) (decodeResult, error) {
	res := decodeResult{}
	var env debeziumEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return res, err
	}

	op := env.Payload.Op
	switch op {
	case "c", "u", "r":
		// insert / update / snapshot read — needs an after row
		if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
			res.skip = true
			return res, nil
		}
		var after map[string]interface{}
		if err := json.Unmarshal(env.Payload.After, &after); err != nil {
			return res, err
		}
		decodeDecimals(env.Schema, after)
		rowJSON, err := json.Marshal(after)
		if err != nil {
			return res, err
		}
		res.op = "u"
		res.payload = rowJSON
		return res, nil

	case "d":
		// delete — needs the before row's PK value at minimum
		if len(env.Payload.Before) == 0 || string(env.Payload.Before) == "null" {
			// no before-image (tombstone); can't construct delete — skip
			res.skip = true
			return res, nil
		}
		var before map[string]interface{}
		if err := json.Unmarshal(env.Payload.Before, &before); err != nil {
			return res, err
		}
		rowJSON, err := json.Marshal(before)
		if err != nil {
			return res, err
		}
		res.op = "d"
		res.payload = rowJSON
		return res, nil
	}

	// unknown op — skip rather than fail
	log.Printf("unknown op=%q in CDC event; skipping", op)
	res.skip = true
	return res, nil
}

// decodeDecimals converts Debezium base64-encoded Decimals in `after` to
// plain JSON numbers, using the embedded value-schema for scale info.
//
// Also decodes Kafka Connect's logical-type wire formats produced by
// connectors configured with time.precision.mode=connect:
//   - Timestamp: int64 unix millis -> ISO 8601 string
//   - Date:      int32 days since 1970-01-01 -> "YYYY-MM-DD"
//   - Time:      int32 ms past midnight     -> "HH:MM:SS"
//   - Duration:  12-byte struct (3×int32: months, days, millis), big-endian,
//                base64-encoded in the JSON value. Decoded to integer
//                microseconds for BIGINT storage.
//
// Decoding failures leave the original value intact (so it lands as NULL
// in StarRocks via the standard "null on type mismatch" behavior, rather
// than emitting garbage that would corrupt downstream analytics).
func decodeDecimals(schemaRaw json.RawMessage, after map[string]interface{}) {
	if len(schemaRaw) == 0 {
		return
	}
	var vs valueSchema
	if err := json.Unmarshal(schemaRaw, &vs); err != nil {
		return
	}
	for _, f := range vs.Fields {
		if f.Field != "after" {
			continue
		}
		for _, cf := range f.Fields {
			raw, ok := after[cf.Field]
			if !ok || raw == nil {
				continue
			}
			switch cf.Name {
			case "org.apache.kafka.connect.data.Decimal":
				b64, ok := raw.(string)
				if !ok {
					continue
				}
				decoded, err := decodeDebeziumDecimal(b64, cf.Parameters["scale"])
				if err == nil {
					after[cf.Field] = decoded
				}
			case "org.apache.kafka.connect.data.Timestamp":
				if n, ok := raw.(float64); ok {
					after[cf.Field] = formatUnixMillisToISO(int64(n))
				}
			case "org.apache.kafka.connect.data.Date":
				if n, ok := raw.(float64); ok {
					after[cf.Field] = formatDaysSinceEpochToDate(int64(n))
				}
			case "org.apache.kafka.connect.data.Time":
				if n, ok := raw.(float64); ok {
					after[cf.Field] = formatMillisPastMidnight(int64(n))
				}
			case "org.apache.kafka.connect.data.Duration":
				// Wire format (Kafka Connect Duration logical type with
				// time.precision.mode=connect): 3×int32 big-endian packed
				// into 12 bytes (months, days, millis), base64-encoded in
				// the JSON value field. Decoder fails soft on length
				// mismatch — duration lands NULL in StarRocks, which the
				// null-validator then surfaces for investigation.
				if decoded, ok := decodeDurationB64(raw); ok {
					after[cf.Field] = decoded
				}
			}
		}
	}
}

// decodeDurationB64 parses the Connect Duration wire format from a JSON
// string field containing base64-encoded bytes. Returns (microseconds, true)
// on success or (0, false) on any decode failure so the caller can leave
// the value untouched.
func decodeDurationB64(raw interface{}) (int64, bool) {
	b64, ok := raw.(string)
	if !ok {
		return 0, false
	}
	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(bytes) != 12 {
		return 0, false
	}
	months := int32(binary.BigEndian.Uint32(bytes[0:4]))
	days := int32(binary.BigEndian.Uint32(bytes[4:8]))
	millis := int32(binary.BigEndian.Uint32(bytes[8:12]))
	return durationMicros(int64(months), int64(days), int64(millis)), true
}

// packDurationB64 is the inverse of decodeDurationB64 — used in tests to
// construct round-trip vectors from declared (months, days, millis) values.
// Exposed at package level so tests can pin wire-format symmetry.
func packDurationB64(months, days, millis int32) string {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], uint32(months))
	binary.BigEndian.PutUint32(buf[4:8], uint32(days))
	binary.BigEndian.PutUint32(buf[8:12], uint32(millis))
	return base64.StdEncoding.EncodeToString(buf)
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	}
	return 0
}

func formatUnixMillisToISO(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func formatDaysSinceEpochToDate(days int64) string {
	return time.Unix(0, 0).AddDate(0, 0, int(days)).UTC().Format("2006-01-02")
}

func formatMillisPastMidnight(ms int64) string {
	h := ms / 3600000
	m := (ms % 3600000) / 60000
	s := (ms % 60000) / 1000
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// durationMicros renders a Postgres interval (months, days, milliseconds)
// as an integer number of microseconds, suitable for storage in a BIGINT
// column. Months are approximated as 30 days; nearly all CDC durations are
// sub-day so the loss is bounded. Negative components propagate correctly.
func durationMicros(months, days, millis int64) int64 {
	const millisPerMonth = 30 * 24 * 3600 * 1000
	totalMillis := months*millisPerMonth + days*int64(24*3600*1000) + millis
	return totalMillis * 1000
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

func streamLoad(cfg Config, res decodeResult) error {
	url := fmt.Sprintf("%s/api/%s/%s/_stream_load", cfg.StarRocksHTTP, cfg.StarRocksDB, cfg.StarRocksTable)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(res.payload))
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "false")

	if res.op == "d" {
		// StarRocks Primary Key model: stream load with the `__op` column set
		// to 1 deletes the row identified by PK. The `columns` header
		// re-projects the body so we only emit `__op` and the PK column.
		req.Header.Set("columns", "__op=1,"+cfg.PrimaryKeyColumn)
	}

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

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

