package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	kafka "github.com/segmentio/kafka-go"
	parquet "github.com/segmentio/parquet-go"
)

type IcebergSinkConsumer struct {
	readers       []*kafka.Reader
	icebergWriter *IcebergWriter
	eventBuffer   []KafkaEventEnvelope
	batchSize     int
	flushInterval time.Duration
	mu            sync.RWMutex
	stopChan      chan struct{}
	running       bool
}

type IcebergWriter struct {
	S3Client   *minio.Client
	BucketName string
	CatalogURI string
}

type auditEventParquet struct {
	EventID    string `parquet:"name=event_id"`
	EventType  string `parquet:"name=event_type"`
	Version    string `parquet:"name=version"`
	Timestamp  int64  `parquet:"name=timestamp"`
	TenantID   string `parquet:"name=tenant_id"`
	Source     string `parquet:"name=source"`
	Payload    string `parquet:"name=payload"`
	UploadedAt int64  `parquet:"name=uploaded_at"`
}

func NewMinIOClient(endpoint, accessKey, secretKey string) (*minio.Client, error) {
	secure := false
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
}

func NewIcebergSinkConsumer(bootstrapServers, groupID string, topics []string, writer *IcebergWriter) (*IcebergSinkConsumer, error) {
	if bootstrapServers == "" {
		return nil, fmt.Errorf("bootstrap servers cannot be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic must be specified")
	}
	if writer == nil {
		return nil, fmt.Errorf("iceberg writer cannot be nil")
	}

	var readers []*kafka.Reader
	for _, topic := range topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{bootstrapServers},
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 10e3,
			MaxBytes: 10e6,
		})
		readers = append(readers, r)
	}

	return &IcebergSinkConsumer{
		readers:       readers,
		icebergWriter: writer,
		eventBuffer:   make([]KafkaEventEnvelope, 0, 1000),
		batchSize:    1000,
		flushInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
		running:       false,
	}, nil
}

func (c *IcebergSinkConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	log.Printf("[Audit-Sink] Starting %d topic readers", len(c.readers))

	flushTicker := time.NewTicker(c.flushInterval)
	defer flushTicker.Stop()

	type fetchResult struct {
		readerIdx int
		msg       kafka.Message
		err       error
	}

	for {
		fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		resultChan := make(chan fetchResult, len(c.readers))

		for i, r := range c.readers {
			go func(idx int, reader *kafka.Reader) {
				msg, err := reader.FetchMessage(fetchCtx)
				resultChan <- fetchResult{idx, msg, err}
			}(i, r)
		}

		var msg kafka.Message
		pendingCommits := 0
		for pendingCommits < len(c.readers) {
			select {
			case <-ctx.Done():
				cancel()
				_ = c.flush(ctx)
				c.mu.Lock()
				c.running = false
				c.mu.Unlock()
				return ctx.Err()

			case <-c.stopChan:
				cancel()
				_ = c.flush(ctx)
				c.mu.Lock()
				c.running = false
				c.mu.Unlock()
				return nil

			case <-flushTicker.C:
				cancel()
				if err := c.flush(ctx); err != nil {
					log.Printf("[Audit-Sink] Flush error: %v", err)
				}

			case res := <-resultChan:
				pendingCommits++
				if res.err != nil {
					if fetchCtx.Err() == nil {
						log.Printf("[Audit-Sink] Fetch error reader[%d]: %v", res.readerIdx, res.err)
					}
					continue
				}
				msg = res.msg
				topic := c.readers[res.readerIdx].Config().Topic

				var envelope KafkaEventEnvelope
				if err := json.Unmarshal(msg.Value, &envelope); err != nil {
					log.Printf("[Audit-Sink] Unmarshaling error topic=%s: %v", topic, err)
					if commitErr := c.readers[res.readerIdx].CommitMessages(fetchCtx, msg); commitErr != nil {
						log.Printf("[Audit-Sink] Failed to commit bad message: %v", commitErr)
					}
					continue
				}

				c.mu.Lock()
				c.eventBuffer = append(c.eventBuffer, envelope)
				shouldFlush := len(c.eventBuffer) >= c.batchSize
				c.mu.Unlock()

				if shouldFlush {
					cancel()
					if err := c.flush(ctx); err != nil {
						log.Printf("[Audit-Sink] Batch flush error: %v", err)
					}
				}

				if err := c.readers[res.readerIdx].CommitMessages(fetchCtx, msg); err != nil {
					log.Printf("[Audit-Sink] Commit error topic=%s: %v", topic, err)
				}

			case <-fetchCtx.Done():
				cancel()
			}
		}
		cancel()
	}
}

