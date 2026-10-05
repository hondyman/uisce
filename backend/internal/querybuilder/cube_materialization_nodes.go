package querybuilder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// CubeMaterializationNode is one pre_aggregation catalog_node owned by a cube grain.
type CubeMaterializationNode struct {
	ID         uuid.UUID
	NodeName   string
	Grain      []string
	GrainHash  string
	Properties models.PreAggProperties
}

// CubeMaterializationRegistry ensures and drives lifecycle for cube grain nodes.
//
// Nodes reuse catalog_type_name=pre_aggregation so CubeRouter's existing lookup
// keeps working (CUBE-1.1).
type CubeMaterializationRegistry struct {
	db        *sqlx.DB
	lifecycle *analytics.PreAggLifecycleService
}

func NewCubeMaterializationRegistry(db *sqlx.DB, lifecycle *analytics.PreAggLifecycleService) *CubeMaterializationRegistry {
	if lifecycle == nil && db != nil {
		lifecycle = analytics.NewPreAggLifecycleService(db)
	}
	return &CubeMaterializationRegistry{db: db, lifecycle: lifecycle}
}

// GrainHash is a stable short fingerprint of a grain's members (sorted) so
// Temporal workflow IDs and reconcile keys stay stable across authoring order.
func GrainHash(grain []string) string {
	sorted := append([]string(nil), grain...)
	for i := range sorted {
		sorted[i] = strings.ToLower(strings.TrimSpace(sorted[i]))
	}
	sort.Strings(sorted)
	h := sha256.Sum256([]byte(strings.Join(sorted, "\x1f")))
	return hex.EncodeToString(h[:8])
}

// EnsureGrainNodes upserts one pre_aggregation catalog node per cube grain.
// Re-ensure preserves lifecycle / attempt_id / LastRefreshedAt.
func (r *CubeMaterializationRegistry) EnsureGrainNodes(ctx context.Context, cube CubeDefinition) ([]CubeMaterializationNode, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("cube materialization registry: database not configured")
	}
	if strings.TrimSpace(cube.ID) == "" {
		return nil, fmt.Errorf("cube materialization registry: cube id is required")
	}
	if len(cube.Grains) == 0 {
		return nil, ErrCubeNoGrains
	}

	var nodeTypeID string
	if err := r.db.GetContext(ctx, &nodeTypeID, `
		SELECT id::text FROM catalog_node_type WHERE catalog_type_name = 'pre_aggregation' LIMIT 1
	`); err != nil {
		return nil, fmt.Errorf("pre_aggregation node type not found: %w", err)
	}

	out := make([]CubeMaterializationNode, 0, len(cube.Grains))
	for _, grain := range cube.Grains {
		g := normalizeGrain(grain)
		if len(g) == 0 {
			continue
		}
		node, err := r.ensureOneGrain(ctx, cube, nodeTypeID, g)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	if len(out) == 0 {
		return nil, ErrCubeNoGrains
	}
	return out, nil
}

