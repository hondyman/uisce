package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// AutomationService handles self-healing and proactive governance tasks.
type AutomationService struct {
	db            *sqlx.DB
	collabService *CollaborationService
	// In a real system, this would be a proper job scheduler.
	isPaused bool
}

// NewAutomationService creates a new AutomationService.
func NewAutomationService(db *sqlx.DB, collabService *CollaborationService) *AutomationService {
	return &AutomationService{
		db:            db,
		collabService: collabService,
		isPaused:      false,
	}
}

// RunAutomationCycle triggers a full automation cycle. Mock implementation.
func (s *AutomationService) RunAutomationCycle(ctx context.Context, actorID string) ([]models.AutomationLog, error) {
	if s.isPaused {
		fmt.Println("Automation cycle skipped: service is paused.")
		return nil, nil
	}

	fmt.Printf("Automation cycle triggered by %s\n", actorID)
	var allLogs []models.AutomationLog

	// 1. Auto-expire drifted claims
	driftLogs, err := s.runDriftExpiration(ctx)
	if err != nil {
		fmt.Printf("Error during drift expiration: %v\n", err)
	} else {
		allLogs = append(allLogs, driftLogs...)
	}

	// 2. Auto-resolve low-risk conflicts
	conflictLogs, err := s.runConflictResolution(ctx)
	if err != nil {
		fmt.Printf("Error during conflict resolution: %v\n", err)
	} else {
		allLogs = append(allLogs, conflictLogs...)
	}

	return allLogs, nil
}

// runDriftExpiration finds and expires inactive claims.
func (s *AutomationService) runDriftExpiration(ctx context.Context) ([]models.AutomationLog, error) {
	driftedClaims, err := s.collabService.DetectClaimDrift(ctx)
	if err != nil {
		return nil, err
	}

	var logs []models.AutomationLog
	for _, claim := range driftedClaims {
		s.collabService.RevokeDirectClaim(ctx, claim.ID, "automation_engine")
		details, _ := json.Marshal(map[string]interface{}{"user_id": claim.UserID, "model_id": claim.ModelID, "reason": fmt.Sprintf("Claim inactive since %v", claim.LastUsedAt)})
		log := models.AutomationLog{ID: uuid.New(), Timestamp: time.Now(), PolicyID: "auto_expire_drifted_claims_60_days", Action: "claim_expired", TargetType: "claim", TargetID: claim.ID.String(), Details: details, Status: "success"}
		logs = append(logs, log)
		fmt.Printf("Automation: Expired drifted claim %s for user %s\n", claim.ID, claim.UserID)
	}
	return logs, nil
}

// runConflictResolution finds and resolves redundant claims.
func (s *AutomationService) runConflictResolution(ctx context.Context) ([]models.AutomationLog, error) {
	userID := "patrick"
	conflicts, err := s.collabService.DetectClaimConflicts(ctx, userID)
	if err != nil {
		return nil, err
	}

	var logs []models.AutomationLog
	for _, conflict := range conflicts {
		if conflict.ConflictType == "contradiction" {
			s.collabService.ResolveClaimConflict(ctx, conflict.ID, "auto_consolidate_to_highest_permission", "automation_engine")
			details, _ := json.Marshal(map[string]interface{}{"user_id": conflict.UserID, "model_id": conflict.ModelID, "description": "Resolved redundant claims by consolidating to highest permission.", "conflict_details": conflict.Details})
			log := models.AutomationLog{ID: uuid.New(), Timestamp: time.Now(), PolicyID: "auto_resolve_redundant_claims", Action: "conflict_resolved", TargetType: "user_claim_group", TargetID: fmt.Sprintf("%s-%s", conflict.UserID, conflict.ModelID), Details: details, Status: "success"}
			logs = append(logs, log)
			fmt.Printf("Automation: Resolved claim conflict %s for user %s\n", conflict.ID, conflict.UserID)
		}
	}
	return logs, nil
}

// ListAutomationPolicies retrieves all automation policies. Mock implementation.
func (s *AutomationService) ListAutomationPolicies(ctx context.Context) ([]models.AutomationPolicy, error) {
	conditions1, _ := json.Marshal(map[string]int{"inactive_days": 60})
	conditions2, _ := json.Marshal(map[string]string{"type": "redundant_read_claim"})
	return []models.AutomationPolicy{
		{ID: uuid.New(), PolicyID: "auto_expire_drifted_claims_60_days", Description: "Automatically expire direct claims that have not been used for 60 days.", Trigger: "daily_schedule", Conditions: conditions1, Action: "auto_expire", IsEnabled: true, UpdatedAt: time.Now().Add(-5 * 24 * time.Hour)},
		{ID: uuid.New(), PolicyID: "auto_resolve_redundant_claims", Description: "Automatically resolve conflicts where a user has the same permission from a role and a direct grant.", Trigger: "claim_changed", Conditions: conditions2, Action: "auto_consolidate", IsEnabled: true, UpdatedAt: time.Now().Add(-10 * 24 * time.Hour)},
	}, nil
}

// ListAutomationLogs retrieves recent automation logs. Mock implementation.
func (s *AutomationService) ListAutomationLogs(ctx context.Context) ([]models.AutomationLog, error) {
	details1, _ := json.Marshal(map[string]string{"reason": "Claim inactive for 62 days", "user_id": "old_employee"})
	return []models.AutomationLog{{ID: uuid.New(), Timestamp: time.Now().Add(-2 * time.Hour), PolicyID: "auto_expire_drifted_claims_60_days", Action: "claim_expired", TargetType: "claim", TargetID: uuid.New().String(), Details: details1, Status: "success"}}, nil
}

// PauseAutomation temporarily disables the automation engine.
func (s *AutomationService) PauseAutomation(ctx context.Context) error {
	s.isPaused = true
	fmt.Println("Automation engine PAUSED.")
	return nil
}

// ResumeAutomation re-enables the automation engine.
func (s *AutomationService) ResumeAutomation(ctx context.Context) error {
	s.isPaused = false
	fmt.Println("Automation engine RESUMED.")
	return nil
}

// GetStatus returns the current status of the automation engine.
func (s *AutomationService) GetStatus() string {
	if s.isPaused {
		return "paused"
	}
	return "running"
}
