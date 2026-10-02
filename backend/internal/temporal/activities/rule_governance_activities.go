package activities

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type RuleGovernanceActivities struct {
	DB     *sqlx.DB
	Logger *zap.SugaredLogger
}

func NewRuleGovernanceActivities(db *sqlx.DB, logger *zap.SugaredLogger) *RuleGovernanceActivities {
	return &RuleGovernanceActivities{
		DB:     db,
		Logger: logger,
	}
}

type ApprovalActivityInput struct {
	TenantID   string `json:"tenant_id"`
	RuleNodeID string `json:"rule_node_id"`
	ApproverID string `json:"approver_id"`
}

type RejectionActivityInput struct {
	TenantID   string `json:"tenant_id"`
	RuleNodeID string `json:"rule_node_id"`
	RejecterID string `json:"rejecter_id"`
	Reason     string `json:"reason"`
}

type PublishVersionActivityInput struct {
	TenantID      string    `json:"tenant_id"`
	RuleNodeID    string    `json:"rule_node_id"`
	PublisherID   string    `json:"publisher_id"`
	EffectiveFrom time.Time `json:"effective_from"`
}

type PublishVersionResult struct {
	Version  int    `json:"version"`
	Checksum string `json:"checksum"`
}

type EscalateReviewInput struct {
	TenantID   string `json:"tenant_id"`
	RuleNodeID string `json:"rule_node_id"`
	Reason     string `json:"reason"`
}

// RecordApprovalActivity enforces four-eyes segregation (author != approver)
// and updates the rule's governance status to 'approved'.
func (a *RuleGovernanceActivities) RecordApprovalActivity(ctx context.Context, input ApprovalActivityInput) error {
	ruleUUID, err := uuid.Parse(input.RuleNodeID)
	if err != nil {
		return fmt.Errorf("invalid rule_node_id %q: %w", input.RuleNodeID, err)
	}

	var row struct {
		Properties json.RawMessage `db:"properties"`
	}
	err = a.DB.GetContext(ctx, &row, `
		SELECT properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, ruleUUID, input.TenantID)
	if err != nil {
		return fmt.Errorf("fetch rule %s: %w", input.RuleNodeID, err)
	}

	props, err := models.ParseValidationRuleProperties(row.Properties)
	if err != nil {
		return fmt.Errorf("parse properties: %w", err)
	}

	// Four-eyes check: author != approver
	if props.AuthorID != "" && props.AuthorID == input.ApproverID {
		return fmt.Errorf("four-eyes governance violation: author %q cannot approve their own rule", props.AuthorID)
	}

	props.GovernanceStatus = models.ValidationRuleGovernanceApproved
	props.ApprovedBy = input.ApproverID
	now := time.Now().UTC()
	props.ApprovedAt = &now

	updatedProps, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("marshal updated properties: %w", err)
	}

	_, err = a.DB.ExecContext(ctx, `
		UPDATE catalog_node
		SET properties = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3::uuid
	`, updatedProps, ruleUUID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update rule governance status: %w", err)
	}

	if a.Logger != nil {
		a.Logger.Infof("[RuleGovernance] Rule %s approved by %s", input.RuleNodeID, input.ApproverID)
	}
	return nil
}

// RecordRejectionActivity records a rejection decision and reason.
func (a *RuleGovernanceActivities) RecordRejectionActivity(ctx context.Context, input RejectionActivityInput) error {
	ruleUUID, err := uuid.Parse(input.RuleNodeID)
	if err != nil {
		return fmt.Errorf("invalid rule_node_id %q: %w", input.RuleNodeID, err)
	}

	var row struct {
		Properties json.RawMessage `db:"properties"`
	}
	err = a.DB.GetContext(ctx, &row, `
		SELECT properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, ruleUUID, input.TenantID)
	if err != nil {
		return fmt.Errorf("fetch rule %s: %w", input.RuleNodeID, err)
	}

	props, err := models.ParseValidationRuleProperties(row.Properties)
	if err != nil {
		return fmt.Errorf("parse properties: %w", err)
	}

	props.GovernanceStatus = models.ValidationRuleGovernanceRejected
	props.RejectionReason = input.Reason
	props.RejectedBy = input.RejecterID
	now := time.Now().UTC()
	props.RejectedAt = &now

	updatedProps, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("marshal updated properties: %w", err)
	}

	_, err = a.DB.ExecContext(ctx, `
		UPDATE catalog_node
		SET properties = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3::uuid
	`, updatedProps, ruleUUID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update rule rejection: %w", err)
	}

	if a.Logger != nil {
		a.Logger.Infof("[RuleGovernance] Rule %s rejected by %s: %s", input.RuleNodeID, input.RejecterID, input.Reason)
	}
	return nil
}

