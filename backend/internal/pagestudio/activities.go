package pagestudio

import (
	"context"

	"github.com/google/uuid"
)

type Activities struct {
	repo         *Repository
	reconService *ReconciliationService
}

func NewActivities(repo *Repository, recon *ReconciliationService) *Activities {
	return &Activities{repo: repo, reconService: recon}
}

type AnalyzeImpactRequest struct {
	CorePageID uuid.UUID `json:"core_page_id"`
	OldVersion int       `json:"old_version"`
	NewVersion int       `json:"new_version"`
}

func (a *Activities) AnalyzeCoreUpgradeImpact(ctx context.Context, req AnalyzeImpactRequest) ([]uuid.UUID, error) {
	// 1. Load active core page
	core, err := a.repo.GetPage(ctx, req.CorePageID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch all tenant overlays for this core page
	overlays, err := a.repo.ListOverlaysForPage(ctx, req.CorePageID)
	if err != nil {
		return nil, err
	}

	// 3. (Mock) Load old core for diffing
	// In a real system, we'd load vN-1 from a version store
	oldCore := *core
	oldCore.Version = req.OldVersion

	impactIDs := []uuid.UUID{}
	for _, ov := range overlays {
		impact, err := a.reconService.ComputeUpgradeImpact(&oldCore, core, &ov)
		if err != nil {
			continue
		}

		err = a.repo.SaveUpgradeImpact(ctx, impact)
		if err == nil {
			impactIDs = append(impactIDs, impact.ID)
		}
	}

	return impactIDs, nil
}
