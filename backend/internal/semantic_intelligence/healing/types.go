package healing

import (
	"context"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/drift"
)

type HealProposal struct {
	PageID      string              `json:"page_id"`
	Description string              `json:"description"`
	Changes     []string            `json:"changes"`
	HealedPage  pagestudio.CorePage `json:"healed_page"`
}

type Strategy interface {
	Name() string
	CanHeal(event drift.DriftEvent) bool
	Heal(ctx context.Context, page *pagestudio.CorePage, event drift.DriftEvent) ([]string, error)
}