// PublishVersionActivity creates a new row in validation_rule_versions,
// updates catalog_node to 'published', and automatically fires the catalog_changed trigger.
func (a *RuleGovernanceActivities) PublishVersionActivity(ctx context.Context, input PublishVersionActivityInput) (*PublishVersionResult, error) {
	ruleUUID, err := uuid.Parse(input.RuleNodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid rule_node_id %q: %w", input.RuleNodeID, err)
	}

	var row struct {
		Config     json.RawMessage `db:"config"`
		Properties json.RawMessage `db:"properties"`
	}
	err = a.DB.GetContext(ctx, &row, `
		SELECT config, properties FROM catalog_node
		WHERE id = $1 AND tenant_id = $2::uuid
	`, ruleUUID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch rule %s: %w", input.RuleNodeID, err)
	}

	var cfg models.ValidationRuleConfig
	if err := json.Unmarshal(row.Config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	var node vm.RuleNode
	if err := json.Unmarshal(cfg.RuleAST, &node); err != nil {
		return nil, fmt.Errorf("unmarshal rule AST: %w", err)
	}

	compactedAST, err := vm.Compact(node)
	if err != nil {
		return nil, fmt.Errorf("canonicalize AST: %w", err)
	}
	checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(compactedAST))

	effFrom := input.EffectiveFrom
	if effFrom.IsZero() {
		effFrom = time.Now().UTC()
	}

	tx, err := a.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Monotonic version calculation
	var currentMaxVersion int
	_ = tx.GetContext(ctx, &currentMaxVersion, `
		SELECT COALESCE(MAX(version), 0) FROM validation_rule_versions
		WHERE rule_node_id = $1 AND tenant_id = $2::uuid
	`, ruleUUID, input.TenantID)

	nextVersion := currentMaxVersion + 1

	// Close out previous version's effective_to if active
	_, err = tx.ExecContext(ctx, `
		UPDATE validation_rule_versions
		SET effective_to = $1, superseded_at = NOW()
		WHERE rule_node_id = $2 AND tenant_id = $3::uuid AND effective_to IS NULL
	`, effFrom, ruleUUID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("close previous version: %w", err)
	}

	props, _ := models.ParseValidationRuleProperties(row.Properties)
	props.GovernanceStatus = models.ValidationRuleGovernancePublished
	props.PublishedBy = input.PublisherID
	now := time.Now().UTC()
	props.PublishedAt = &now
	props.Version = nextVersion

	updatedProps, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("marshal published properties: %w", err)
	}

	// Insert into validation_rule_versions
	_, err = tx.ExecContext(ctx, `
		INSERT INTO validation_rule_versions
			(id, rule_node_id, tenant_id, version, rule_ast, properties, checksum, effective_from, effective_to, recorded_at, governance_status, published_by, parent_version)
		VALUES
			(gen_random_uuid(), $1, $2::uuid, $3, $4, $5, $6, $7, NULL, NOW(), 'published', $8, $9)
	`, ruleUUID, input.TenantID, nextVersion, compactedAST, updatedProps, checksum, effFrom, input.PublisherID, currentMaxVersion)
	if err != nil {
		return nil, fmt.Errorf("insert validation_rule_versions: %w", err)
	}

	// Update catalog_node — fires trg_catalog_node_notify trigger
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_node
		SET properties = $1, is_active = true, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3::uuid
	`, updatedProps, ruleUUID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("update catalog_node to published: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit publish transaction: %w", err)
	}

	if a.Logger != nil {
		a.Logger.Infof("[RuleGovernance] Rule %s published version %d (checksum %s)", input.RuleNodeID, nextVersion, checksum)
	}

	return &PublishVersionResult{
		Version:  nextVersion,
		Checksum: checksum,
	}, nil
}

// EscalateReviewActivity logs or notifies review SLA breaches.
func (a *RuleGovernanceActivities) EscalateReviewActivity(ctx context.Context, input EscalateReviewInput) error {
	if a.Logger != nil {
		a.Logger.Warnf("[RuleGovernance] REVIEW SLA BREACH: rule %s tenant %s: %s", input.RuleNodeID, input.TenantID, input.Reason)
	}
	return nil
}
