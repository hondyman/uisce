package api

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/hondyman/uisce/backend/internal/finops"
	"github.com/hondyman/uisce/backend/internal/optimizer"
	"github.com/jmoiron/sqlx"
)

var (
	// Regex patterns for redacting constant literals from SQL strings (Zero-Leak Audit)
	literalStringRegex = regexp.MustCompile(`'[^']*'`)
	literalNumberRegex = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
)

type ExecutionContextRequest struct {
	TenantID        uuid.UUID          `json:"tenantId"`
	RequestID       string             `json:"requestId"`
	UserID          string             `json:"userId"`
	ExecutionType   string             `json:"executionType"` // SEMANTIC_QUERY, SSRS_REPORT_RENDER, VECTOR_CALC
	CompiledSQL     string             `json:"compiledSql"`
	AST             optimizer.QueryAST `json:"ast"`
	EngineType      string             `json:"engineType"` // STARROCKS, POSTGRES_ALPHA, ICEBERG
	TermPermissions []TermPermission   `json:"termPermissions"` // Semantic term permissions for validation
}

// TermPermission represents a semantic term-level permission
type TermPermission struct {
	TermNodeID      string `json:"termNodeId"`
	TermName        string `json:"termName"`
	PermissionLevel string `json:"permissionLevel"` // 'none' | 'read' | 'write' | 'mask'
}

type ExecutionResultEnvelope struct {
	Rows           []map[string]interface{}    `json:"rows"`
	RowCount       int64                       `json:"rowCount"`
	ScannedBytes   int64                       `json:"scannedBytes"`
	CPUDurationMs  int                         `json:"cpuDurationMs"`
	TotalLatencyMs int                         `json:"totalLatencyMs"`
	Plan           *optimizer.ExplainPlanResult `json:"plan,omitempty"`
}

type QueryExecutionAuditor struct {
	db                 *sqlx.DB
	complexityScorer   *optimizer.ComplexityScorer
	auditInterceptor   *audit.AnalyticalAuditInterceptor
	budgetAlertService *finops.BudgetAlertService
	serverHost         string
}

func NewQueryExecutionAuditor(
	db *sqlx.DB,
	scorer *optimizer.ComplexityScorer,
	interceptor *audit.AnalyticalAuditInterceptor,
	alertService *finops.BudgetAlertService,
) *QueryExecutionAuditor {
	host := os.Getenv("SERVER_HOSTNAME")
	if host == "" {
		host = "100.84.50.65"
	}
	return &QueryExecutionAuditor{
		db:                 db,
		complexityScorer:   scorer,
		auditInterceptor:   interceptor,
		budgetAlertService: alertService,
		serverHost:         host,
	}
}

// ExecuteAndAudit runs the query, captures execution telemetry, and records audit metrics without row payloads
func (a *QueryExecutionAuditor) ExecuteAndAudit(
	ctx context.Context,
	req ExecutionContextRequest,
	executorFunc func(ctx context.Context, sql string) ([]map[string]interface{}, int64, int, error),
) (*ExecutionResultEnvelope, error) {
	// Rule 7 Guard: Strict Tenant Verification
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	start := time.Now()

	// 0. Pre-Flight Term Permission Validation (Semantic-Level RBAC)
	if err := validateTermPermissions(req.AST, req.TermPermissions); err != nil {
		a.recordAuditAsync(req, 0, 0, 0, int(time.Since(start).Milliseconds()), nil, "BLOCKED_TERM_PERMISSION", err.Error())
		return nil, fmt.Errorf("term permission violation: %w", err)
	}

	// 1. Pre-Flight Complexity Scoring & Rule 8 Circuit Breaker Evaluation
	plan, err := a.complexityScorer.AnalyzeQueryAST(ctx, req.TenantID, req.AST)
	if err != nil {
		return nil, fmt.Errorf("pre-flight optimization analysis failed: %w", err)
	}

	if !plan.CanExecute {
		// Circuit breaker tripped -> log blocked execution and return error
		a.recordAuditAsync(req, 0, 0, 0, int(time.Since(start).Milliseconds()), plan, "BLOCKED_CIRCUIT_BREAKER", "Query complexity exceeded threshold (Rule 8)")
		return nil, fmt.Errorf("query execution blocked by Cardinal Rule 8 cost circuit breaker (Score: %d > 80)", plan.ComplexityScore)
	}

	// 2. Physical Execution
	rows, scannedBytes, cpuMs, execErr := executorFunc(ctx, req.CompiledSQL)
	latencyMs := int(time.Since(start).Milliseconds())

	status := "COMPLETED"
	errorSummary := ""
	if execErr != nil {
		status = "FAILED"
		errorSummary = execErr.Error()
	}

	rowCount := int64(len(rows))

	// 3. Asynchronously Record Non-Leaking Audit & Evaluate FinOps Budgets
	a.recordAuditAsync(req, rowCount, scannedBytes, cpuMs, latencyMs, plan, status, errorSummary)

	if execErr != nil {
		return nil, execErr
	}

	return &ExecutionResultEnvelope{
		Rows:           rows,
		RowCount:       rowCount,
		ScannedBytes:   scannedBytes,
		CPUDurationMs:  cpuMs,
		TotalLatencyMs: latencyMs,
		Plan:           plan,
	}, nil
}

