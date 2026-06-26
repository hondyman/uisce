package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// GovernanceContextProvider provides governance context for NL queries
type GovernanceContextProvider struct {
	evaluator     domain.Evaluator
	policyChecker domain.PolicyChecker
	schemaRepo    domain.SchemaProvider
}

// GovernanceContext contains governance information for query generation
type GovernanceContext struct {
	UserID            string
	TenantID          string
	Datasource        string
	AllowedMetrics    []string
	AllowedDimensions []string
	RequiredFilters   []QueryFilter
	AppliedPolicies   []AppliedGovernancePolicy
	AssetMappings     map[string]string // Maps semantic names to asset IDs
}

// AppliedGovernancePolicy represents a policy that was applied
type AppliedGovernancePolicy struct {
	ID     string
	RuleID string
	Action string
	Reason string
}

// QuerySkeleton represents the intermediate query structure before SQL generation
type QuerySkeleton struct {
	Measures    []string
	Dimensions  []string
	Filters     []QueryFilter
	OrderBy     []OrderBySpec
	SemanticSQL string
}

// NewGovernanceContextProvider creates a new governance context provider
func NewGovernanceContextProvider(evaluator domain.Evaluator, policyChecker domain.PolicyChecker, schemaRepo domain.SchemaProvider) *GovernanceContextProvider {
	return &GovernanceContextProvider{
		evaluator:     evaluator,
		policyChecker: policyChecker,
		schemaRepo:    schemaRepo,
	}
}

// GetContext retrieves governance context for a user/tenant/datasource combination
func (gcp *GovernanceContextProvider) GetContext(ctx context.Context, userID, tenantID, datasource string) (*GovernanceContext, error) {
	govCtx := &GovernanceContext{
		UserID:            userID,
		TenantID:          tenantID,
		Datasource:        datasource,
		AllowedMetrics:    []string{},
		AllowedDimensions: []string{},
		RequiredFilters:   []QueryFilter{},
		AppliedPolicies:   []AppliedGovernancePolicy{},
		AssetMappings:     make(map[string]string),
	}

	// Get effective claims for the user
	req := domain.EvaluationRequest{
		UserID:   userID,
		TenantID: tenantID,
		AssetID:  datasource,
		Action:   domain.PermRead,
	}

	allow, reason, claims, err := gcp.evaluator.Evaluate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate access: %w", err)
	}

	if !allow {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	// Extract allowed assets from claims
	for _, claim := range claims {
		// Parse claim scopes to determine allowed metrics/dimensions
		for _, scope := range claim.Scope {
			if strings.HasPrefix(scope, "metric:") {
				metric := strings.TrimPrefix(scope, "metric:")
				govCtx.AllowedMetrics = append(govCtx.AllowedMetrics, metric)
			} else if strings.HasPrefix(scope, "dimension:") {
				dimension := strings.TrimPrefix(scope, "dimension:")
				govCtx.AllowedDimensions = append(govCtx.AllowedDimensions, dimension)
			}
		}
	}

	// Check policies for additional constraints
	allow, reason, matched, scopes, err := gcp.policyChecker.Check(ctx, req, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to check policies: %w", err)
	}

	if !allow {
		return nil, fmt.Errorf("policy violation: %s", reason)
	}

	// Add required filters based on policies
	for _, scope := range scopes {
		if scope == "tenant_isolation" {
			govCtx.RequiredFilters = append(govCtx.RequiredFilters, QueryFilter{
				Field:    "tenant_id",
				Operator: "=",
				Value:    tenantID,
			})
		}
	}

	// Record applied policies
	for _, match := range matched {
		if policyID, ok := match["policyId"].(string); ok {
			govCtx.AppliedPolicies = append(govCtx.AppliedPolicies, AppliedGovernancePolicy{
				ID:     policyID,
				RuleID: "rule_1", // Extract from match if available
				Action: "filter",
				Reason: "Access control policy applied",
			})
		}
	}

	// Build asset mappings from schema
	if err := gcp.buildAssetMappings(govCtx, datasource); err != nil {
		// Log error but don't fail - mappings are optional
		fmt.Printf("Failed to build asset mappings: %v\n", err)
	}

	return govCtx, nil
}

// buildAssetMappings creates mappings from semantic names to asset IDs
func (gcp *GovernanceContextProvider) buildAssetMappings(govCtx *GovernanceContext, datasource string) error {
	// Get schema information
	schema, err := gcp.schemaRepo.GetAssetSchema(datasource)
	if err != nil {
		return err
	}

	// Build mappings for allowed assets from all scopes
	for _, columns := range schema.ColumnsByScope {
		for _, column := range columns {
			// Create semantic name mappings
			semanticName := strings.ToLower(strings.ReplaceAll(column, "_", " "))
			govCtx.AssetMappings[semanticName] = column

			// Also map the column name directly
			govCtx.AssetMappings[column] = column
		}
	}

	return nil
}
