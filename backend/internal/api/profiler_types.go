package api

import (
	"sync"
	"time"
)

// ProfileRequest mirrors the request used by the standalone profiler and
// is reused by the HTTP API profiler endpoints.
type ProfileRequest struct {
	DataSource   string   `json:"datasource"`
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
	Schema       string   `json:"schema"`
	Tables       []string `json:"tables"`
	NodeIDs      []string `json:"node_ids,omitempty"` // New field for catalog node IDs
	SampleSize   int      `json:"sample_size,omitempty"`
	FPRate       float64  `json:"fp_rate,omitempty"`
	BatchSize    int      `json:"batch_size,omitempty"`
}

// ProfileJob represents an in-memory profiling job tracked by the Server.
type ProfileJob struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Req       ProfileRequest
	Results   interface{} `json:"results,omitempty"`
	mu        sync.Mutex  `json:"-"`
}
