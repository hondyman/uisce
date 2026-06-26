package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	kafka "github.com/segmentio/kafka-go"
)

// RuleSeverity represents the severity level of a validation rule
type RuleSeverity string

const (
	SeverityWarning RuleSeverity = "WARNING"
	SeverityBlock   RuleSeverity = "BLOCK"
	SeverityInfo    RuleSeverity = "INFO"
)

// RuleScope represents the scope of a validation rule
type RuleScope string

const (
	ScopeIndividualAccount RuleScope = "INDIVIDUAL_ACCOUNT"
	ScopeJointAccount      RuleScope = "JOINT_ACCOUNT"
	ScopeTrustAccount      RuleScope = "TRUST_ACCOUNT"
	ScopeIRAAccount        RuleScope = "IRA_ACCOUNT"
	ScopeAllAccounts       RuleScope = "ALL_ACCOUNTS"
)

// RuleFrequency represents how often a rule is evaluated
type RuleFrequency string

const (
	FrequencyContinuous  RuleFrequency = "CONTINUOUS"
	FrequencyDaily       RuleFrequency = "DAILY"
	FrequencyWeekly      RuleFrequency = "WEEKLY"
	FrequencyOnTrade     RuleFrequency = "ON_TRADE"
	FrequencyOnRebalance RuleFrequency = "ON_REBALANCE"
)

// ValidationRule represents a single validation rule
type ValidationRule struct {
	ID                 string                 `db:"id" json:"id"`
	Name               string                 `db:"name" json:"name"`
	Description        string                 `db:"description" json:"description"`
	RuleType           string                 `db:"rule_type" json:"ruleType"`
	Scope              []string               `db:"scope" json:"scope"`
	Severity           RuleSeverity           `db:"severity" json:"severity"`
	IsActive           bool                   `db:"is_active" json:"isActive"`
	EffectiveFrom      time.Time              `db:"effective_from" json:"effectiveFrom"`
	EffectiveTo        *time.Time             `db:"effective_to" json:"effectiveTo,omitempty"`
	Frequency          RuleFrequency          `db:"frequency" json:"frequency"`
	EvaluationOrder    int                    `db:"evaluation_order" json:"evaluationOrder"`
	OverrideConditions []string               `db:"override_conditions" json:"overrideConditions,omitempty"`
	RequiredAuthority  *string                `db:"required_authority" json:"requiredAuthority,omitempty"`
	Parameters         map[string]interface{} `db:"parameters" json:"parameters"`
	CreatedAt          time.Time              `db:"created_at" json:"createdAt"`
	UpdatedAt          time.Time              `db:"updated_at" json:"updatedAt"`
	TenantID           string                 `db:"tenant_id" json:"tenantId"`
	DatasourceID       string                 `db:"datasource_id" json:"datasourceId"`
}

// ValidationContext represents the context for validation
type ValidationContext struct {
	AccountID             string                 `json:"accountId"`
	AccountType           string                 `json:"accountType"`
	ClientID              string                 `json:"clientId"`
	PortfolioData         map[string]interface{} `json:"portfolioData,omitempty"`
	TransactionData       map[string]interface{} `json:"transactionData,omitempty"`
	ClientProfile         map[string]interface{} `json:"clientProfile,omitempty"`
	Timestamp             time.Time              `json:"timestamp"`
	UserID                *string                `json:"userId,omitempty"`
	OverrideAuthorization *string                `json:"overrideAuthorization,omitempty"`
	TenantID              string                 `json:"tenantId"`
	DatasourceID          string                 `json:"datasourceId"`
}

// ValidationResult represents the result of a single validation
type ValidationResult struct {
	RuleID                   string                 `json:"ruleId"`
	RuleName                 string                 `json:"ruleName"`
	Passed                   bool                   `json:"passed"`
	Severity                 RuleSeverity           `json:"severity"`
	Message                  string                 `json:"message"`
	Details                  map[string]interface{} `json:"details,omitempty"`
	Timestamp                time.Time              `json:"timestamp"`
	RequiresOverride         bool                   `json:"requiresOverride,omitempty"`
	AllowedOverrideAuthority *string                `json:"allowedOverrideAuthority,omitempty"`
	FailedValue              interface{}            `json:"failedValue,omitempty"`
	Threshold                interface{}            `json:"threshold,omitempty"`
}

