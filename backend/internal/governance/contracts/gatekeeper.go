package contracts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/agentic"
	"github.com/jmoiron/sqlx"
)

// Gatekeeper evaluates proposed DDL changes against the semantic catalog.
type Gatekeeper struct {
	db           *sqlx.DB
	mcService    *agentic.MakerCheckerService
	openTickets  bool
}

// NewGatekeeper constructs a Gatekeeper. Pass nil for mcService to disable ticket auto-opening.
func NewGatekeeper(db *sqlx.DB, mcService *agentic.MakerCheckerService) *Gatekeeper {
	return &Gatekeeper{
		db:          db,
		mcService:   mcService,
		openTickets: os.Getenv("CONTRACT_GATEWAY_OPEN_TICKETS") == "true",
	}
}

// Validate evaluates the proposed diffs and returns a full validation response.
func (g *Gatekeeper) Validate(ctx context.Context, req *ContractValidationRequest) (*ContractValidationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if len(req.ProposedDiffs) == 0 {
		return &ContractValidationResponse{
			RequestID:   uuid.New().String(),
			TenantID:    req.TenantID,
			AllSafe:     true,
			HasCritical: false,
			EvaluatedAt: time.Now(),
		}, nil
	}

	resp := &ContractValidationResponse{
		RequestID:   uuid.New().String(),
		TenantID:    req.TenantID,
		Results:     make([]ValidationResult, 0, len(req.ProposedDiffs)),
		EvaluatedAt: time.Now(),
	}

	for _, diff := range req.ProposedDiffs {
		result := g.evaluateTableDiff(ctx, req.TenantID, diff)
		resp.Results = append(resp.Results, result)
		if result.Severity == SeverityCritical {
			resp.HasCritical = true
		}
		if result.SafeToApply {
			resp.SafeCount++
		} else {
			resp.ViolationsCount += len(result.Violations)
		}
	}

	resp.AllSafe = !resp.HasCritical

	downstreamBOs, _ := g.traverseDownstreamBOs(ctx, req.TenantID, req.ProposedDiffs)
	resp.DownstreamBOs = downstreamBOs

	if resp.HasCritical && g.openTickets && g.mcService != nil {
		ticketID, err := g.openComplianceTicket(ctx, req, resp)
		if err != nil {
			log.Printf("[Gatekeeper] failed to open compliance ticket: %v", err)
		} else {
			resp.TicketID = ticketID
		}
	}

	return resp, nil
}

// evaluateTableDiff classifies all column changes in a single table diff.
func (g *Gatekeeper) evaluateTableDiff(ctx context.Context, tenantID string, diff TableDiff) ValidationResult {
	result := ValidationResult{
		TableName:    diff.TableName,
		Violations:   make([]Violation, 0),
		SafeToApply:  true,
		Severity:     SeveritySafe,
	}

	for _, col := range diff.Columns {
		violations := g.classifyChange(ctx, tenantID, diff.TableName, diff.DatasourceID, col)
		for _, v := range violations {
			result.Violations = append(result.Violations, v)
			if v.Severity == SeverityCritical {
				result.SafeToApply = false
				result.Severity = SeverityCritical
			}
		}
	}

	return result
}

// classifyChange determines whether a column change breaks any contract rule.
// It queries bo_field_def and catalog_edge to determine impact.
func (g *Gatekeeper) classifyChange(ctx context.Context, tenantID, tableName, datasourceID string, col ColumnDiff) []Violation {
	var violations []Violation

	switch col.ChangeKind {
	case ColumnDropped:
		violations = append(violations, g.checkDroppedColumn(ctx, tenantID, tableName, datasourceID, col.ColumnName)...)

	case ColumnRenamed:
		violations = append(violations, Violation{
			Type:        ViolationBusinessKeyAltered,
			Severity:    SeverityCritical,
			Column:      col.ColumnName,
			Description: fmt.Sprintf("Column '%s' is being renamed; downstream BOs and semantic terms referencing it will break.", col.ColumnName),
		})

	case ColumnTypeChanged:
		if !typeCompatible(col.OldType, col.NewType) {
			violations = append(violations, Violation{
				Type:        ViolationTypeIncompatible,
				Severity:    SeverityCritical,
				Column:      col.ColumnName,
				OldValue:    col.OldType,
				NewValue:    col.NewType,
				Description: fmt.Sprintf("Column '%s' type changed from '%s' to '%s' which is not backward-compatible.", col.ColumnName, col.OldType, col.NewType),
			})
		}

	case ColumnNullabilityChanged:
		if col.OldNull != nil && !*col.OldNull && col.NewNull != nil && *col.NewNull {
			violations = append(violations, Violation{
				Type:        ViolationNonNullableToNullable,
				Severity:    SeverityCritical,
				Column:      col.ColumnName,
				Description: fmt.Sprintf("Column '%s' is being changed from NON-NULL to NULL; existing NOT NULL constraints in downstream queries may be violated.", col.ColumnName),
			})
		}

	case ColumnAdded:
		violations = append(violations, g.checkAddedColumn(ctx, tenantID, tableName, datasourceID, col)...)

	case ColumnDefaultChanged:
		// Default changes are generally safe
	}

	return violations
}

