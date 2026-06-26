package parity

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestComparatorMatch(t *testing.T) {
	c := NewComparator(0.001)
	req := ComparisonRequest{
		TenantID:  "tenant-1",
		QueryID:   "q-1",
		LegacyRaw: []byte(`{"value": 100.0}`),
		CubeRaw:   []byte(`{"value": 100.0005}`),
	}
	res, err := c.Compare(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusMatch {
		t.Fatalf("expected match got %s", res.Status)
	}
	if res.MaxDelta == 0 {
		t.Fatalf("expected non-zero delta")
	}
}

func TestComparatorMismatch(t *testing.T) {
	c := NewComparator(0.0001)
	req := ComparisonRequest{
		TenantID:  "tenant-1",
		QueryID:   "q-2",
		LegacyRaw: []byte(`{"value": 1}`),
		CubeRaw:   []byte(`{"value": 2}`),
	}
	res, err := c.Compare(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusMismatch {
		t.Fatalf("expected mismatch got %s", res.Status)
	}
}

func TestStoreResultRequiresDB(t *testing.T) {
	res := ComparisonResult{TenantID: "t", QueryID: "q", ObservedAt: time.Now()}
	err := StoreResult(context.Background(), (*sql.DB)(nil), res)
	if err == nil {
		t.Fatalf("expected error when db is nil")
	}
}

func TestCompareRequiresIDs(t *testing.T) {
	c := NewComparator(0.01)
	_, err := c.Compare(ComparisonRequest{})
	if err == nil {
		t.Fatalf("expected error for missing IDs")
	}
	if !regexp.MustCompile(`required`).MatchString(err.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}