// ValidationExecutionResult represents the overall validation result
type ValidationExecutionResult struct {
	ContextID       string             `json:"contextId"`
	AccountID       string             `json:"accountId"`
	Passed          bool               `json:"passed"`
	Timestamp       time.Time          `json:"timestamp"`
	Results         []ValidationResult `json:"results"`
	BlockedRules    []ValidationResult `json:"blockedRules"`
	WarningRules    []ValidationResult `json:"warningRules"`
	InfoRules       []ValidationResult `json:"infoRules"`
	ExecutionTimeMs int64              `json:"executionTimeMs"`
	TenantID        string             `json:"tenantId"`
	DatasourceID    string             `json:"datasourceId"`
}

// WealthManagementValidationEngine orchestrates validation rules
type WealthManagementValidationEngine struct {
	db        *sqlx.DB
	writer    *kafka.Writer
	topicName string
	rules     map[string]*ValidationRule
}

// NewWealthManagementValidationEngine creates a new validation engine
func NewWealthManagementValidationEngine(db *sqlx.DB, kafkaBrokers string) (*WealthManagementValidationEngine, error) {
	engine := &WealthManagementValidationEngine{
		db:        db,
		topicName: "wealth-management-events",
		rules:     make(map[string]*ValidationRule),
	}

	// If Kafka brokers are provided, create a writer to publish events
	if kafkaBrokers != "" {
		brokers := strings.Split(kafkaBrokers, ",")
		w := &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
		}
		engine.writer = w
	}

	return engine, nil
}

// ExecuteValidations runs all applicable validation rules
func (e *WealthManagementValidationEngine) ExecuteValidations(ctx context.Context, validationCtx *ValidationContext) (*ValidationExecutionResult, error) {
	startTime := time.Now()

	// Get applicable rules for account type
	applicableRules, err := e.getApplicableRules(ctx, validationCtx.AccountType, validationCtx.TenantID, validationCtx.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable rules: %w", err)
	}

	// Sort by evaluation order
	sort.Slice(applicableRules, func(i, j int) bool {
		return applicableRules[i].EvaluationOrder < applicableRules[j].EvaluationOrder
	})

	// Execute each rule
	results := []ValidationResult{}
	for _, rule := range applicableRules {
		result, err := e.executeRule(ctx, rule, validationCtx)
		if err != nil {
			log.Printf("Error executing rule %s: %v\n", rule.ID, err)
			results = append(results, ValidationResult{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Passed:    false,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("Rule execution error: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			results = append(results, result)
		}
	}

	// Categorize results
	blockedRules := []ValidationResult{}
	warningRules := []ValidationResult{}
	infoRules := []ValidationResult{}

	for _, result := range results {
		if !result.Passed {
			switch result.Severity {
			case SeverityBlock:
				blockedRules = append(blockedRules, result)
			case SeverityWarning:
				warningRules = append(warningRules, result)
			case SeverityInfo:
				infoRules = append(infoRules, result)
			}
		}
	}

	executionResult := &ValidationExecutionResult{
		ContextID:       fmt.Sprintf("%s-%d", validationCtx.AccountID, time.Now().UnixMilli()),
		AccountID:       validationCtx.AccountID,
		Passed:          len(blockedRules) == 0,
		Timestamp:       time.Now(),
		Results:         results,
		BlockedRules:    blockedRules,
		WarningRules:    warningRules,
		InfoRules:       infoRules,
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		TenantID:        validationCtx.TenantID,
		DatasourceID:    validationCtx.DatasourceID,
	}

	// Persist results
	err = e.persistValidationResults(ctx, executionResult)
	if err != nil {
		log.Printf("Warning: Failed to persist validation results: %v\n", err)
	}

	// Publish events if there are blocked rules
	if len(blockedRules) > 0 && e.writer != nil {
		err = e.publishValidationFailureEvent(executionResult)
		if err != nil {
			log.Printf("Warning: Failed to publish validation event: %v", err)
		}
	}

	return executionResult, nil
}

// getApplicableRules retrieves rules applicable to the account type
func (e *WealthManagementValidationEngine) getApplicableRules(ctx context.Context, accountType string, tenantID string, datasourceID string) ([]*ValidationRule, error) {
	query := `
		SELECT id, name, description, rule_type, scope, severity, is_active,
		       effective_from, effective_to, frequency, evaluation_order,
		       override_conditions, required_authority, parameters, created_at,
		       updated_at, tenant_id, datasource_id
		FROM validation_rules
		WHERE tenant_id = $1 AND datasource_id = $2
		  AND is_active = true
		  AND effective_from <= NOW()
		  AND (effective_to IS NULL OR effective_to >= NOW())
		ORDER BY evaluation_order ASC
	`

	var rules []*ValidationRule
	err := e.db.SelectContext(ctx, &rules, query, tenantID, datasourceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return []*ValidationRule{}, nil
	}

	// Filter by scope
	var filtered []*ValidationRule
	for _, rule := range rules {
		if contains(rule.Scope, "ALL_ACCOUNTS") || contains(rule.Scope, accountType) {
			filtered = append(filtered, rule)
		}
	}

	return filtered, nil
}

// executeRule executes an individual rule
func (e *WealthManagementValidationEngine) executeRule(ctx context.Context, rule *ValidationRule, validationCtx *ValidationContext) (ValidationResult, error) {
	switch rule.RuleType {
	case "CONCENTRATION":
		return e.validateConcentration(rule, validationCtx), nil
	case "KYC":
		return e.validateKYC(rule, validationCtx), nil
	case "ASSET_RESTRICTION":
		return e.validateAssetRestriction(rule, validationCtx), nil
	case "LIQUIDITY":
		return e.validateLiquidity(rule, validationCtx), nil
	case "DATA_INTEGRITY":
		return e.validateDataIntegrity(rule, validationCtx), nil
	case "TRADE":
		return e.validateTrade(rule, validationCtx), nil
	case "FEE":
		return e.validateFee(rule, validationCtx), nil
	case "ACCESS_CONTROL":
		return e.validateAccessControl(ctx, rule, validationCtx), nil
	default:
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    true,
			Severity:  SeverityInfo,
			Message:   "Unknown rule type",
			Timestamp: time.Now(),
		}, nil
	}
}

// validateConcentration checks concentration limits
func (e *WealthManagementValidationEngine) validateConcentration(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	if ctx.PortfolioData == nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    true,
			Severity:  rule.Severity,
			Message:   "No portfolio data provided",
			Timestamp: time.Now(),
		}
	}

	blockThreshold := getFloatParam(rule.Parameters, "blockThreshold", 0.35)
	minSize := getFloatParam(rule.Parameters, "minimumPositionSize", 100000)

	positions := getArrayParam(ctx.PortfolioData, "positions", []interface{}{})
	portfolioValue := getFloatParam(ctx.PortfolioData, "totalValue", 1)

	violations := []map[string]interface{}{}

	for _, pos := range positions {
		posMap := pos.(map[string]interface{})
		posValue := getFloatParam(posMap, "marketValue", 0)

		if posValue < minSize {
			continue
		}

		posPercentage := posValue / portfolioValue
		if posPercentage > blockThreshold {
			violations = append(violations, map[string]interface{}{
				"security":   posMap["ticker"],
				"percentage": fmt.Sprintf("%.2f", posPercentage*100),
				"threshold":  fmt.Sprintf("%.2f", blockThreshold*100),
			})
		}
	}

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    len(violations) == 0,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Positions within limits: %d violations", len(violations)),
		Details:   map[string]interface{}{"violations": violations},
		Timestamp: time.Now(),
	}
}

