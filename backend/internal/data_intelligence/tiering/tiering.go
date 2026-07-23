package tiering

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StorageTier string

const (
	TierHot     StorageTier = "hot"     // Frequently accessed, latency-sensitive
	TierWarm    StorageTier = "warm"    // Moderately accessed, cost-optimized
	TierCold    StorageTier = "cold"    // Archival, compliance-driven
	TierArchive StorageTier = "archive" // Tape/Glacier, long-term retention
)

const (
	StatusPending   = "pending"
	StatusMigrating = "migrating"
	StatusCompleted = "completed"
	StatusDismissed = "dismissed"
)

type TieringRule struct {
	TableName   string      `json:"table_name"`
	Condition   string      `json:"condition"`
	TargetTier  StorageTier `json:"target_tier"`
	Rationale   string      `json:"rationale"`
	DataVolume  string      `json:"data_volume"`
	CostSavings string      `json:"cost_savings"`
}

type TieringPlan struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  string        `json:"tenant_id"`
	Rules     []TieringRule `json:"rules"`
	Summary   string        `json:"summary"`
	Status    string        `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type StorageTiering struct {
	repo      *TieringRepository
	logger    *zap.Logger
	listeners []StorageEventListener
}

func (st *StorageTiering) Subscribe(listener StorageEventListener) {
	st.listeners = append(st.listeners, listener)
}

func (st *StorageTiering) notifyListeners(event StorageEvent) {
	for _, listener := range st.listeners {
		if err := listener.OnStorageEvent(event); err != nil {
			st.logger.Error("failed to notify storage event listener",
				zap.Error(err),
				zap.String("listener", fmt.Sprintf("%T", listener)),
			)
		}
	}
}

func NewStorageTiering(repo *TieringRepository, logger *zap.Logger) *StorageTiering {
	return &StorageTiering{
		repo:   repo,
		logger: logger,
	}
}

func (st *StorageTiering) GeneratePlan(ctx context.Context, tenantID string) (*TieringPlan, error) {
	st.logger.Info("Generating storage tiering plan", zap.String("tenant_id", tenantID))

	// In production, we would:
	// 1. Query information_schema.tables to get sizes
	// 2. Query access logs or query history to get frequency
	// 3. Apply AI rules to determine best tier
	// For now, we utilize the real repo but keep mock analysis logic for demonstration

	plan := &TieringPlan{
		ID:       uuid.New(),
		TenantID: tenantID,
		Status:   StatusPending,
		Rules: []TieringRule{
			{
				TableName:   "positions",
				Condition:   "as_of_date >= CURRENT_DATE - INTERVAL '90 days'",
				TargetTier:  TierHot,
				Rationale:   "Last 90 days of positions accessed frequently (847 queries/week). Keep in hot tier for low latency.",
				DataVolume:  "2.5GB",
				CostSavings: "N/A (hot tier required for SLO compliance)",
			},
			{
				TableName:   "positions",
				Condition:   "as_of_date < CURRENT_DATE - INTERVAL '90 days' AND as_of_date >= CURRENT_DATE - INTERVAL '1 year'",
				TargetTier:  TierWarm,
				Rationale:   "91-365 day old positions accessed moderately (120 queries/week). Move to warm tier for cost optimization.",
				DataVolume:  "8.2GB",
				CostSavings: "$450/month",
			},
			{
				TableName:   "positions",
				Condition:   "as_of_date < CURRENT_DATE - INTERVAL '1 year'",
				TargetTier:  TierCold,
				Rationale:   "Positions older than 1 year rarely accessed (5 queries/month). Move to cold tier for compliance retention.",
				DataVolume:  "45GB",
				CostSavings: "$2,100/month",
			},
			{
				TableName:   "trades",
				Condition:   "trade_date < CURRENT_DATE - INTERVAL '2 years'",
				TargetTier:  TierArchive,
				Rationale:   "Tenant-specific: rarely queried historical trades (2 queries/month). Move to archive tier.",
				DataVolume:  "12GB",
				CostSavings: "$600/month",
			},
		},
		Summary: "Total cost savings: $3,150/month. No SLO impact. Compliance requirements met.",
	}

	if err := st.repo.SavePlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to save tiering plan: %w", err)
	}

	return plan, nil
}

func (st *StorageTiering) ListPlans(ctx context.Context, tenantID string) ([]TieringPlan, error) {
	return st.repo.ListPlans(ctx, tenantID)
}

func (st *StorageTiering) GetPlan(ctx context.Context, id uuid.UUID) (*TieringPlan, error) {
	return st.repo.GetPlan(ctx, id)
}

func (st *StorageTiering) ExecutePlan(ctx context.Context, plan *TieringPlan) error {
	// Mock: Execute tiering plan and emit events
	for _, rule := range plan.Rules {
		// Mock simulation of movement
		st.logger.Info("Executing tiering rule",
			zap.String("table", rule.TableName),
			zap.String("target_tier", string(rule.TargetTier)),
		)

		// Create validation event
		event := StorageEvent{
			ID:        uuid.New(),
			TenantID:  plan.TenantID,
			Type:      getEventType(rule.TargetTier),
			TableName: rule.TableName,
			NewTier:   rule.TargetTier,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"plan_id":   plan.ID,
				"rationale": rule.Rationale,
			},
		}

		st.notifyListeners(event)
	}

	return nil
}

func getEventType(tier StorageTier) StorageEventType {
	switch tier {
	case TierCold:
		return EventMovedToCold
	case TierHot:
		return EventMovedToHot
	case TierArchive:
		return EventMovedToArchive
	default:
		return EventClassChanged
	}
}