// checkDroppedColumn queries bo_field_def to see if the dropped column is a required or
// business-key field in any active BO.
func (g *Gatekeeper) checkDroppedColumn(ctx context.Context, tenantID, tableName, datasourceID, columnName string) []Violation {
	var violations []Violation

	if g.db == nil {
		return violations
	}

	query := `
		SELECT bf.field_key, bf.is_required, bo.key as bo_key
		FROM bo_fields bf
		JOIN business_objects bo ON bf.business_object_id = bo.id
		WHERE bo.tenant_id = $1
		  AND bf.technical_name = $2
		  AND bo.tenant_datasource_id = $3
		  AND bo.is_active = true
		LIMIT 20
	`
	var rows []struct {
		FieldKey   string `db:"field_key"`
		IsRequired bool   `db:"is_required"`
		BOKey     string `db:"bo_key"`
	}
	err := g.db.SelectContext(ctx, &rows, query, tenantID, columnName, datasourceID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[Gatekeeper] warning: bo_fields lookup failed: %v", err)
		return nil
	}

	for _, row := range rows {
		violations = append(violations, Violation{
			Type:        ViolationRequiredFieldDropped,
			Severity:    SeverityCritical,
			Column:      columnName,
			Description: fmt.Sprintf("Column '%s' (mapped to BO field '%s' in '%s') is being dropped; the Business Object '%s' will break.", columnName, row.FieldKey, datasourceID, row.BOKey),
		})
	}

	return violations
}

// checkAddedColumn determines if a new column is being added as NOT NULL without a default,
// which would break existing INSERT patterns.
func (g *Gatekeeper) checkAddedColumn(ctx context.Context, tenantID, tableName, datasourceID string, col ColumnDiff) []Violation {
	var violations []Violation

	if col.NewNull != nil && !*col.NewNull && col.NewDefault == "" {
		violations = append(violations, Violation{
			Type:        ViolationRequiredColumnAdded,
			Severity:    SeverityCritical,
			Column:      col.ColumnName,
			Description: fmt.Sprintf("Column '%s' is being added as NOT NULL without a DEFAULT value; existing INSERT statements will fail.", col.ColumnName),
		})
	}

	return violations
}

