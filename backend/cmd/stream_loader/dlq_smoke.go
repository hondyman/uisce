package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// DLQ smoke check.
//
// Requiring DLQ_TOPIC at boot closes the "no DLQ at all" case, but the topic can still
// be present in configuration and unusable in practice: the topic may not exist, or the
// principal may not be authorized to produce to it. That is the same silent-loss path
// one step later -- the produce fails mid-stream, the floor holds, the partition stalls,
// and the first sign of trouble is an incident rather than a deploy.
//
// So the check runs at boot, where the container refuses to start and the failure is
// cheap and loud. Two parts, because they catch different faults:
//
//	metadata   -- the topic exists. Catches a typo or a topic that was never created.
//	produce     -- this principal may write to it. Only a real produce proves
//	               authorization; a metadata read can succeed for a topic the caller is
//	               not allowed to produce to.
//
// The probe leaves one marked record in the DLQ topic per deploy. That is deliberate:
// a consumer must be able to tolerate it, and it is self-identifying rather than
// looking like a lost row. It is also the honest signal that the check ran.

// dlqProbeMarker identifies the smoke record. Anything reading the DLQ should skip it.
const dlqProbeMarker = "uisce-dlq-smoke-probe"

// dlqSmokeTimeout bounds the boot check so a wedged broker fails the deploy quickly
// instead of hanging it.
const dlqSmokeTimeout = 15 * time.Second

// smokeDLQ verifies the dead-letter topic is usable before the loader accepts traffic.
//
// It is fatal by design: a loader that cannot record a terminal outcome cannot honour
// at-least-once, so continuing would trade a failed deploy for a stalled partition.
func smokeDLQ(ctx context.Context, brokers []string, topic string, pub dlqPublisher) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no Kafka brokers configured for the DLQ")
	}

	checkCtx, cancel := context.WithTimeout(ctx, dlqSmokeTimeout)
	defer cancel()

	conn, err := kafka.DialContext(checkCtx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("DLQ smoke: cannot reach broker %s: %w", brokers[0], err)
	}
	defer conn.Close()

	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return fmt.Errorf("DLQ smoke: cannot read metadata for topic %q: %w", topic, err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("DLQ smoke: topic %q exists but has no partitions", topic)
	}

	probe, err := json.Marshal(map[string]interface{}{
		"probe":      dlqProbeMarker,
		"topic":      topic,
		"timestamp":  time.Now().UTC(),
		"partitions": len(parts),
		"note":       "startup smoke record; not a lost row",
	})
	if err != nil {
		return fmt.Errorf("DLQ smoke: cannot encode probe: %w", err)
	}
	if pub == nil {
		return fmt.Errorf("DLQ smoke: no publisher configured")
	}
	if err := pub.WriteMessages(checkCtx, kafka.Message{Value: probe}); err != nil {
		return fmt.Errorf("DLQ smoke: cannot produce to topic %q (missing ACL?): %w", topic, err)
	}
	return nil
}

// runDLQSmoke performs the check and exits the process when it fails.
//
// The exit is the alarm. It happens at deploy time, where the blast radius is one
// container that never took traffic, rather than mid-stream where the blast radius is a
// partition that stops committing and a tenant whose rows quietly stop landing.
func runDLQSmoke(ctx context.Context, cfg Config, pub dlqPublisher) {
	brokers := splitBrokers(cfg.KafkaBrokers)
	if err := smokeDLQ(ctx, brokers, cfg.DLQTopic, pub); err != nil {
		log.Fatalf("Refusing to start: %v\n"+
			"This loader dead-letters terminally-failed rows. Without a working DLQ those rows "+
			"would exist in neither StarRocks nor any record.", err)
	}
	log.Printf("DLQ smoke passed: %s reachable and writable (%d partition(s))",
		cfg.DLQTopic, countTopicPartitions(brokers, cfg.DLQTopic))
}

func splitBrokers(s string) []string {
	var out []string
	for _, b := range strings.Split(s, ",") {
		if b != "" {
			out = append(out, b)
		}
	}
	return out
}

// countTopicPartitions is best-effort and only used for the startup log line.
func countTopicPartitions(brokers []string, topic string) int {
	if len(brokers) == 0 {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), dlqSmokeTimeout)
	defer cancel()
	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return 0
	}
	defer conn.Close()
	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return 0
	}
	return len(parts)
}
