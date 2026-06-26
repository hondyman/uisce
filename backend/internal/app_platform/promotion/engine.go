package promotion

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AppChangeSet struct {
	ID              uuid.UUID   `json:"id"`
	AppID           uuid.UUID   `json:"app_id"`
	AppName         string      `json:"app_name"`
	SourceEnv       string      `json:"source_env"`
	TargetEnv       string      `json:"target_env"`
	BOChanges       []uuid.UUID `json:"bo_changes"`
	APIChanges      []uuid.UUID `json:"api_changes"`
	PageChanges     []uuid.UUID `json:"page_changes"`
	WorkflowChanges []uuid.UUID `json:"workflow_changes"`
	ThemeChanges    []uuid.UUID `json:"theme_changes"`
	SLOChanges      []uuid.UUID `json:"slo_changes"`
	PolicyChanges   []uuid.UUID `json:"policy_changes"`
	Status          string      `json:"status"` // pending, approved, promoted
	CreatedAt       time.Time   `json:"created_at"`
}

type PromotionResult struct {
	ChangeSetID    uuid.UUID `json:"changeset_id"`
	Success        bool      `json:"success"`
	ObjectsApplied int       `json:"objects_applied"`
	Errors         []string  `json:"errors,omitempty"`
	PromotedAt     time.Time `json:"promoted_at"`
}

type PromotionEngine struct {
	// Integration with CRS, BO service, API service, Page service
}

func NewPromotionEngine() *PromotionEngine {
	return &PromotionEngine{}
}

func (e *PromotionEngine) CreateChangeSet(ctx context.Context, appID uuid.UUID, sourceEnv, targetEnv string) (*AppChangeSet, error) {
	// Mock: Generate changeset for app
	cs := &AppChangeSet{
		ID:          uuid.New(),
		AppID:       appID,
		AppName:     "Wealth Management App",
		SourceEnv:   sourceEnv,
		TargetEnv:   targetEnv,
		BOChanges:   []uuid.UUID{uuid.New()},
		APIChanges:  []uuid.UUID{uuid.New(), uuid.New()},
		PageChanges: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	return cs, nil
}

func (e *PromotionEngine) PromoteApp(ctx context.Context, changeSetID uuid.UUID) (*PromotionResult, error) {
	// Mock: Atomic promotion
	// Real: Apply changes in order: BOs → APIs → Pages → Workflows → Themes → SLOs → Policies
	result := &PromotionResult{
		ChangeSetID:    changeSetID,
		Success:        true,
		ObjectsApplied: 8,
		PromotedAt:     time.Now(),
	}
	return result, nil
}

func (e *PromotionEngine) RollbackPromotion(ctx context.Context, changeSetID uuid.UUID) error {
	// Mock: Rollback entire app promotion
	return nil
}