// traverseDownstreamBOs finds all Business Objects whose semantic bindings reference any
// of the tables being modified.
func (g *Gatekeeper) traverseDownstreamBOs(ctx context.Context, tenantID string, diffs []TableDiff) ([]string, error) {
	if g.db == nil {
		return nil, nil
	}

	tables := make([]string, len(diffs))
	for i, d := range diffs {
		tables[i] = d.TableName
	}

	placeholders := make([]string, len(tables))
	args := make([]interface{}, 0, len(tables)+1)
	args = append(args, tenantID)
	for i, t := range tables {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, t)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT bo.key
		FROM business_objects bo
		JOIN bo_fields bf ON bf.business_object_id = bo.id
		WHERE bo.tenant_id = $1
		  AND bo.is_active = true
		  AND bf.technical_name IN (%s)
		UNION
		SELECT DISTINCT bo.key
		FROM business_objects bo
		JOIN catalog_edge ce ON ce.target_id = bo.id
		JOIN catalog_node cn ON ce.source_id = cn.id
		WHERE bo.tenant_id = $1
		  AND bo.is_active = true
		  AND cn.name IN (%s)
		LIMIT 50
	`, strings.Join(placeholders, ","), strings.Join(placeholders, ","))

	var boKeys []string
	err := g.db.SelectContext(ctx, &boKeys, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("downstream BO traversal failed: %w", err)
	}

	return boKeys, nil
}

// openComplianceTicket submits a MakerChecker ticket for critical violations.
func (g *Gatekeeper) openComplianceTicket(ctx context.Context, req *ContractValidationRequest, resp *ContractValidationResponse) (string, error) {
	if g.mcService == nil {
		return "", fmt.Errorf("maker-checker service not configured")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"type":            "DATA_CONTRACT_VIOLATION",
		"request_id":      resp.RequestID,
		"diffs":           req.ProposedDiffs,
		"violations":      resp.Results,
		"downstream_bos":  resp.DownstreamBOs,
		"source":          "CONTRACT_GATEWAY",
	})

	ticketReq := agentic.ProposalRequest{
		TenantID:   req.TenantID,
		AgentID:    "CONTRACT_GATEWAY",
		TargetBOID: "data_contract_violations",
		ActionType: "SCHEMA_CHANGE_BLOCKED",
		Payload:    payload,
	}

	ticketID, err := g.mcService.SubmitAgentProposal(ctx, ticketReq)
	if err != nil {
		return "", fmt.Errorf("SubmitAgentProposal failed: %w", err)
	}

	contractID := g.computeContractID(req)
	insertQuery := `
		INSERT INTO public.data_contract_violations
		  (violation_id, tenant_id, contract_id, status, severity, target_table,
		   datasource_id, proposed_diff, violations, ticket_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, result := range resp.Results {
		violationsJSON, _ := json.Marshal(result.Violations)
		diffJSON, _ := json.Marshal(req.ProposedDiffs)
		severity := "CRITICAL"
		if result.SafeToApply {
			severity = "SAFE"
		}
		_, err := g.db.ExecContext(ctx, insertQuery,
			uuid.New().String(), req.TenantID, contractID, ContractStatusTicketed,
			severity, result.TableName, req.DatasourceID, string(diffJSON),
			string(violationsJSON), ticketID,
		)
		if err != nil {
			log.Printf("[Gatekeeper] warning: failed to persist violation record: %v", err)
		}
	}

	return ticketID, nil
}

// computeContractID produces a stable hash key for deduplication of repeated violations.
func (g *Gatekeeper) computeContractID(req *ContractValidationRequest) string {
	combined := req.TenantID + ":" + req.DatasourceID
	for _, d := range req.ProposedDiffs {
		combined += ":" + d.TableName
	}
	h := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(h[:8])
}

// ListViolations returns all violation records for a tenant, optionally filtered by status.
func (g *Gatekeeper) ListViolations(ctx context.Context, tenantID, status string, limit int) ([]ViolationRecord, error) {
	if g.db == nil {
		return nil, nil
	}
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT violation_id, tenant_id, contract_id, status, severity,
		       target_table, datasource_id, proposed_diff, violations,
		       detected_at, reviewed_by, reviewed_at, ticket_id,
		       created_at, updated_at
		FROM public.data_contract_violations
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY detected_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	var rows []ViolationRecord
	err := g.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListViolations failed: %w", err)
	}
	return rows, nil
}

// ApproveViolation marks a pending violation as approved by the data steward.
func (g *Gatekeeper) ApproveViolation(ctx context.Context, violationID, reviewerID string) error {
	if g.db == nil {
		return nil
	}
	query := `
		UPDATE public.data_contract_violations
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW()
		WHERE violation_id = $3
	`
	_, err := g.db.ExecContext(ctx, query, ContractStatusApproved, reviewerID, violationID)
	return err
}

// RejectViolation marks a pending violation as blocked by the data steward.
func (g *Gatekeeper) RejectViolation(ctx context.Context, violationID, reviewerID, reason string) error {
	if g.db == nil {
		return nil
	}
	query := `
		UPDATE public.data_contract_violations
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW()
		WHERE violation_id = $3
	`
	_, err := g.db.ExecContext(ctx, query, ContractStatusBlocked, reviewerID, violationID)
	return err
}

// typeCompatible is a best-effort backward-compatibility check between SQL types.
func typeCompatible(oldType, newType string) bool {
	oldType = strings.ToUpper(oldType)
	newType = strings.ToUpper(newType)

	if oldType == newType {
		return true
	}

	wideners := map[string][]string{
		"INTEGER":  {"BIGINT", "BIGSERIAL", "SERIAL"},
		"BIGINT":  {"NUMERIC", "DECIMAL"},
		"REAL":    {"DOUBLE PRECISION", "NUMERIC", "DECIMAL"},
		"VARCHAR": {"TEXT"},
		"CHAR":    {"VARCHAR", "TEXT"},
		"DATE":    {"TIMESTAMP", "TIMESTAMPTZ"},
		"TIMESTAMP": {"TIMESTAMPTZ"},
	}

	if widening, ok := wideners[oldType]; ok {
		for _, w := range widening {
			if strings.HasPrefix(newType, w) {
				return true
			}
		}
	}

	return false
}
