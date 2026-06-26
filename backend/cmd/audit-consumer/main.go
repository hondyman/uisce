//go:build !no_parquet

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	kafka "github.com/segmentio/kafka-go"
	parquet "github.com/segmentio/parquet-go"
)

// AuditEventParquet is a Parquet row schema for audit events
type AuditEventParquet struct {
	ID         string `parquet:"name=id"`
	InstanceID string `parquet:"name=instance_id"`
	TenantID   string `parquet:"name=tenant_id"`
	BPKey      string `parquet:"name=bp_key"`
	EventType  string `parquet:"name=event_type"`
	StepKey    string `parquet:"name=step_key"`
	ActorID    string `parquet:"name=actor_id"`
	ActorRole  string `parquet:"name=actor_role"`
	Reason     string `parquet:"name=reason"`
	IPAddress  string `parquet:"name=ip_address"`
	UserAgent  string `parquet:"name=user_agent"`
	CreatedAt  string `parquet:"name=created_at"`
	OldValue   string `parquet:"name=old_value_json"`
	NewValue   string `parquet:"name=new_value_json"`
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func main() {
	ctx := context.Background()

	brokers := getenv("KAFKA_BROKERS", "redpanda:9092")
	topic := getenv("AUDIT_TOPIC", "audit.events")
	groupID := getenv("AUDIT_GROUP_ID", "audit-parquet-consumer")

	minioEndpoint := getenv("MINIO_ENDPOINT", "http://minio:9000")
	minioAccess := getenv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecret := getenv("MINIO_SECRET_KEY", "minioadmin")
	minioBucket := getenv("MINIO_BUCKET", "audit")

	secure := strings.HasPrefix(minioEndpoint, "https://")
	endpoint := strings.TrimPrefix(strings.TrimPrefix(minioEndpoint, "http://"), "https://")

	// MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccess, minioSecret, ""),
		Secure: secure,
	})
	if err != nil {
		log.Fatalf("failed to init minio: %v", err)
	}
	// Ensure bucket exists
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("failed to check bucket: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("failed to create bucket: %v", err)
		}
	}

	// Kafka Reader
	brokerList := strings.Split(brokers, ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Printf("audit-consumer started: brokers=%s topic=%s group=%s minio=%s/%s", brokers, topic, groupID, endpoint, minioBucket)

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("fetch error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var evt events.AuditEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("unmarshal error: %v", err)
			// Commit bad message to skip
			reader.CommitMessages(ctx, m)
			continue
		}

		// Build Parquet row
		oldJSON, _ := json.Marshal(evt.OldValue)
		newJSON, _ := json.Marshal(evt.NewValue)
		row := &AuditEventParquet{
			ID:         evt.ID,
			InstanceID: evt.InstanceID,
			TenantID:   evt.TenantID,
			BPKey:      evt.BPKey,
			EventType:  evt.EventType,
			StepKey:    evt.StepKey,
			ActorID:    evt.ActorID,
			ActorRole:  evt.ActorRole,
			Reason:     evt.Reason,
			IPAddress:  evt.IPAddress,
			UserAgent:  evt.UserAgent,
			CreatedAt:  evt.CreatedAt,
			OldValue:   string(oldJSON),
			NewValue:   string(newJSON),
		}

		// Write single-row Parquet to memory buffer (segmentio/parquet-go)
		var buf bytes.Buffer
		w := parquet.NewWriter(&buf)
		if err := w.Write(row); err != nil {
			log.Printf("parquet write error: %v", err)
			// Don't commit, likely retriable or stuck
			time.Sleep(1 * time.Second)
			continue
		}
		if err := w.Close(); err != nil {
			log.Printf("parquet close error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		date := time.Now().Format("2006-01-02")
		// Use Partition/Offset for uniqueness if ID not available? Event ID should be there.
		fileName := evt.ID
		if fileName == "" {
			fileName = fmt.Sprintf("%d-%d", m.Partition, m.Offset)
		}
		objectName := fmt.Sprintf("tenant_id=%s/date=%s/events/%s.parquet", safe(evt.TenantID), date, fileName)
		data := buf.Bytes()
		_, err = minioClient.PutObject(ctx, minioBucket, objectName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			log.Printf("minio put error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("uploaded parquet: %s", objectName)

		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("commit error: %v", err)
		}
	}
}

func safe(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