// validateKYC checks KYC completeness
func (e *WealthManagementValidationEngine) validateKYC(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	requiredFields := getArrayStringParam(rule.Parameters, "requiredFields", []string{})
	profile := ctx.ClientProfile

	if profile == nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    false,
			Severity:  rule.Severity,
			Message:   "No client profile provided",
			Timestamp: time.Now(),
		}
	}

	missingFields := []string{}
	for _, field := range requiredFields {
		if _, ok := profile[field]; !ok {
			missingFields = append(missingFields, field)
		}
	}

	passed := len(missingFields) == 0
	message := "KYC requirements met"

	if !passed {
		message = fmt.Sprintf("Missing KYC fields: %v", missingFields)
	}

	return ValidationResult{
		RuleID:           rule.ID,
		RuleName:         rule.Name,
		Passed:           passed,
		Severity:         rule.Severity,
		Message:          message,
		Details:          map[string]interface{}{"missingFields": missingFields},
		Timestamp:        time.Now(),
		RequiresOverride: !passed,
	}
}

// validateAssetRestriction checks asset restrictions
func (e *WealthManagementValidationEngine) validateAssetRestriction(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	positions := getArrayParam(ctx.PortfolioData, "positions", []interface{}{})

	// Get restrictions for account type
	restrictions := rule.Parameters[ctx.AccountType].(map[string]interface{})
	prohibited := getArrayStringParam(restrictions, "prohibitedAssets", []string{})

	violations := []string{}
	for _, pos := range positions {
		posMap := pos.(map[string]interface{})
		assetType := getStringParam(posMap, "assetType", "")
		if validateRuleContainsString(prohibited, assetType) {
			violations = append(violations, assetType)
		}
	}

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    len(violations) == 0,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Prohibited assets: %d found", len(violations)),
		Details:   map[string]interface{}{"violations": violations},
		Timestamp: time.Now(),
	}
}

