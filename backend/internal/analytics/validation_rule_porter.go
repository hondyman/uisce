package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// ValidationRulePorter implements rule export/import between environments.
// Export produces environment-agnostic RuleBundles: no catalog UUIDs, no
// physical column bindings — only semantic terms and stable rule keys.
type ValidationRulePorter struct {
	db *sqlx.DB
}

func NewValidationRulePorter(db *sqlx.DB) *ValidationRulePorter {
	return &ValidationRulePorter{db: db}
}

// ExportRules exports validation rules as a portable bundle.
//
//   - originFilter == "core": exports only gold-copy (core) rules. Requires the
//     gold-copy tenant to be configured.
//   - originFilter == "custom": exports only the requesting tenant's custom rules.
//   - originFilter == "": exports the effective ruleset (core + tenant custom,
//     with core winning on rule_key collision — mirroring ListByBO semantics).
//
// Only active rules are exported. The bundle checksum is computed over
// canonicalized ASTs (vm.Compact), so re-exporting the same ruleset from two
// environments yields byte-identical checksums.
func (p *ValidationRulePorter) ExportRules(ctx context.Context, tenantID, boName, domain, originFilter string) (*models.RuleBundle, error) {
	gold, err := goldCopyTenantID(ctx, p.db)
	if err != nil {
		return nil, err
	}

	queryTenants := visibleTenants(tenantID, gold)
	if originFilter == models.ValidationRuleOriginCore {
		if gold == "" {
			return nil, fmt.Errorf("export origin=core: gold-copy tenant is not configured")
		}
		queryTenants = []string{gold}
	}

	var nodes []struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
		IsActive    bool            `db:"is_active"`
		TenantID    string          `db:"tenant_id"`
	}

	// Domain filter mirrors ListByBO: empty domain excludes survivorship rules
	// by default; an explicit domain filters to exactly that domain.
	err = p.db.SelectContext(ctx, &nodes, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description,
		       n.properties, n.config, n.is_active, n.tenant_id::text AS tenant_id
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = ANY($1::uuid[])
		  AND n.is_active
		  AND ($2 = '' OR n.properties->>'bo_name' = $2)
		  AND (($3 = '' AND COALESCE(NULLIF(n.properties->>'domain', ''), $4) <> $5)
		       OR COALESCE(NULLIF(n.properties->>'domain', ''), $4) = $3)
		ORDER BY n.node_name
	`, pq.Array(queryTenants), boName, domain, models.ValidationRuleDomainDefault, models.ValidationRuleDomainSurvivorship)
	if err != nil {
		return nil, fmt.Errorf("query validation rule nodes: %w", err)
	}

	bundle := &models.RuleBundle{
		BundleVersion: models.RuleBundleVersion,
		TenantScope:   tenantScopeFor(originFilter, tenantID, gold),
		Origin:        originFilter,
		Rules:         make([]models.PortableRuleSpec, 0, len(nodes)),
	}

	// rule_key collision resolution: core wins over custom (mirrors ListByBO).
	byKey := make(map[string]int, len(nodes))

	for _, n := range nodes {
		origin := originOf(n.TenantID, gold)
		if originFilter == models.ValidationRuleOriginCustom && origin != models.ValidationRuleOriginCustom {
			continue
		}

		spec, ruleKey, err := portableSpecFromNode(n.ID, n.NodeName, n.Description, n.Properties, n.Config)
		if err != nil {
			return nil, fmt.Errorf("export rule %q: %w", n.NodeName, err)
		}

		if i, dup := byKey[ruleKey]; dup {
			if origin == models.ValidationRuleOriginCore {
				bundle.Rules[i] = *spec // core wins
			}
			continue
		}
		byKey[ruleKey] = len(bundle.Rules)
		bundle.Rules = append(bundle.Rules, *spec)
	}

	sort.Slice(bundle.Rules, func(i, j int) bool { return bundle.Rules[i].RuleKey < bundle.Rules[j].RuleKey })

	if err := bundle.ComputeChecksum(); err != nil {
		return nil, fmt.Errorf("compute bundle checksum: %w", err)
	}
	return bundle, nil
}

func tenantScopeFor(originFilter, tenantID, gold string) string {
	if originFilter == models.ValidationRuleOriginCore {
		return models.ValidationRuleOriginCore
	}
	return tenantID
}

// portableSpecFromNode converts a catalog_node row into a PortableRuleSpec.
// It is pure (no DB) so the no-UUID-leak guarantees can be unit-tested
// without a database.
//
// rule_key resolution: properties->>'rule_key' if present, else node_name
// (legacy rows). BindingIDs are deliberately NOT exported: they are physical
// UUIDs. Logical binding export requires resolving binding node names and is
// deferred until the binding naming scheme is finalized (see plan notes).
func portableSpecFromNode(nodeID uuid.UUID, nodeName, description string, properties, config json.RawMessage) (*models.PortableRuleSpec, string, error) {
	var props models.ValidationRuleProperties
	if len(properties) > 0 {
		if err := json.Unmarshal(properties, &props); err != nil {
			return nil, "", fmt.Errorf("parse properties: %w", err)
		}
	}

	ruleKey := props.BoRuleKey() // properties->>'rule_key' with node_name fallback
	if ruleKey == "" {
		ruleKey = nodeName
	}

	// AST: parse from config, then canonicalize. Never pass through raw bytes —
	// stored ASTs may carry arbitrary whitespace/key ordering from prior edits,
	// which would make checksums environment-dependent.
	if len(config) == 0 {
		return nil, "", fmt.Errorf("rule has no config (missing rule_ast)")
	}
	var cfg models.ValidationRuleConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, "", fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.RuleAST) == 0 {
		return nil, "", fmt.Errorf("rule has no rule_ast")
	}
	var node vm.RuleNode
	if err := json.Unmarshal(cfg.RuleAST, &node); err != nil {
		return nil, "", fmt.Errorf("parse rule_ast: %w", err)
	}
	canonicalAST, err := vm.Compact(node)
	if err != nil {
		return nil, "", fmt.Errorf("canonicalize rule_ast: %w", err)
	}

	governance := props.GovernanceStatus
	if governance == "" {
		governance = models.ValidationRuleGovernanceDraft
	}

	spec := &models.PortableRuleSpec{
		RuleKey:          ruleKey,
		Name:             nodeName,
		BOName:           props.BOName,
		Description:      description,
		Domain:           props.Domain,
		Severity:         props.Severity,
		Timing:           props.Timing,
		Category:         props.Category,
		GovernanceStatus: governance,
		RuleAST:          canonicalAST,
	}
	return spec, ruleKey, nil
}