func (a *QueryExecutionAuditor) recordAuditAsync(
	req ExecutionContextRequest,
	rowCount int64,
	scannedBytes int64,
	cpuMs int,
	latencyMs int,
	plan *optimizer.ExplainPlanResult,
	status string,
	errSummary string,
) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Sanitize SQL to redact literals and preserve client confidentiality
		sanitizedSQL := redactLiterals(req.CompiledSQL)

		var planJSON map[string]interface{}
		if plan != nil {
			planJSON = map[string]interface{}{
				"planId":          plan.PlanID.String(),
				"complexityScore": plan.ComplexityScore,
				"costBand":        string(plan.CostBand),
				"nodes":           plan.Nodes,
				"edges":           plan.Edges,
			}
		}

		telemetry := audit.QueryAuditTelemetry{
			TenantID:          req.TenantID,
			RequestID:         req.RequestID,
			UserID:            req.UserID,
			ServerHost:        a.serverHost,
			ExecutionType:     req.ExecutionType,
			NormalizedQuery:   sanitizedSQL,
			ExecutionPlanJSON: planJSON,
			RowCountReturned:  rowCount,
			ScannedBytes:      scannedBytes,
			CPUDurationMs:     cpuMs,
			TotalLatencyMs:    latencyMs,
			EngineType:        req.EngineType,
			AttributedCostUSD: plan.AttributedCostUSD,
			Status:            status,
			ErrorSummary:      errSummary,
		}

		// A. Write to audit.analytical_query_execution_logs
		_ = a.auditInterceptor.RecordQueryExecution(bgCtx, telemetry)

		// B. Increment Spend in finops.tenant_compute_quotas & Write Chargeback Ledger
		if a.db != nil && plan != nil && plan.AttributedCostUSD > 0 {
			billingPeriod := time.Now().Format("2006-01")
			_, _ = a.db.ExecContext(bgCtx, `
				INSERT INTO finops.tenant_compute_quotas (tenant_id, billing_period, current_spend_usd)
				VALUES ($1, $2, $3)
				ON CONFLICT (tenant_id, billing_period)
				DO UPDATE SET current_spend_usd = finops.tenant_compute_quotas.current_spend_usd + EXCLUDED.current_spend_usd,
				              updated_at = NOW();
			`, req.TenantID, billingPeriod, plan.AttributedCostUSD)

			_, _ = a.db.ExecContext(bgCtx, `
				INSERT INTO finops.tenant_chargeback_ledger (
					tenant_id, plan_id, engine_type, scanned_bytes, cpu_milliseconds, line_item_cost_usd
				) VALUES ($1, $2, $3, $4, $5, $6);
			`, req.TenantID, plan.PlanID, req.EngineType, scannedBytes, cpuMs, plan.AttributedCostUSD)

			// C. Evaluate 80% and 95% Thresholds for Slack Webhooks
			_ = a.budgetAlertService.EvaluateTenantBudgetAndAlert(bgCtx, req.TenantID, billingPeriod)
		}
	}()
}

func redactLiterals(sql string) string {
	redacted := literalStringRegex.ReplaceAllString(sql, "'?'")
	redacted = literalNumberRegex.ReplaceAllString(redacted, "?")
	return redacted
}

// validateTermPermissions checks if the user has permission to access all fields in the AST
func validateTermPermissions(ast optimizer.QueryAST, termPermissions []TermPermission) error {
	if len(termPermissions) == 0 {
		return nil // No restrictions configured
	}

	// Build a permission map for quick lookup
	permMap := make(map[string]string) // fieldName -> permissionLevel
	for _, tp := range termPermissions {
		if tp.TermNodeID != "" {
			permMap[tp.TermNodeID] = tp.PermissionLevel
		}
		if tp.TermName != "" {
			permMap[tp.TermName] = tp.PermissionLevel
		}
	}

	// Check all selected fields
	for _, field := range ast.SelectedFields {
		if perm, ok := permMap[field]; ok {
			if perm == "none" {
				return fmt.Errorf("access denied to field '%s': permission level is 'none'", field)
			}
		}
		// If field not in permission map, it's allowed (no restriction)
	}

	return nil
}
