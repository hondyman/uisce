package aso

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Semantic ChangeSet Types
// ============================================================================

// SemanticChangeSet represents a promotion request containing semantic changes
type SemanticChangeSet struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	TenantID    *uuid.UUID       `json:"tenant_id,omitempty" db:"tenant_id"`
	SourceEnv   string           `json:"source_env" db:"source_env"`
	TargetEnv   string           `json:"target_env" db:"target_env"`
	Status      ChangeSetStatus  `json:"status" db:"status"`
	Changes     []SemanticChange `json:"changes" db:"-"`
	ChangesJSON json.RawMessage  `json:"-" db:"changes_json"`
	Description string           `json:"description" db:"description"`
	CreatedBy   string           `json:"created_by" db:"created_by"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	ApprovedBy  *string          `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt  *time.Time       `json:"approved_at,omitempty" db:"approved_at"`
	AppliedAt   *time.Time       `json:"applied_at,omitempty" db:"applied_at"`
	RejectedBy  *string          `json:"rejected_by,omitempty" db:"rejected_by"`
	RejectedAt  *time.Time       `json:"rejected_at,omitempty" db:"rejected_at"`

	// ASO-related fields
	ASOValidation *ASOValidationResult `json:"aso_validation,omitempty" db:"-"`
	ASOSource     bool                 `json:"aso_source" db:"aso_source"` // True if ASO-generated
}

// ChangeSetStatus represents the lifecycle of a changeset
type ChangeSetStatus string

const (
	ChangeSetDraft             ChangeSetStatus = "draft"
	ChangeSetPendingValidation ChangeSetStatus = "pending_validation"
	ChangeSetValidated         ChangeSetStatus = "validated"
	ChangeSetPendingApproval   ChangeSetStatus = "pending_approval"
	ChangeSetApproved          ChangeSetStatus = "approved"
	ChangeSetApplied           ChangeSetStatus = "applied"
	ChangeSetRejected          ChangeSetStatus = "rejected"
	ChangeSetFailed            ChangeSetStatus = "failed"
)

// SemanticChange represents a single change in a changeset
type SemanticChange struct {
	ID         uuid.UUID       `json:"id"`
	Type       ChangeType      `json:"type"`
	TargetType TargetType      `json:"target_type"`
	TargetID   uuid.UUID       `json:"target_id"`
	TargetName string          `json:"target_name"`
	Action     ChangeAction    `json:"action"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`

	// ASO metadata
	ASOOptimizationID *uuid.UUID `json:"aso_optimization_id,omitempty"`
}

// ChangeType categorizes the semantic change
type ChangeType string

const (
	ChangeTypeBO           ChangeType = "business_object"
	ChangeTypeCalc         ChangeType = "calculation"
	ChangeTypeTerm         ChangeType = "term"
	ChangeTypePreAgg       ChangeType = "preagg"
	ChangeTypeRelationship ChangeType = "relationship"
)

// ChangeAction represents create/update/delete
type ChangeAction string

const (
	ChangeActionCreate ChangeAction = "create"
	ChangeActionUpdate ChangeAction = "update"
	ChangeActionDelete ChangeAction = "delete"
)

// ============================================================================
// Promotion Service
// ============================================================================

// PromotionService handles semantic model promotions with ASO integration
type PromotionService interface {
	// CreateChangeSet creates a new changeset for promotion
	CreateChangeSet(ctx context.Context, cs *SemanticChangeSet) error

	// ValidateChangeSet runs validation including ASO performance checks
	ValidateChangeSet(ctx context.Context, csID uuid.UUID) (*ASOValidationResult, error)

	// ApproveChangeSet marks a changeset as approved
	ApproveChangeSet(ctx context.Context, csID uuid.UUID, approver string) error

	// ApplyChangeSet applies the changeset to the target environment
	ApplyChangeSet(ctx context.Context, csID uuid.UUID, applier string) error

	// RejectChangeSet rejects a changeset
	RejectChangeSet(ctx context.Context, csID uuid.UUID, rejector, reason string) error

	// CreateASOChangeSet creates a changeset from ASO optimizations
	CreateASOChangeSet(ctx context.Context, env string, optIDs []uuid.UUID, creator string) (*SemanticChangeSet, error)

	// GetChangeSet retrieves a changeset by ID
	GetChangeSet(ctx context.Context, csID uuid.UUID) (*SemanticChangeSet, error)

	// ListChangeSets lists changesets with filters
	ListChangeSets(ctx context.Context, filter ChangeSetFilter) ([]SemanticChangeSet, error)
}

