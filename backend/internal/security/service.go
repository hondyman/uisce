package security

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// AccessRuleFilters represents query parameters for listing rules.
type AccessRuleFilters struct {
	TenantID         string
	BusinessObjectID string
	Status           string
}

// DslValidationResult represents the output of DSL validation.
type DslValidationResult struct {
	Valid        bool    `json:"valid"`
	SqlPredicate *string `json:"sqlPredicate"`
	ErrorMessage *string `json:"errorMessage"`
}

// AccessRuleImpact represents downstream impact of a rule.
type AccessRuleImpact struct {
	RuleID           string   `json:"ruleId"`
	BusinessObjectID string   `json:"businessObjectId"`
	SemanticTerms    []string `json:"semanticTerms"`
	Apis             []string `json:"apis"`
	BiArtifacts      []string `json:"biArtifacts"`
	AiArtifacts      []string `json:"aiArtifacts"`
}

// AccessRuleService encapsulates business logic for access rules.
type AccessRuleService struct {
	repo      *AccessRuleRepository
	validator *DslValidator
	analyzer  *ImpactAnalyzer
}

// NewAccessRuleService creates a new service.
func NewAccessRuleService(db *sqlx.DB) *AccessRuleService {
	repo := NewAccessRuleRepository(db)
	validator := NewDslValidator(db)
	analyzer := NewImpactAnalyzer(db, repo)

	return &AccessRuleService{
		repo:      repo,
		validator: validator,
		analyzer:  analyzer,
	}
}

// List retrieves access rules with optional filters.
func (s *AccessRuleService) List(ctx context.Context, filters AccessRuleFilters) ([]*models.AccessRule, error) {
	return s.repo.List(ctx, filters.TenantID, filters.BusinessObjectID, filters.Status)
}

// Get retrieves a single access rule by ID.
func (s *AccessRuleService) Get(ctx context.Context, ruleID string) (*models.AccessRule, error) {
	return s.repo.Get(ctx, ruleID)
}

// Create validates and creates a new access rule.
func (s *AccessRuleService) Create(ctx context.Context, rule *models.AccessRule) (*models.AccessRule, error) {
	// Validate DSL if present
	if rule.RowFilterDsl != "" {
		result, err := s.validator.Validate(ctx, rule.BusinessObjectID, rule.RowFilterDsl)
		if err != nil {
			return nil, fmt.Errorf("DSL validation failed: %w", err)
		}
		if !result.Valid {
			return nil, fmt.Errorf("invalid DSL: %s", *result.ErrorMessage)
		}
	}

	// Set defaults
	if rule.Status == "" {
		rule.Status = "DRAFT"
	}
	if rule.AppliesToApis == nil {
		defaultTrue := true
		rule.AppliesToApis = &defaultTrue
	}
	if rule.AppliesToBi == nil {
		defaultTrue := true
		rule.AppliesToBi = &defaultTrue
	}
	if rule.AppliesToAi == nil {
		defaultTrue := true
		rule.AppliesToAi = &defaultTrue
	}

	// Create rule in transaction
	tx, err := s.repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	created, err := s.repo.CreateTx(ctx, tx, rule)
	if err != nil {
		return nil, err
	}

	// Publish audit event (non-blocking via outbox)
	auditEvent := events.SecurityAuditEvent{
		EventID:          uuid.New().String(),
		EventType:        "rule.created",
		TenantID:         created.TenantID,
		RuleID:           created.RuleID,
		BusinessObjectID: created.BusinessObjectID,
		GroupDN:          created.GroupDn,
		AccessLevel:      created.AccessLevel,
		ActorID:          created.CreatedBy,
		Timestamp:        time.Now(),
		NewValue:         ruleToMap(created),
		Environment:      getEnvironment(),
	}
	if err := events.PublishSecurityAuditEvent(ctx, tx, auditEvent); err != nil {
		return nil, fmt.Errorf("publish audit event: %w", err)
	}

	// Publish snapshot event for Iceberg
	snapshotEvent := buildSnapshotEvent(created)
	if err := events.PublishSecuritySnapshotEvent(ctx, tx, snapshotEvent); err != nil {
		return nil, fmt.Errorf("publish snapshot event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

// Update validates and updates an existing access rule.
func (s *AccessRuleService) Update(ctx context.Context, ruleID string, rule *models.AccessRule) (*models.AccessRule, error) {
	// Get old value for audit
	oldRule, err := s.repo.Get(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get existing rule: %w", err)
	}

	// Validate DSL if present
	if rule.RowFilterDsl != "" {
		result, err := s.validator.Validate(ctx, rule.BusinessObjectID, rule.RowFilterDsl)
		if err != nil {
			return nil, fmt.Errorf("DSL validation failed: %w", err)
		}
		if !result.Valid {
			return nil, fmt.Errorf("invalid DSL: %s", *result.ErrorMessage)
		}
	}

	// Update in transaction
	tx, err := s.repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	updated, err := s.repo.UpdateTx(ctx, tx, ruleID, rule)
	if err != nil {
		return nil, err
	}

	// Publish audit event
	auditEvent := events.SecurityAuditEvent{
		EventID:          uuid.New().String(),
		EventType:        "rule.updated",
		TenantID:         updated.TenantID,
		RuleID:           updated.RuleID,
		BusinessObjectID: updated.BusinessObjectID,
		GroupDN:          updated.GroupDn,
		AccessLevel:      updated.AccessLevel,
		ActorID:          updated.UpdatedBy,
		Timestamp:        time.Now(),
		OldValue:         ruleToMap(oldRule),
		NewValue:         ruleToMap(updated),
		Environment:      getEnvironment(),
	}
	if err := events.PublishSecurityAuditEvent(ctx, tx, auditEvent); err != nil {
		return nil, fmt.Errorf("publish audit event: %w", err)
	}

	// Publish snapshot event
	snapshotEvent := buildSnapshotEvent(updated)
	if err := events.PublishSecuritySnapshotEvent(ctx, tx, snapshotEvent); err != nil {
		return nil, fmt.Errorf("publish snapshot event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return updated, nil
}

// ValidateDsl validates a row filter DSL expression.
func (s *AccessRuleService) ValidateDsl(ctx context.Context, businessObjectID, dsl string) (*DslValidationResult, error) {
	if dsl == "" {
		return &DslValidationResult{Valid: true}, nil
	}
	return s.validator.Validate(ctx, businessObjectID, dsl)
}

// GetImpact computes the downstream impact of an access rule.
func (s *AccessRuleService) GetImpact(ctx context.Context, ruleID string) (*AccessRuleImpact, error) {
	return s.analyzer.ComputeImpact(ctx, ruleID)
}
