package bundles

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	statusdb "github.com/hondyman/uisce/backend/internal/db"
)

var proposalStatusUpdater *statusdb.StatusUpdater

func SetProposalStatusUpdater(sqlxDB *sqlx.DB) {
	proposalStatusUpdater = statusdb.NewStatusUpdater(sqlxDB, "bundle_change_proposal", "id")
}

func ApproveProposal(ctx context.Context, proposalID uuid.UUID, approver string, decidedAt time.Time) error {
	return proposalStatusUpdater.TransitionWithActor(ctx, proposalID, "approved", approver, decidedAt)
}

func RejectProposal(ctx context.Context, proposalID uuid.UUID, approver string, decidedAt time.Time) error {
	return proposalStatusUpdater.TransitionWithActor(ctx, proposalID, "rejected", approver, decidedAt)
}

func AutoApplyProposal(ctx context.Context, proposalID uuid.UUID, actor string, decidedAt time.Time) error {
	return proposalStatusUpdater.TransitionWithActor(ctx, proposalID, "auto_applied", actor, decidedAt)
}
