package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/apistudio"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type Repository interface {
	GetEndpoint(ctx context.Context, id uuid.UUID) (*apistudio.APIEndpoint, error)
}

type DriftDetector struct {
	apiRepo *apistudio.Repository
}

func NewDriftDetector(apiRepo *apistudio.Repository) *DriftDetector {
	return &DriftDetector{
		apiRepo: apiRepo,
	}
}

// CheckDrift compares the stored fingerprint against current semantic state
func (d *DriftDetector) CheckDrift(ctx context.Context, pageID string, fp *pagestudio.PageFingerprint) (*DriftReport, error) {
	report := &DriftReport{
		PageID:     pageID,
		DetectedAt: time.Now().Format(time.RFC3339),
		Events:     make([]DriftEvent, 0),
	}

	// 1. Check API Endpoints
	for _, epID := range fp.APIEndpointIDs {
		// epID is uuid.UUID
		ep, err := d.apiRepo.GetEndpoint(ctx, epID)
		if err != nil {
			report.Events = append(report.Events, DriftEvent{
				Type:        DriftTypeEndpointMissing,
				Severity:    "critical",
				Description: fmt.Sprintf("Referenced API endpoint %s not found", epID),
				ItemName:    epID.String(),
			})
			continue
		}
		// TODO: Deep check filters
		_ = ep
	}

	// 2. Check BO Fields (Mock for now, assumes success)

	report.HasDrift = len(report.Events) > 0
	return report, nil
}