// ChangeSetFilter for listing changesets
type ChangeSetFilter struct {
	TenantID  *uuid.UUID
	Status    *ChangeSetStatus
	SourceEnv *string
	TargetEnv *string
	ASOSource *bool
	Limit     int
	Offset    int
}

// promotionService implements PromotionService
type promotionService struct {
	db        *sqlx.DB
	asoEngine ASOEngine
	optRepo   ASOOptimizationRepository
}

// NewPromotionService creates a new promotion service
func NewPromotionService(
	db *sqlx.DB,
	asoEngine ASOEngine,
	optRepo ASOOptimizationRepository,
) PromotionService {
	return &promotionService{
		db:        db,
		asoEngine: asoEngine,
		optRepo:   optRepo,
	}
}

// CreateChangeSet creates a new changeset for promotion
func (s *promotionService) CreateChangeSet(ctx context.Context, cs *SemanticChangeSet) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}
	cs.Status = ChangeSetDraft
	cs.CreatedAt = time.Now()

	// Serialize changes
	changesJSON, err := json.Marshal(cs.Changes)
	if err != nil {
		return fmt.Errorf("failed to serialize changes: %w", err)
	}
	cs.ChangesJSON = changesJSON

	query := `
		INSERT INTO semantic.changeset (
			id, tenant_id, source_env, target_env, status,
			changes_json, description, created_by, created_at, aso_source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err = s.db.ExecContext(ctx, query,
		cs.ID, cs.TenantID, cs.SourceEnv, cs.TargetEnv, cs.Status,
		cs.ChangesJSON, cs.Description, cs.CreatedBy, cs.CreatedAt, cs.ASOSource,
	)

	return err
}

// ValidateChangeSet runs validation including ASO performance checks
func (s *promotionService) ValidateChangeSet(ctx context.Context, csID uuid.UUID) (*ASOValidationResult, error) {
	cs, err := s.GetChangeSet(ctx, csID)
	if err != nil {
		return nil, err
	}

	// Update status to pending validation
	_, err = s.db.ExecContext(ctx, `
		UPDATE semantic.changeset SET status = $1 WHERE id = $2
	`, ChangeSetPendingValidation, csID)
	if err != nil {
		return nil, err
	}

	// Run ASO validation
	result, err := s.asoEngine.ValidateChangeSet(ctx, csID)
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `
			UPDATE semantic.changeset SET status = $1 WHERE id = $2
		`, ChangeSetFailed, csID)
		return nil, err
	}

	// Update with validation result
	status := ChangeSetValidated
	if !result.Valid || len(result.Errors) > 0 {
		status = ChangeSetPendingApproval // Needs manual review
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE semantic.changeset SET status = $1 WHERE id = $2
	`, status, csID)

	cs.ASOValidation = result
	return result, err
}

// ApproveChangeSet marks a changeset as approved
func (s *promotionService) ApproveChangeSet(ctx context.Context, csID uuid.UUID, approver string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.changeset 
		SET status = $1, approved_by = $2, approved_at = $3
		WHERE id = $4 AND status IN ('validated', 'pending_approval')
	`, ChangeSetApproved, approver, now, csID)
	return err
}

// ApplyChangeSet applies the changeset to the target environment
func (s *promotionService) ApplyChangeSet(ctx context.Context, csID uuid.UUID, applier string) error {
	cs, err := s.GetChangeSet(ctx, csID)
	if err != nil {
		return err
	}

	if cs.Status != ChangeSetApproved {
		return fmt.Errorf("changeset must be approved before applying")
	}

	// Apply each change
	for _, change := range cs.Changes {
		if err := s.applyChange(ctx, cs.TargetEnv, change); err != nil {
			_, _ = s.db.ExecContext(ctx, `
				UPDATE semantic.changeset SET status = $1 WHERE id = $2
			`, ChangeSetFailed, csID)
			return fmt.Errorf("failed to apply change %s: %w", change.ID, err)
		}

		// If this change is from ASO, mark the optimization as applied
		if change.ASOOptimizationID != nil {
			_ = s.optRepo.MarkApplied(ctx, *change.ASOOptimizationID, applier, change.After)
		}
	}

	// Mark as applied
	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		UPDATE semantic.changeset SET status = $1, applied_at = $2 WHERE id = $3
	`, ChangeSetApplied, now, csID)

	// Trigger ASO post-promotion optimization
	go func() {
		tenantID := ""
		if cs.TenantID != nil {
			tenantID = cs.TenantID.String()
		}
		s.asoEngine.EvaluateTenant(context.Background(), cs.TargetEnv, tenantID)
	}()

	return err
}

