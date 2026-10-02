package activities

import (
	"context"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type AuditAnchorActivities struct {
	DB     *sqlx.DB
	Logger *zap.SugaredLogger
}

func NewAuditAnchorActivities(db *sqlx.DB, logger *zap.SugaredLogger) *AuditAnchorActivities {
	return &AuditAnchorActivities{
		DB:     db,
		Logger: logger,
	}
}

type AuditAnchorActivityInput struct {
	TenantID     string `json:"tenant_id"`
	MaxBatchSize int    `json:"max_batch_size,omitempty"`
}

type AuditAnchorActivityResult struct {
	Anchored   bool                        `json:"anchored"`
	AnchorInfo *analytics.AuditAnchorResult `json:"anchor_info,omitempty"`
}

// RunAuditAnchorActivity folds unchained validation violations into retroactive hash chains
// and commits a signed anchor block into violation_audit_anchors.
func (a *AuditAnchorActivities) RunAuditAnchorActivity(ctx context.Context, input AuditAnchorActivityInput) (*AuditAnchorActivityResult, error) {
	if input.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	maxBatch := input.MaxBatchSize
	if maxBatch <= 0 {
		maxBatch = 5000
	}

	res, err := analytics.FoldUnchainedViolations(ctx, a.DB, input.TenantID, maxBatch)
	if err != nil {
		return nil, fmt.Errorf("fold unchained violations for tenant %s: %w", input.TenantID, err)
	}

	if res == nil {
		if a.Logger != nil {
			a.Logger.Debugf("[AuditAnchor] No unchained violations for tenant %s", input.TenantID)
		}
		return &AuditAnchorActivityResult{Anchored: false}, nil
	}

	if a.Logger != nil {
		a.Logger.Infof("[AuditAnchor] Anchored %d rows for tenant %s (seq %d..%d, hash %s)",
			res.RowCount, res.TenantID, res.SeqFrom, res.SeqTo, res.AnchorHash)
	}

	return &AuditAnchorActivityResult{
		Anchored:   true,
		AnchorInfo: res,
	}, nil
}