func (r *CubeMaterializationRegistry) ensureOneGrain(
	ctx context.Context,
	cube CubeDefinition,
	nodeTypeID string,
	grain []string,
) (CubeMaterializationNode, error) {
	nodeName := CubeMaterializationName(cube.TenantID, cube.IsCore, cube, grain)
	gHash := GrainHash(grain)
	qualifiedPath := fmt.Sprintf("pre_aggregation/%s/%s", cube.TenantID, nodeName)

	refreshStrategy := strings.TrimSpace(cube.Materialization.RefreshStrategy)
	if refreshStrategy == "" {
		refreshStrategy = "manual"
	}
	targetDB := fmt.Sprintf("tenant_%s", sanitizeIdentifier(cube.TenantID))
	if cube.IsCore {
		targetDB = "gold"
	}

	cfg := models.PreAggConfig{
		Terms:        grain,
		GroupBy:      grain,
		Calculations: append([]string(nil), cube.MetricIDs...),
		Materialization: models.MaterializationConfig{
			Type:       "table",
			TargetName: nodeName,
		},
	}
	if strings.EqualFold(strings.TrimSpace(cube.Materialization.Strategy), "starrocks_mv") {
		cfg.Materialization.Type = "materialized_view"
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return CubeMaterializationNode{}, err
	}

	var existing struct {
		ID         uuid.UUID       `db:"id"`
		Properties json.RawMessage `db:"properties"`
	}
	err = r.db.GetContext(ctx, &existing, `
		SELECT id, properties FROM catalog_node
		WHERE tenant_id = $1::uuid AND qualified_path = $2
	`, cube.TenantID, qualifiedPath)

	switch {
	case err == nil:
		props, perr := models.ParsePreAggProperties(existing.Properties)
		if perr != nil {
			props = &models.PreAggProperties{}
		}
		// Preserve lifecycle / attempt / freshness; refresh cube linkage fields.
		props.BOName = cube.BOID
		props.TenantID = cube.TenantID
		props.Dialect = "starrocks"
		props.RefreshStrategy = refreshStrategy
		props.RefreshIntervalMinutes = cube.Materialization.RefreshIntervalMinutes
		props.TargetDatabase = targetDB
		props.CubeID = cube.ID
		props.ContractVersion = cube.ContractVersion
		props.Grain = grain
		props.GrainHash = gHash
		props.CubeContentHash = cube.ContentHash
		if props.LifecycleStatus == "" {
			props.LifecycleStatus = models.LifecycleIdle
		}
		propsJSON, mErr := json.Marshal(props)
		if mErr != nil {
			return CubeMaterializationNode{}, mErr
		}
		if _, uErr := r.db.ExecContext(ctx, `
			UPDATE catalog_node
			SET node_name = $1, description = $2, properties = $3::jsonb, config = $4::jsonb, updated_at = NOW()
			WHERE id = $5
		`, nodeName,
			fmt.Sprintf("Cube %s grain %s (v%d)", cube.Name, gHash, cube.ContractVersion),
			propsJSON, cfgJSON, existing.ID,
		); uErr != nil {
			return CubeMaterializationNode{}, fmt.Errorf("update cube materialization node %s: %w", nodeName, uErr)
		}
		return CubeMaterializationNode{
			ID:         existing.ID,
			NodeName:   nodeName,
			Grain:      grain,
			GrainHash:  gHash,
			Properties: *props,
		}, nil

	case err == sql.ErrNoRows:
		props := models.PreAggProperties{
			BOName:                 cube.BOID,
			TenantID:               cube.TenantID,
			Dialect:                "starrocks",
			RefreshStrategy:        refreshStrategy,
			RefreshIntervalMinutes: cube.Materialization.RefreshIntervalMinutes,
			GovernanceStatus:       "published",
			TargetDatabase:         targetDB,
			LifecycleStatus:        models.LifecycleIdle,
			CubeID:                 cube.ID,
			ContractVersion:        cube.ContractVersion,
			Grain:                  grain,
			GrainHash:              gHash,
			CubeContentHash:        cube.ContentHash,
		}
		propsJSON, mErr := json.Marshal(props)
		if mErr != nil {
			return CubeMaterializationNode{}, mErr
		}
		nodeID := uuid.New()
		if _, iErr := r.db.ExecContext(ctx, `
			INSERT INTO catalog_node (
				id, node_name, description, node_type_id, tenant_id, qualified_path,
				properties, config, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4::uuid, $5::uuid, $6, $7::jsonb, $8::jsonb, NOW(), NOW()
			)
		`, nodeID, nodeName,
			fmt.Sprintf("Cube %s grain %s (v%d)", cube.Name, gHash, cube.ContractVersion),
			nodeTypeID, cube.TenantID, qualifiedPath, propsJSON, cfgJSON,
		); iErr != nil {
			return CubeMaterializationNode{}, fmt.Errorf("insert cube materialization node %s: %w", nodeName, iErr)
		}
		return CubeMaterializationNode{
			ID:         nodeID,
			NodeName:   nodeName,
			Grain:      grain,
			GrainHash:  gHash,
			Properties: props,
		}, nil

	default:
		return CubeMaterializationNode{}, fmt.Errorf("lookup cube materialization node %s: %w", nodeName, err)
	}
}

// BeginAttempt marks a grain node materializing and stamps attempt_id.
func (r *CubeMaterializationRegistry) BeginAttempt(ctx context.Context, nodeID uuid.UUID, attemptID string) error {
	if r == nil || r.lifecycle == nil {
		return fmt.Errorf("cube materialization registry: lifecycle not configured")
	}
	if strings.TrimSpace(attemptID) == "" {
		return fmt.Errorf("cube materialization registry: attempt_id is required")
	}
	return r.lifecycle.MarkMaterializingAttempt(ctx, nodeID, attemptID)
}

// CompleteAttempt marks Active and sets LastRefreshedAt (freshness clock).
func (r *CubeMaterializationRegistry) CompleteAttempt(ctx context.Context, nodeID uuid.UUID, attemptID string, stats *models.PreAggStats) error {
	if r == nil || r.lifecycle == nil {
		return fmt.Errorf("cube materialization registry: lifecycle not configured")
	}
	if strings.TrimSpace(attemptID) == "" {
		return fmt.Errorf("cube materialization registry: attempt_id is required")
	}
	return r.lifecycle.MarkActiveAttempt(ctx, nodeID, attemptID, stats)
}

// FailAttempt marks Failed for the attempt without advancing LastRefreshedAt.
func (r *CubeMaterializationRegistry) FailAttempt(ctx context.Context, nodeID uuid.UUID, attemptID string, cause error) error {
	if r == nil || r.lifecycle == nil {
		return fmt.Errorf("cube materialization registry: lifecycle not configured")
	}
	if strings.TrimSpace(attemptID) == "" {
		return fmt.Errorf("cube materialization registry: attempt_id is required")
	}
	return r.lifecycle.MarkFailedAttempt(ctx, nodeID, attemptID, cause)
}

func normalizeGrain(grain []string) []string {
	out := make([]string, 0, len(grain))
	seen := map[string]struct{}{}
	for _, g := range grain {
		v := strings.ToLower(strings.TrimSpace(g))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