// RejectChangeSet rejects a changeset
func (s *promotionService) RejectChangeSet(ctx context.Context, csID uuid.UUID, rejector, reason string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.changeset 
		SET status = $1, rejected_by = $2, rejected_at = $3
		WHERE id = $4
	`, ChangeSetRejected, rejector, now, csID)
	return err
}

// CreateASOChangeSet creates a changeset from ASO optimizations
func (s *promotionService) CreateASOChangeSet(ctx context.Context, env string, optIDs []uuid.UUID, creator string) (*SemanticChangeSet, error) {
	var changes []SemanticChange

	for _, optID := range optIDs {
		opt, err := s.optRepo.GetByID(ctx, optID)
		if err != nil || opt == nil {
			continue
		}

		change := SemanticChange{
			ID:                uuid.New(),
			Type:              changeTypeFromTargetType(opt.TargetType),
			TargetType:        opt.TargetType,
			TargetID:          opt.TargetID,
			TargetName:        opt.TargetName,
			Action:            actionFromOptType(opt.OptimizationType),
			After:             opt.Details,
			ASOOptimizationID: &opt.ID,
		}
		changes = append(changes, change)
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no valid optimizations found")
	}

	cs := &SemanticChangeSet{
		ID:          uuid.New(),
		SourceEnv:   env,
		TargetEnv:   env, // ASO changesets apply to same env
		Status:      ChangeSetDraft,
		Changes:     changes,
		Description: fmt.Sprintf("ASO-generated optimization bundle (%d changes)", len(changes)),
		CreatedBy:   creator,
		ASOSource:   true,
	}

	if err := s.CreateChangeSet(ctx, cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// GetChangeSet retrieves a changeset by ID
func (s *promotionService) GetChangeSet(ctx context.Context, csID uuid.UUID) (*SemanticChangeSet, error) {
	var cs SemanticChangeSet
	err := s.db.GetContext(ctx, &cs, `
		SELECT * FROM semantic.changeset WHERE id = $1
	`, csID)
	if err != nil {
		return nil, err
	}

	// Deserialize changes
	if cs.ChangesJSON != nil {
		_ = json.Unmarshal(cs.ChangesJSON, &cs.Changes)
	}

	return &cs, nil
}

// ListChangeSets lists changesets with filters
func (s *promotionService) ListChangeSets(ctx context.Context, filter ChangeSetFilter) ([]SemanticChangeSet, error) {
	query := `SELECT * FROM semantic.changeset WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
		args = append(args, *filter.TenantID)
		argNum++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.SourceEnv != nil {
		query += fmt.Sprintf(" AND source_env = $%d", argNum)
		args = append(args, *filter.SourceEnv)
		argNum++
	}
	if filter.TargetEnv != nil {
		query += fmt.Sprintf(" AND target_env = $%d", argNum)
		args = append(args, *filter.TargetEnv)
		argNum++
	}
	if filter.ASOSource != nil {
		query += fmt.Sprintf(" AND aso_source = $%d", argNum)
		args = append(args, *filter.ASOSource)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var changesets []SemanticChangeSet
	err := s.db.SelectContext(ctx, &changesets, query, args...)
	return changesets, err
}

// applyChange applies a single change to the target environment
func (s *promotionService) applyChange(ctx context.Context, env string, change SemanticChange) error {
	// This would integrate with your semantic model service to apply changes
	// For now, this is a placeholder that would call the appropriate service
	switch change.Type {
	case ChangeTypeBO:
		// Call BO service
	case ChangeTypeCalc:
		// Call calculation service
	case ChangeTypePreAgg:
		// Call pre-agg service
	case ChangeTypeTerm:
		// Call term service
	case ChangeTypeRelationship:
		// Call relationship service
	}
	return nil
}

// Helper functions
func changeTypeFromTargetType(tt TargetType) ChangeType {
	switch tt {
	case TargetTypeBO:
		return ChangeTypeBO
	case TargetTypeCalc:
		return ChangeTypeCalc
	case TargetTypePreAgg:
		return ChangeTypePreAgg
	case TargetTypeTerm:
		return ChangeTypeTerm
	default:
		return ChangeTypeBO
	}
}

func actionFromOptType(ot OptimizationType) ChangeAction {
	switch ot {
	case OptTypeCreatePreAgg:
		return ChangeActionCreate
	case OptTypeRetireAsset:
		return ChangeActionDelete
	default:
		return ChangeActionUpdate
	}
}