// validateLiquidity checks liquidity constraints
func (e *WealthManagementValidationEngine) validateLiquidity(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	maxIlliquid := getFloatParam(rule.Parameters, "maxIlliquidPercentage", 0.2)
	illiquidTypes := getArrayStringParam(rule.Parameters, "illiquidAssetTypes", []string{})

	positions := getArrayParam(ctx.PortfolioData, "positions", []interface{}{})
	portfolioValue := getFloatParam(ctx.PortfolioData, "totalValue", 1)

	illiquidValue := 0.0
	for _, pos := range positions {
		posMap := pos.(map[string]interface{})
		assetType := getStringParam(posMap, "assetType", "")
		if validateRuleContainsString(illiquidTypes, assetType) {
			illiquidValue += getFloatParam(posMap, "marketValue", 0)
		}
	}

	illiquidPct := illiquidValue / portfolioValue

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    illiquidPct <= maxIlliquid,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Illiquid assets: %.2f%% (max: %.2f%%)", illiquidPct*100, maxIlliquid*100),
		Details:   map[string]interface{}{"illiquidPercentage": illiquidPct},
		Timestamp: time.Now(),
	}
}

// validateDataIntegrity checks data integrity
func (e *WealthManagementValidationEngine) validateDataIntegrity(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    true,
		Severity:  rule.Severity,
		Message:   "Data integrity check passed",
		Timestamp: time.Now(),
	}
}

// validateTrade checks trade execution feasibility
func (e *WealthManagementValidationEngine) validateTrade(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	if ctx.TransactionData == nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    true,
			Severity:  rule.Severity,
			Message:   "No transaction data",
			Timestamp: time.Now(),
		}
	}

	cashBuffer := getFloatParam(rule.Parameters, "cashBuffer", 0.01)
	availableCash := getFloatParam(ctx.PortfolioData, "cash", 0)
	amount := getFloatParam(ctx.TransactionData, "amount", 0)
	requiredAmount := amount * (1 + cashBuffer)

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    availableCash >= requiredAmount,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Available: $%.2f, Required: $%.2f", availableCash, requiredAmount),
		Timestamp: time.Now(),
	}
}

// validateFee checks fee compliance
func (e *WealthManagementValidationEngine) validateFee(rule *ValidationRule, ctx *ValidationContext) ValidationResult {
	if ctx.TransactionData == nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    true,
			Severity:  rule.Severity,
			Message:   "No transaction data",
			Timestamp: time.Now(),
		}
	}

	maxFee := getFloatParam(rule.Parameters, "maxAdvisoryFeePercentage", 0.02)
	feePercentage := getFloatParam(ctx.TransactionData, "feePercentage", 0)

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    feePercentage <= maxFee,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Fee: %.3f%% (max: %.3f%%)", feePercentage*100, maxFee*100),
		Timestamp: time.Now(),
	}
}

// validateAccessControl checks access permissions
func (e *WealthManagementValidationEngine) validateAccessControl(ctx context.Context, rule *ValidationRule, validationCtx *ValidationContext) ValidationResult {
	if validationCtx.UserID == nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    false,
			Severity:  rule.Severity,
			Message:   "User ID required",
			Timestamp: time.Now(),
		}
	}

	// Query database for advisor assignment
	query := `
		SELECT COUNT(*) FROM advisor_assignments
		WHERE user_id = $1 AND account_id = $2
	`

	var count int
	err := e.db.GetContext(ctx, &count, query, *validationCtx.UserID, validationCtx.AccountID)
	if err != nil {
		return ValidationResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Passed:    false,
			Severity:  rule.Severity,
			Message:   "Failed to verify access permissions",
			Timestamp: time.Now(),
		}
	}

	hasAccess := count > 0

	return ValidationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Passed:    hasAccess,
		Severity:  rule.Severity,
		Message:   fmt.Sprintf("Access: %v", hasAccess),
		Timestamp: time.Now(),
	}
}

