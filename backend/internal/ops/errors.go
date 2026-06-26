package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// Pattern to find UUIDs, hex strings, and other IDs
	idPattern = regexp.MustCompile(`[0-9a-fA-F]{8}[-]?[0-9a-fA-F]{4}[-]?[0-9a-fA-F]{4}[-]?[0-9a-fA-F]{4}[-]?[0-9a-fA-F]{12}|[0-9a-fA-F]{8,}`)
	// Pattern for timestamps
	timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	// Pattern for numbers
	numberPattern = regexp.MustCompile(`\d+`)
)

// ErrorFingerprinter generates stable fingerprints for similar errors
type ErrorFingerprinter struct {
	store    Store
	timeline *TimelineService
}

// NewErrorFingerprinter creates a new error fingerprinter
func NewErrorFingerprinter(store Store) *ErrorFingerprinter {
	return &ErrorFingerprinter{store: store, timeline: nil}
}

// NewErrorFingerprinterWithTimeline creates a new error fingerprinter with timeline service
func NewErrorFingerprinterWithTimeline(store Store, timeline *TimelineService) *ErrorFingerprinter {
	return &ErrorFingerprinter{store: store, timeline: timeline}
}

// Fingerprint generates a stable hash for an error
func (e *ErrorFingerprinter) Fingerprint(path string, statusCode int, message string) string {
	// Normalize the message to remove dynamic parts
	normalized := normalizeErrorMessage(message)

	// Create base string
	base := fmt.Sprintf("path=%s|status=%d|msg=%s", path, statusCode, normalized)

	// Hash with SHA256
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

// RecordError records an error and updates fingerprint counts
func (e *ErrorFingerprinter) RecordError(ctx context.Context, input CreateErrorInput) error {
	fingerprint := e.Fingerprint(input.Path, input.StatusCode, input.Message)

	// Get or create fingerprint
	fp, err := e.store.GetOrCreateFingerprint(ctx, fingerprint, input.Path, input.StatusCode, input.Message)
	if err != nil {
		return fmt.Errorf("get or create fingerprint: %w", err)
	}

	// Increment count
	if err := e.store.UpdateFingerprintCount(ctx, fp.ID, fp.Count+1); err != nil {
		return fmt.Errorf("update fingerprint count: %w", err)
	}

	// Record the error event
	event := ErrorEvent{
		ID:            uuid.New(),
		FingerprintID: fp.ID,
		TenantID:      input.TenantID,
		Endpoint:      input.Path,
		StatusCode:    input.StatusCode,
		Message:       input.Message,
		RequestID:     input.RequestID,
		OccurredAt:    time.Now().UTC(),
	}

	if err := e.store.InsertErrorEvent(ctx, event); err != nil {
		return fmt.Errorf("insert error event: %w", err)
	}

	// Emit timeline event if timeline service available
	if e.timeline != nil {
		_ = e.timeline.RecordErrorFingerprint(ctx, *fp)
	}

	return nil
}

// ListFingerprints returns the top error fingerprints
func (e *ErrorFingerprinter) ListFingerprints(ctx context.Context, limit int) ([]ErrorFingerprint, error) {
	return e.store.ListFingerprints(ctx, limit)
}

// GetFingerprintHistory returns recent occurrences of a fingerprinted error
func (e *ErrorFingerprinter) GetFingerprintHistory(ctx context.Context, fingerprintID uuid.UUID, limit int) ([]ErrorEvent, error) {
	return e.store.GetFingerprintEvents(ctx, fingerprintID, limit)
}

// normalizeErrorMessage removes dynamic parts from error messages
func normalizeErrorMessage(msg string) string {
	// Remove UUIDs and hex IDs
	msg = idPattern.ReplaceAllString(msg, "<id>")
	// Remove timestamps
	msg = timestampPattern.ReplaceAllString(msg, "<timestamp>")
	// Remove large numbers (for things like file sizes, request IDs, etc)
	msg = regexp.MustCompile(`\b\d{10,}\b`).ReplaceAllString(msg, "<num>")

	// Convert to lowercase for consistency
	msg = strings.ToLower(msg)

	return msg
}
