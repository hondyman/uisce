package apistudio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/region"
	"github.com/jmoiron/sqlx"
)

// GraphQLFieldCaller carries the caller's JWT claims needed for entitlement
// evaluation (role gate, row filter binding, field masking) — mirrors what
// ServeHTTP pulls from the request's JWT claims for the REST path.
type GraphQLFieldCaller struct {
	Roles          []string
	OrganizationID string
}

// GraphQLManager handles dynamic GraphQL schema generation and resolution
type GraphQLManager struct {
	repo      *Repository
	resolver  *analytics.BOContextResolver
	db        *sqlx.DB
	planCache *GraphQLPlanCache
}

// NewGraphQLManager creates a new GraphQL manager
func NewGraphQLManager(repo *Repository, resolver *analytics.BOContextResolver, db *sqlx.DB, redisClient *redis.Client) *GraphQLManager {
	return &GraphQLManager{
		repo:      repo,
		resolver:  resolver,
		db:        db,
		planCache: NewGraphQLPlanCache(redisClient),
	}
}

// GenerateSchemaSnippet builds a GraphQL SDL snippet for the defined endpoints
func (m *GraphQLManager) GenerateSchemaSnippet(ctx context.Context, env, tenantID string) (string, error) {
	endpoints, err := m.repo.ListEndpoints(ctx, env, tenantID)
	if err != nil {
		return "", err
	}

	var schema strings.Builder
	schema.WriteString("type Query {\n")
	for _, ep := range endpoints {
		if ep.Type != "graphql" {
			continue
		}
		// Assuming ep.Name is the field name
		schema.WriteString(fmt.Sprintf("  %s(limit: Int, offset: Int): [%s]\n", ep.Name, ep.BOName))
	}
	schema.WriteString("}\n")

	return schema.String(), nil
}

// ResolveGraphQLField handles a single GraphQL field execution
func (m *GraphQLManager) ResolveGraphQLField(ctx context.Context, ep *APIEndpoint, args map[string]interface{}, caller GraphQLFieldCaller) (interface{}, error) {
	var fields []string
	json.Unmarshal(ep.Fields, &fields)

	tenantUUID, _ := uuid.Parse(ep.TenantID)

	reg := ""
	if rg, ok := region.GetRegionFromContext(ctx); ok {
		reg = rg
	}

	req := analytics.BOSQLRequest{
		Env:                  ep.Env,
		TenantID:             &tenantUUID,
		BOName:               ep.BOName,
		EndpointID:           &ep.ID,
		Measures:             fields,
		Filters:              args,
		Region:               reg,
		CallerRoles:          caller.Roles,
		CallerOrganizationID: caller.OrganizationID,
	}

	// 1. Generate Cache Key — includes caller roles: a cache hit implies
	// entitlement evaluation already ran and passed for this role set.
	planKey := GeneratePlanKey(ep.TenantID, ep.ID.String(), ep.Version, fields, args, caller.Roles)

	// 2. Check Cache
	plan, cacheErr := m.planCache.GetPlan(ctx, planKey)
	if cacheErr != nil || plan == nil {
		// 3. Cache Miss - Resolve (also evaluates the entitlement role gate)
		resolvedSQL, meta, err := m.resolver.ResolveQuery(ctx, req)
		if err != nil {
			return nil, err
		}

		// 4. Cache Plan
		plan = &CachedPlan{SQL: resolvedSQL, MaskedFields: meta.MaskedFields}
		_ = m.planCache.SetPlan(ctx, planKey, *plan)
	}

	var result []map[string]interface{}
	err := withTenantScopedQuery(ctx, m.db, ep.TenantID, plan.SQL, func(rows *sqlx.Rows) error {
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return err
			}
			result = append(result, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	applyFieldMasking(result, plan.MaskedFields, caller.Roles)

	return result, nil
}
