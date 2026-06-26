package offline

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Mutation struct {
	ID          uuid.UUID       `json:"id"`
	APIEndpoint string          `json:"api_endpoint"`
	Method      string          `json:"method"`
	Payload     json.RawMessage `json:"payload"`
	Timestamp   time.Time       `json:"timestamp"`
}

type SyncResult struct {
	MutationID uuid.UUID `json:"mutation_id"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

type SyncManager struct {
	// In a real app, this would execute the mutations against the APIRuntime
}

func NewSyncManager() *SyncManager {
	return &SyncManager{}
}

func (m *SyncManager) SyncMutations(ctx context.Context, mutations []Mutation) ([]SyncResult, error) {
	results := make([]SyncResult, 0, len(mutations))

	for _, mut := range mutations {
		// Mock execution
		// In reality: http.NewRequest(mut.Method, mut.APIEndpoint, ...)
		// ...

		results = append(results, SyncResult{
			MutationID: mut.ID,
			Success:    true,
		})
	}

	return results, nil
}