func (c *IcebergSinkConsumer) flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.eventBuffer) == 0 {
		c.mu.Unlock()
		return nil
	}
	eventsToWrite := make([]KafkaEventEnvelope, len(c.eventBuffer))
	copy(eventsToWrite, c.eventBuffer)
	c.eventBuffer = c.eventBuffer[:0]
	c.mu.Unlock()

	eventsByTenantAndTopic := make(map[string][]KafkaEventEnvelope)
	for _, event := range eventsToWrite {
		topic := c.getTopicForEvent(event.EventType)
		key := fmt.Sprintf("%s::%s", event.TenantID, topic)
		eventsByTenantAndTopic[key] = append(eventsByTenantAndTopic[key], event)
	}

	for key, events := range eventsByTenantAndTopic {
		var tenantID, topic string
		for i, ch := range key {
			if ch == ':' && i+2 < len(key) && key[i+1] == ':' {
				tenantID = key[:i]
				topic = key[i+2:]
				break
			}
		}
		if tenantID == "" {
			tenantID = "global"
		}

		if err := c.writeParquetBatch(ctx, tenantID, topic, events); err != nil {
			log.Printf("[Audit-Sink] Write error tenant=%s topic=%s: %v", tenantID, topic, err)
		} else {
			log.Printf("[Audit-Sink] Flushed %d events tenant=%s topic=%s", len(events), tenantID, topic)
		}
	}

	return nil
}

func (c *IcebergSinkConsumer) writeParquetBatch(ctx context.Context, tenantID, topic string, events []KafkaEventEnvelope) error {
	if len(events) == 0 {
		return nil
	}

	var buf bytes.Buffer
	pw := parquet.NewWriter(&buf)

	for _, e := range events {
		payloadStr := string(e.Payload)
		row := &auditEventParquet{
			EventID:    e.EventID,
			EventType:  e.EventType,
			Version:    e.Version,
			Timestamp:  e.Timestamp.Unix(),
			TenantID:   e.TenantID,
			Source:     e.Source,
			Payload:    payloadStr,
			UploadedAt: time.Now().Unix(),
		}
		if err := pw.Write(row); err != nil {
			return fmt.Errorf("parquet write: %w", err)
		}
	}

	if err := pw.Close(); err != nil {
		return fmt.Errorf("parquet close: %w", err)
	}

	objectKey := fmt.Sprintf("%s/audit/%s/data/%s-%s.parquet",
		tenantID,
		topic,
		time.Now().Format("20060102-150405"),
		shortUUID(),
	)

	_, err := c.icebergWriter.S3Client.PutObject(ctx, c.icebergWriter.BucketName, objectKey, bytes.NewReader(buf.Bytes()), int64(buf.Len()), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("S3 put %s: %w", objectKey, err)
	}

	log.Printf("[Audit-Sink] Uploaded s3://%s/%s (%d bytes)", c.icebergWriter.BucketName, objectKey, buf.Len())
	return nil
}

func (c *IcebergSinkConsumer) getTopicForEvent(eventType string) string {
	switch eventType {
	case EventTypeJobRunCompleted:
		return TopicSchedulerJobRuns
	case EventTypeDAGRunCompleted:
		return TopicSchedulerDAGRuns
	case EventTypeChangeSetCreated, EventTypeChangeSetApproved, EventTypeChangeSetApplied:
		return TopicGovernanceChangeSets
	case EventTypeSemanticSnapshot:
		return TopicSemanticSnapshots
	case EventTypeWorkflowStarted, EventTypeWorkflowCompleted, EventTypeWorkflowFailed:
		return TopicOrchestrationEvents
	case EventTypeComplianceViolation:
		return TopicComplianceViolations
	case EventTypeAINarrativeGenerated:
		return TopicAISuggestions
	default:
		return "unknown"
	}
}

func (c *IcebergSinkConsumer) GetBufferStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"buffer_size": len(c.eventBuffer),
		"max_size":    c.batchSize,
		"running":     c.running,
	}
}

func (c *IcebergSinkConsumer) Stop(ctx context.Context) error {
	select {
	case c.stopChan <- struct{}{}:
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			c.mu.RLock()
			if !c.running {
				c.mu.RUnlock()
				return nil
			}
			c.mu.RUnlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *IcebergSinkConsumer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Stop(ctx); err != nil && err != context.DeadlineExceeded {
		return err
	}
	for _, r := range c.readers {
		r.Close()
	}
	return nil
}

var uuidCounter int64 = 0

func shortUUID() string {
	uuidCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), uuidCounter)
}
