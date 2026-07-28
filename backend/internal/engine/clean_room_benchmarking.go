package engine

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/hondyman/uisce/backend/internal/domain"
)

// BenchmarkPayload represents the anonymized metric entering the Clean Room
type BenchmarkPayload struct {
	BOID             string    `json:"bo_id"`
	ClassificationID string    `json:"classification_id"` // e.g., "HedgeFund", "RetailBank" - NEVER the TenantID
	MetricName       string    `json:"metric_name"`
	AnonymizedValue  float64   `json:"anonymized_value"`
	WindowEndTime    time.Time `json:"window_end_time"`
}

type CleanRoomBenchmarkingPublisher struct {
	eventStream domain.EventPublisher
	noiseFactor float64 // Epsilon / noise factor for differential privacy (e.g., 0.02 for +/- 2% noise)
}

func NewCleanRoomBenchmarkingPublisher(publisher domain.EventPublisher, noiseFactor float64) *CleanRoomBenchmarkingPublisher {
	if noiseFactor <= 0 {
		noiseFactor = 0.02
	}
	return &CleanRoomBenchmarkingPublisher{
		eventStream: publisher,
		noiseFactor: noiseFactor,
	}
}

// PublishAnonymizedMetric strips PII, injects differential privacy noise, and routes to the global Redpanda topic
func (c *CleanRoomBenchmarkingPublisher) PublishAnonymizedMetric(ctx context.Context, boID, classificationID, metricName string, rawValue float64) error {
	// 🚨 RULE 7 ENFORCEMENT: Explicitly drop tenant_id. It is NOT passed into the payload.

	// Inject Differential Privacy Noise
	noiseMultiplier, _ := rand.Int(rand.Reader, big.NewInt(200))          // 0 to 200
	normalizedNoise := (float64(noiseMultiplier.Int64()) - 100.0) / 100.0 // -1.0 to 1.0

	appliedNoise := rawValue * c.noiseFactor * normalizedNoise
	safeValue := rawValue + appliedNoise

	payload := BenchmarkPayload{
		BOID:             boID,
		ClassificationID: classificationID,
		MetricName:       metricName,
		AnonymizedValue:  safeValue,
		WindowEndTime:    time.Now().UTC(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark payload: %w", err)
	}

	if c.eventStream != nil {
		// Publish to the cross-tenant global analytics topic
		// Note: The routing key is the ClassificationID (e.g., "AssetManager"), NOT the TenantID.
		err = c.eventStream.Publish("bo_benchmark_anonymized", classificationID, payloadBytes)
		if err != nil {
			return fmt.Errorf("clean room dispatch failed: %w", err)
		}
	}

	return nil
}
