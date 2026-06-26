package apistudio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/region"
	"github.com/jmoiron/sqlx"
)

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
func (m *GraphQLManager) ResolveGraphQLField(ctx context.Context, ep *APIEndpoint, args map[string]interface{}) (interface{}, error) {
	var fields []string
	json.Unmarshal(ep.Fields, &fields)

	tenantUUID, _ := uuid.Parse(ep.TenantID)

	reg := ""
	if rg, ok := region.GetRegionFromContext(ctx); ok {
		reg = rg
	}

	req := analytics.BOSQLRequest{
		Env:        ep.Env,
		TenantID:   &tenantUUID,
		BOName:     ep.BOName,
		EndpointID: &ep.ID,
		Measures:   fields,
		Filters:    args,
		Region:     reg,
	}

	// 1. Generate Cache Key
	var filterKeys []string
	for k := range args {
		filterKeys = append(filterKeys, k)
	}
	planKey := GeneratePlanKey(ep.TenantID, ep.ID.String(), ep.Version, fields, filterKeys)

	// 2. Check Cache
	var sql string
	cachedSQL, err := m.planCache.GetPlan(ctx, planKey)
	if err == nil && cachedSQL != "" {
		sql = cachedSQL
	} else {
		// 3. Cache Miss - Resolve
		resolvedSQL, _, err := m.resolver.ResolveQuery(ctx, req)
		if err != nil {
			return nil, err
		}

		// 4. Cache Plan
		_ = m.planCache.SetPlan(ctx, planKey, resolvedSQL)
		sql = resolvedSQL
	}

	rows, err := m.db.QueryxContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err == nil {
			result = append(result, row)
		}
	}

	return result, nil
}
