package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

var (
	isinRegex  = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{9}[0-9]$`)
	cusipRegex = regexp.MustCompile(`^[0-9A-Z]{9}$`)
	sedolRegex = regexp.MustCompile(`^[0-9B-DF-HJ-NP-TV-Z]{7}$`)
	figiRegex  = regexp.MustCompile(`^BBG[0-9A-Z]{9}$`)
	leiRegex   = regexp.MustCompile(`^[0-9A-Z]{18}[0-9]{2}$`)
)

// CatalogMutationEvent represents a metadata or schema mutation event.
type CatalogMutationEvent struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	DatasourceID uuid.UUID `json:"datasource_id"`
	NodeID       uuid.UUID `json:"node_id"`
	Action       string    `json:"action"` // INSERT, UPDATE, SCHEMA_DRIFT
}

// TermDisambiguator defines node comparison logic for distinction reasoning.
type TermDisambiguator interface {
	CompareNodes(sourceName, targetName string) (diffRationale string, isSynonym bool)
}

// RuleBasedTermDisambiguator provides deterministic distinction rules across enterprise domains.
type RuleBasedTermDisambiguator struct{}

// NewRuleBasedTermDisambiguator creates a default rule-based disambiguator.
func NewRuleBasedTermDisambiguator() *RuleBasedTermDisambiguator {
	return &RuleBasedTermDisambiguator{}
}

// CompareNodes evaluates distinction rationale or synonymy between two terms.
func (d *RuleBasedTermDisambiguator) CompareNodes(sourceName, targetName string) (string, bool) {
	s := strings.ToUpper(strings.TrimSpace(sourceName))
	t := strings.ToUpper(strings.TrimSpace(targetName))

	if s == t {
		return "Exact semantic match", true
	}

	// Exact Synonyms
	if (s == "CUST_ID" && t == "CUSTOMER_IDENTIFIER") || (s == "CUSTOMER_IDENTIFIER" && t == "CUST_ID") ||
		(s == "TRADES" && t == "ORDER") || (s == "ORDER" && t == "TRADES") ||
		(s == "PX" && t == "PRICE") || (s == "PRICE" && t == "PX") ||
		(s == "CCY" && t == "CURRENCY") || (s == "CURRENCY" && t == "CCY") {
		return "Lexical alias denoting equivalent domain concept", true
	}

	// Account Family Differentiations
	if strings.Contains(s, "ACCOUNT") && strings.Contains(t, "ACCOUNT") {
		if strings.Contains(s, "ALLOCATION") && strings.Contains(t, "CUSTODIAL") {
			return "Allocation accounts route post-trade execution sleeve fills; Custodial accounts identify external bank safekeeping depositories.", false
		}
		if strings.Contains(s, "GL") || strings.Contains(t, "GL") {
			return "GL Account Code is strictly for double-entry trial balance ledger posting.", false
		}
		if strings.Contains(s, "CLIENT") || strings.Contains(t, "CLIENT") {
			return "Client Account Number represents the investor legal customer account entity.", false
		}
		return "Specialized account sub-type with distinct operational lifecycle boundaries.", false
	}

	// Symbology Family Differentiations
	if (s == "ISIN" || s == "CUSIP" || s == "SEDOL" || s == "FIGI") && (t == "ISIN" || t == "CUSIP" || t == "SEDOL" || t == "FIGI") {
		return fmt.Sprintf("Symbology peer in Capital Markets (%s vs %s) with distinct issuer country prefixes and validation checksums.", sourceName, targetName), false
	}

	// Date Lifecycle Differentiations
	if strings.Contains(s, "DATE") && strings.Contains(t, "DATE") {
		if strings.Contains(s, "TRADE") && strings.Contains(t, "SETTLE") {
			return "Trade Date is market execution agreement date (T); Settlement Date is when cash and securities are delivered.", false
		}
	}

	return fmt.Sprintf("Distinct domain concepts with operational differences between %s and %s.", sourceName, targetName), false
}

// MetadataEvolutionDaemon continuously monitors schema mutations, value profiles, and topology shifts.
type MetadataEvolutionDaemon struct {
	db           *sqlx.DB
	aiClassifier TermDisambiguator
}

// NewMetadataEvolutionDaemon creates a new evolution daemon instance.
func NewMetadataEvolutionDaemon(db *sqlx.DB, classifier TermDisambiguator) *MetadataEvolutionDaemon {
	if classifier == nil {
		classifier = NewRuleBasedTermDisambiguator()
	}
	return &MetadataEvolutionDaemon{
		db:           db,
		aiClassifier: classifier,
	}
}

// ProcessCatalogEvent executes the continuous evolution pipeline on a mutated catalog node.
func (d *MetadataEvolutionDaemon) ProcessCatalogEvent(ctx context.Context, evt CatalogMutationEvent) error {
	if d.db == nil {
		return nil
	}

	// 1. Fetch node attributes and sample values
	var nodeKey, nodeName, dataType string
	var rawProperties []byte
	query := `
		SELECT node_name, qualified_path, properties->>'data_type', properties
		FROM catalog_node 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id = '00000000-0000-0000-0000-000000000000' OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		LIMIT 1`
	err := d.db.QueryRowContext(ctx, query, evt.NodeID.String(), evt.TenantID.String()).Scan(&nodeName, &nodeKey, &dataType, &rawProperties)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("failed resolving mutated node: %w", err)
	}

	// 2. Identify candidate symbology / format peers via regex heuristics
	if peerEdgeType, family := d.DetectSymbologyFamily(nodeName, rawProperties); peerEdgeType != "" {
		if err := d.LinkSymbologyPeers(ctx, evt.TenantID, evt.NodeID, family); err != nil {
			logging.GetLogger().Sugar().Warnf("Error linking symbology peers: %v", err)
		}
	}

	// 3. Cluster with nearest semantic neighbors using lexical token overlap & unlinked adjacency
	if err := d.DiscoverDifferentiationsAndSynonyms(ctx, evt.TenantID, evt.NodeID, nodeName); err != nil {
		logging.GetLogger().Sugar().Warnf("Error discovering differentiations: %v", err)
	}

	// 4. Invalidate Business Object fields if schema drift or column alteration occurred
	if strings.Contains(evt.Action, "DRIFT") || strings.Contains(evt.Action, "DROP") || strings.Contains(evt.Action, "ALTER") {
		resilience := NewBOResilienceEngine(d.db)
		if affected, err := resilience.HandleSchemaDrift(ctx, evt.TenantID, evt.DatasourceID, nodeKey, nodeName, evt.Action); err == nil && affected > 0 {
			logging.GetLogger().Sugar().Infof("Marked %d business object fields as DRIFT_DEGRADED for %s", affected, nodeName)
		}
	}

	return nil
}

// DetectSymbologyFamily inspects column names and sampled values against international symbology formats.
func (d *MetadataEvolutionDaemon) DetectSymbologyFamily(nodeName string, props []byte) (string, string) {
	norm := strings.ToUpper(nodeName)
	if strings.Contains(norm, "ISIN") {
		return "IS_PEER_IDENTIFIER_OF", "ISO_6166_SECURITY_IDENTIFIERS"
	}
	if strings.Contains(norm, "CUSIP") {
		return "IS_PEER_IDENTIFIER_OF", "NORTH_AMERICA_SECURITY_IDENTIFIERS"
	}
	if strings.Contains(norm, "SEDOL") {
		return "IS_PEER_IDENTIFIER_OF", "UK_EUROPE_SECURITY_IDENTIFIERS"
	}
	if strings.Contains(norm, "FIGI") || strings.Contains(norm, "BBG") {
		return "IS_PEER_IDENTIFIER_OF", "OPEN_SYMBOLOGY_IDENTIFIERS"
	}
	if strings.Contains(norm, "LEI") {
		return "IS_PEER_IDENTIFIER_OF", "LEGAL_ENTITY_IDENTIFIERS"
	}

	if len(props) > 0 {
		var p struct {
			SampleValues []string `json:"sample_values"`
		}
		if err := json.Unmarshal(props, &p); err == nil && len(p.SampleValues) > 0 {
			for _, val := range p.SampleValues {
				valTrim := strings.TrimSpace(val)
				if isinRegex.MatchString(valTrim) {
					return "IS_PEER_IDENTIFIER_OF", "ISO_6166_SECURITY_IDENTIFIERS"
				}
				if cusipRegex.MatchString(valTrim) {
					return "IS_PEER_IDENTIFIER_OF", "NORTH_AMERICA_SECURITY_IDENTIFIERS"
				}
				if sedolRegex.MatchString(valTrim) {
					return "IS_PEER_IDENTIFIER_OF", "UK_EUROPE_SECURITY_IDENTIFIERS"
				}
				if figiRegex.MatchString(valTrim) {
					return "IS_PEER_IDENTIFIER_OF", "OPEN_SYMBOLOGY_IDENTIFIERS"
				}
				if leiRegex.MatchString(valTrim) {
					return "IS_PEER_IDENTIFIER_OF", "LEGAL_ENTITY_IDENTIFIERS"
				}
			}
		}
	}
	return "", ""
}

// LinkSymbologyPeers links symbology peers in the catalog_edge table.
func (d *MetadataEvolutionDaemon) LinkSymbologyPeers(ctx context.Context, tenantID, sourceNodeID uuid.UUID, family string) error {
	if d.db == nil {
		return nil
	}

	query := `
		INSERT INTO catalog_edge (id, tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at)
		SELECT 
			gen_random_uuid(), $1, $2, cn.id, 'IS_PEER_IDENTIFIER_OF', cet.id,
			jsonb_build_object('symbology_family', $3, 'discovered_at', CURRENT_TIMESTAMP, 'auto_generated', true),
			NOW(), NOW()
		FROM catalog_node cn
		CROSS JOIN catalog_edge_type cet
		WHERE cet.edge_type_name = 'IS_PEER_IDENTIFIER_OF'
		  AND cn.id != $2
		  AND (cn.tenant_id = $1::text OR cn.tenant_id = (SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1))
		  AND (cn.properties->>'symbology_family' = $3 OR cn.node_name ILIKE '%ISIN%' OR cn.node_name ILIKE '%CUSIP%' OR cn.node_name ILIKE '%SEDOL%')
		ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING`
	_, err := d.db.ExecContext(ctx, query, tenantID.String(), sourceNodeID.String(), family)
	return err
}

// DiscoverDifferentiationsAndSynonyms clusters near-neighbor nodes and inserts proposed relationship edges.
func (d *MetadataEvolutionDaemon) DiscoverDifferentiationsAndSynonyms(ctx context.Context, tenantID, sourceNodeID uuid.UUID, nodeName string) error {
	if d.db == nil {
		return nil
	}

	query := `
		SELECT cn.id, cn.node_name
		FROM catalog_node cn
		WHERE cn.id != $1
		  AND (cn.tenant_id = $2::text OR cn.tenant_id = (SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1))
		  AND (cn.node_name ILIKE '%' || $3 || '%' OR $3 ILIKE '%' || cn.node_name || '%')
		  AND NOT EXISTS (
		      SELECT 1 FROM catalog_edge ce
		      WHERE (ce.source_id = $1 AND ce.target_id = cn.id)
		         OR (ce.source_id = cn.id AND ce.target_id = $1)
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM catalog_edge_rejection_store r
		      WHERE r.tenant_id = $2
		        AND ( (r.source_node_id = $1 AND r.rejected_target_id = cn.id) OR (r.source_node_id = cn.id AND r.rejected_target_id = $1) )
		  )
		LIMIT 5`

	rows, err := d.db.QueryContext(ctx, query, sourceNodeID.String(), tenantID.String(), nodeName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var targetID string
		var targetName string
		if err := rows.Scan(&targetID, &targetName); err != nil {
			continue
		}

		diffRationale, isSynonym := d.aiClassifier.CompareNodes(nodeName, targetName)
		edgeType := "DIFFERENTIATED_FROM"
		if isSynonym {
			edgeType = "HAS_SYNONYM"
		}

		edgeInsert := `
			INSERT INTO catalog_edge (id, tenant_id, source_id, target_id, edge_type_name, properties, created_at, updated_at)
			VALUES (
				gen_random_uuid(), $1, $2, $3, $4,
				jsonb_build_object('distinction_rationale', $5, 'confidence', 0.85, 'suggested', true),
				NOW(), NOW()
			) ON CONFLICT (tenant_id, source_id, target_id, edge_type_name) DO NOTHING`
		_, _ = d.db.ExecContext(ctx, edgeInsert, tenantID.String(), sourceNodeID.String(), targetID, edgeType, diffRationale)
	}
	return nil
}