// persistValidationResults saves results to database
func (e *WealthManagementValidationEngine) persistValidationResults(ctx context.Context, result *ValidationExecutionResult) error {
	for _, r := range result.Results {
		detailsJSON, _ := json.Marshal(r.Details)
		query := `
			INSERT INTO validation_results 
			(account_id, rule_id, rule_name, passed, severity, message, details, executed_at, tenant_id, datasource_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
		_, err := e.db.ExecContext(ctx, query,
			result.AccountID,
			r.RuleID,
			r.RuleName,
			r.Passed,
			r.Severity,
			r.Message,
			string(detailsJSON),
			result.Timestamp,
			result.TenantID,
			result.DatasourceID,
		)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	return nil
}

// publishValidationFailureEvent publishes a validation failure to Kafka
func (e *WealthManagementValidationEngine) publishValidationFailureEvent(result *ValidationExecutionResult) error {
	if e.writer == nil {
		return nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	routingKey := fmt.Sprintf("validation.failure.%s", result.AccountID)
	msg := kafka.Message{
		Topic: e.topicName,
		Key:   []byte(routingKey),
		Value: data,
		Time:  time.Now(),
	}

	return e.writer.WriteMessages(context.Background(), msg)
}

// UpsertRule adds or updates a validation rule
func (e *WealthManagementValidationEngine) UpsertRule(ctx context.Context, rule *ValidationRule) error {
	scopeJSON, _ := json.Marshal(rule.Scope)
	paramsJSON, _ := json.Marshal(rule.Parameters)
	overrideJSON, _ := json.Marshal(rule.OverrideConditions)

	query := `
		INSERT INTO validation_rules 
		(id, name, description, rule_type, scope, severity, is_active, effective_from, 
		 effective_to, frequency, evaluation_order, override_conditions, required_authority, 
		 parameters, created_at, updated_at, tenant_id, datasource_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id, tenant_id, datasource_id) DO UPDATE SET
		  name = $2, description = $3, rule_type = $4, scope = $5, severity = $6,
		  is_active = $7, effective_from = $8, effective_to = $9, frequency = $10,
		  evaluation_order = $11, override_conditions = $12, required_authority = $13,
		  parameters = $14, updated_at = $16
	`

	_, err := e.db.ExecContext(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.RuleType, string(scopeJSON),
		rule.Severity, rule.IsActive, rule.EffectiveFrom, rule.EffectiveTo,
		rule.Frequency, rule.EvaluationOrder, string(overrideJSON),
		rule.RequiredAuthority, string(paramsJSON), rule.CreatedAt, rule.UpdatedAt,
		rule.TenantID, rule.DatasourceID,
	)

	return err
}

// GetRules retrieves rules with optional filters
func (e *WealthManagementValidationEngine) GetRules(ctx context.Context, tenantID string, datasourceID string, filters map[string]interface{}) ([]*ValidationRule, error) {
	query := `
		SELECT id, name, description, rule_type, scope, severity, is_active,
		       effective_from, effective_to, frequency, evaluation_order,
		       override_conditions, required_authority, parameters, created_at,
		       updated_at, tenant_id, datasource_id
		FROM validation_rules
		WHERE tenant_id = $1 AND datasource_id = $2
	`

	args := []interface{}{tenantID, datasourceID}
	argIndex := 3

	if ruleType, ok := filters["ruleType"].(string); ok {
		query += fmt.Sprintf(" AND rule_type = $%d", argIndex)
		args = append(args, ruleType)
		argIndex++
	}

	if scope, ok := filters["scope"].(string); ok {
		query += fmt.Sprintf(" AND scope @> $%d::text[]", argIndex)
		args = append(args, "{"+scope+"}")
		argIndex++
	}

	query += " ORDER BY evaluation_order ASC"

	var rules []*ValidationRule
	err := e.db.SelectContext(ctx, &rules, query, args...)

	return rules, err
}

// Helper functions
func validateRuleContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getFloatParam(m map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return defaultVal
}

func getStringParam(m map[string]interface{}, key string, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getArrayParam(m map[string]interface{}, key string, defaultVal []interface{}) []interface{} {
	if v, ok := m[key]; ok {
		if a, ok := v.([]interface{}); ok {
			return a
		}
	}
	return defaultVal
}

func getArrayStringParam(m map[string]interface{}, key string, defaultVal []string) []string {
	if v, ok := m[key]; ok {
		if a, ok := v.([]interface{}); ok {
			var result []string
			for _, item := range a {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return defaultVal
}

// Close closes resources
func (e *WealthManagementValidationEngine) Close() error {
	if e.writer != nil {
		return e.writer.Close()
	}
	return nil
}
