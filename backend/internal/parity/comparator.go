package parity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/nsf/jsondiff"
)

// Status represents the outcome of a comparison between the legacy semantic engine and Cube.
type Status string

const (
	StatusMatch    Status = "match"
	StatusMismatch Status = "mismatch"
	StatusError    Status = "error"
)

// ComparisonRequest captures a single query's responses that should be compared for parity.
type ComparisonRequest struct {
	TenantID  string            `json:"tenant_id"`
	QueryID   string            `json:"query_id"`
	LegacyRaw json.RawMessage   `json:"legacy_payload"`
	CubeRaw   json.RawMessage   `json:"cube_payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tolerance float64           `json:"tolerance,omitempty"`
}

// ComparisonResult describes the final verdict persisted for auditing.
type ComparisonResult struct {
	TenantID   string            `json:"tenant_id"`
	QueryID    string            `json:"query_id"`
	Status     Status            `json:"status"`
	MaxDelta   float64           `json:"max_delta"`
	Diff       string            `json:"diff,omitempty"`
	LegacyHash string            `json:"legacy_hash"`
	CubeHash   string            `json:"cube_hash"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ObservedAt time.Time         `json:"observed_at"`
}

// Comparator executes parity checks with a configured numeric tolerance.
type Comparator struct {
	tolerance float64
}

// NewComparator returns a comparator with a default tolerance (defaults to 1e-6 if <=0).
func NewComparator(tolerance float64) *Comparator {
	if tolerance <= 0 {
		tolerance = 1e-6
	}
	return &Comparator{tolerance: tolerance}
}

// Compare evaluates the two payloads, returning a ComparisonResult with the verdict.
func (c *Comparator) Compare(req ComparisonRequest) (ComparisonResult, error) {
	res := ComparisonResult{
		TenantID:   req.TenantID,
		QueryID:    req.QueryID,
		Status:     StatusMatch,
		ObservedAt: time.Now().UTC(),
		Metadata:   req.Metadata,
	}

	if req.TenantID == "" || req.QueryID == "" {
		return res, errors.New("tenant_id and query_id are required")
	}

	legacyHash := sha256.Sum256(req.LegacyRaw)
	cubeHash := sha256.Sum256(req.CubeRaw)
	res.LegacyHash = fmt.Sprintf("%x", legacyHash[:])
	res.CubeHash = fmt.Sprintf("%x", cubeHash[:])

	var legacyPayload any
	if len(req.LegacyRaw) > 0 {
		if err := json.Unmarshal(req.LegacyRaw, &legacyPayload); err != nil {
			res.Status = StatusError
			res.Diff = fmt.Sprintf("legacy payload decode error: %v", err)
			return res, err
		}
	}
	var cubePayload any
	if len(req.CubeRaw) > 0 {
		if err := json.Unmarshal(req.CubeRaw, &cubePayload); err != nil {
			res.Status = StatusError
			res.Diff = fmt.Sprintf("cube payload decode error: %v", err)
			return res, err
		}
	}

	tol := req.Tolerance
	if tol <= 0 {
		tol = c.tolerance
	}

	maxDelta := comparePayloads(legacyPayload, cubePayload)
	res.MaxDelta = maxDelta
	if maxDelta > tol {
		res.Status = StatusMismatch
	}

	opts := jsondiff.DefaultConsoleOptions()
	if diffType, diff := jsondiff.Compare(req.LegacyRaw, req.CubeRaw, &opts); diffType != jsondiff.FullMatch {
		res.Diff = diff
	}

	return res, nil
}

func comparePayloads(a, b any) float64 {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return math.Inf(1)
		}
		return math.Abs(av - bv)
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return math.Inf(1)
		}
		var max float64
		for key, val := range av {
			max = math.Max(max, comparePayloads(val, bv[key]))
		}
		for key, val := range bv {
			if _, ok := av[key]; !ok {
				max = math.Max(max, comparePayloads(nil, val))
			}
		}
		return max
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return math.Inf(1)
		}
		var max float64
		maxLen := len(av)
		if len(bv) > maxLen {
			maxLen = len(bv)
		}
		for i := 0; i < maxLen; i++ {
			var avItem, bvItem any
			if i < len(av) {
				avItem = av[i]
			}
			if i < len(bv) {
				bvItem = bv[i]
			}
			max = math.Max(max, comparePayloads(avItem, bvItem))
		}
		return max
	case bool, string, nil:
		if fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) {
			return 0
		}
		return math.Inf(1)
	default:
		if fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) {
			return 0
		}
		return math.Inf(1)
	}
}

// StoreResult persists the comparison result into the migration parity table for auditing.
func StoreResult(ctx context.Context, db *sql.DB, res ComparisonResult) error {
	if db == nil {
		return errors.New("db is nil")
	}
	metadataJSON, err := json.Marshal(res.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	const query = `INSERT INTO migration.parity_results (
		tenant_id, query_id, status, max_delta, diff, legacy_hash, cube_hash, metadata, observed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err = db.ExecContext(
		ctx,
		query,
		res.TenantID,
		res.QueryID,
		res.Status,
		res.MaxDelta,
		res.Diff,
		res.LegacyHash,
		res.CubeHash,
		metadataJSON,
		res.ObservedAt,
	)
	if err != nil {
		return fmt.Errorf("insert parity_results: %w", err)
	}
	return nil
}
