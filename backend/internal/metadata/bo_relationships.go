package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
)

// Scanner FOREIGN_KEY catalog_edge_types.id (ansi_scanner.go EDGE_TYPE_FOREIGN_KEY).
var fkEdgeTypeID = uuid.MustParse("f21b4a8f-05af-43b9-92cd-061265ed54e0")

type fkGraphEdge struct {
	ID            string
	SourceNodeID  string
	TargetNodeID  string
	Properties    []byte
	SourceName    string
	TargetName    string
	SourcePath    string
	TargetPath    string
}

type boMeta struct {
	ID   string
	Name string
	Key  string
}

func (s *BusinessObjectService) resolveDrivingTableNode(ctx context.Context, tenantID, boID string) (string, error) {
	var nodeID sql.NullString
	err := s.db.GetContext(ctx, &nodeID, `
		SELECT driving_node_id::text
		FROM public.business_object_binding
		WHERE bo_id = $1::uuid AND tenant_id = $2::uuid
		ORDER BY is_default DESC, created_at DESC
		LIMIT 1
	`, boID, tenantID)
	if err == nil && nodeID.Valid && nodeID.String != "" {
		return nodeID.String, nil
	}
	err = s.db.GetContext(ctx, &nodeID, `
		SELECT driver_table_id::text
		FROM public.business_objects
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, boID, tenantID)
	if err != nil {
		return "", err
	}
	if !nodeID.Valid || nodeID.String == "" {
		return "", nil
	}
	return nodeID.String, nil
}

func (s *BusinessObjectService) drivingNodeToBO(ctx context.Context, tenantID string) (map[string]string, map[string]boMeta, error) {
	nodeToBO := map[string]string{}
	bos := map[string]boMeta{}

	type boRow struct {
		ID   string         `db:"id"`
		Name string         `db:"bo_name"`
		Key  string         `db:"bo_key"`
		Node sql.NullString `db:"node_id"`
	}

	var fromBO []boRow
	if err := s.db.SelectContext(ctx, &fromBO, `
		SELECT id::text AS id, bo_name, bo_key, driver_table_id::text AS node_id
		FROM public.business_objects
		WHERE tenant_id = $1::uuid
	`, tenantID); err != nil {
		return nil, nil, err
	}
	for _, r := range fromBO {
		bos[r.ID] = boMeta{ID: r.ID, Name: r.Name, Key: r.Key}
		if r.Node.Valid && r.Node.String != "" {
			if existing, ok := nodeToBO[r.Node.String]; ok && existing != r.ID {
				logging.GetLogger().Sugar().Warnf("two BOs claim driving table %s: %s and %s", r.Node.String, existing, r.ID)
			} else {
				nodeToBO[r.Node.String] = r.ID
			}
		}
	}

	var fromBind []boRow
	if err := s.db.SelectContext(ctx, &fromBind, `
		SELECT b.bo_id::text AS id, COALESCE(bo.bo_name, '') AS bo_name, COALESCE(bo.bo_key, '') AS bo_key, b.driving_node_id::text AS node_id
		FROM public.business_object_binding b
		JOIN public.business_objects bo ON bo.id = b.bo_id
		WHERE b.tenant_id = $1::uuid
	`, tenantID); err != nil {
		return nil, nil, err
	}
	for _, r := range fromBind {
		if r.Name != "" {
			bos[r.ID] = boMeta{ID: r.ID, Name: r.Name, Key: r.Key}
		}
		if r.Node.Valid && r.Node.String != "" {
			if existing, ok := nodeToBO[r.Node.String]; ok && existing != r.ID {
				logging.GetLogger().Sugar().Warnf("two BOs claim driving table %s: %s and %s", r.Node.String, existing, r.ID)
			} else {
				nodeToBO[r.Node.String] = r.ID
			}
		}
	}
	return nodeToBO, bos, nil
}

func (s *BusinessObjectService) loadFKEdges(ctx context.Context, tenantID string) ([]fkGraphEdge, error) {
	type row struct {
		ID         string          `db:"id"`
		Source     string          `db:"source_node_id"`
		Target     string          `db:"target_node_id"`
		Properties json.RawMessage `db:"properties"`
		SourceName string          `db:"source_name"`
		TargetName string          `db:"target_name"`
		SourcePath string          `db:"source_path"`
		TargetPath string          `db:"target_path"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT e.id::text AS id,
		       e.source_node_id::text AS source_node_id,
		       e.target_node_id::text AS target_node_id,
		       COALESCE(e.properties, '{}'::jsonb) AS properties,
		       COALESCE(src.node_name, '') AS source_name,
		       COALESCE(tgt.node_name, '') AS target_name,
		       COALESCE(src.qualified_path, '') AS source_path,
		       COALESCE(tgt.qualified_path, '') AS target_path
		FROM public.catalog_edge e
		LEFT JOIN public.catalog_node src ON src.id = e.source_node_id
		LEFT JOIN public.catalog_node tgt ON tgt.id = e.target_node_id
		WHERE e.tenant_id = $1::uuid
		  AND e.edge_type_id = $2::uuid
	`, tenantID, fkEdgeTypeID)
	if err != nil {
		return nil, err
	}
	out := make([]fkGraphEdge, 0, len(rows))
	for _, r := range rows {
		out = append(out, fkGraphEdge{
			ID: r.ID, SourceNodeID: r.Source, TargetNodeID: r.Target, Properties: r.Properties,
			SourceName: r.SourceName, TargetName: r.TargetName, SourcePath: r.SourcePath, TargetPath: r.TargetPath,
		})
	}
	return out, nil
}

func parseJoinColumns(props []byte, childTable, parentTable string) (join string, cols []JoinColumn) {
	var m map[string]interface{}
	if err := json.Unmarshal(props, &m); err != nil {
		return "", nil
	}
	raw, _ := m["columns"].([]interface{})
	for _, item := range raw {
		cm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		src, _ := cm["source_column"].(string)
		tgt, _ := cm["target_column"].(string)
		if src == "" || tgt == "" {
			continue
		}
		cols = append(cols, JoinColumn{Source: src, Target: tgt})
	}
	if len(cols) == 0 {
		fk, _ := m["fk_column"].(string)
		ref, _ := m["ref_column"].(string)
		if fk != "" && ref != "" {
			cols = append(cols, JoinColumn{Source: fk, Target: ref})
		}
	}
	if childTable == "" {
		childTable, _ = m["source_table"].(string)
	}
	if parentTable == "" {
		parentTable, _ = m["target_table"].(string)
	}
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		left, right := c.Source, c.Target
		if childTable != "" {
			left = childTable + "." + c.Source
		}
		if parentTable != "" {
			right = parentTable + "." + c.Target
		}
		parts = append(parts, left+" = "+right)
	}
	return strings.Join(parts, " AND "), cols
}

func storedCardinality(props []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(props, &m); err != nil {
		return "N:1"
	}
	c, _ := m["cardinality"].(string)
	c = strings.ToUpper(strings.TrimSpace(c))
	if c == "" {
		return "N:1"
	}
	return c
}

func flipCardinality(card string) string {
	switch strings.ToUpper(card) {
	case "N:1", "M:1", "MANY:1":
		return "1:N"
	case "1:N", "1:M", "1:MANY":
		return "N:1"
	default:
		return card
	}
}

func (s *BusinessObjectService) relatedObjectsFromFKGraph(ctx context.Context, tenantID, boID, drivingNode string) ([]RelationshipResult, int, error) {
	edges, err := s.loadFKEdges(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}
	nodeToBO, bos, err := s.drivingNodeToBO(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	adj := map[string][]fkGraphEdge{}
	touching := 0
	for _, e := range edges {
		adj[e.SourceNodeID] = append(adj[e.SourceNodeID], e)
		adj[e.TargetNodeID] = append(adj[e.TargetNodeID], e)
		if e.SourceNodeID == drivingNode || e.TargetNodeID == drivingNode {
			touching++
		}
	}

	claimantsAt := func(start string) []string {
		type hop struct {
			node  string
			depth int
		}
		seen := map[string]int{start: 0}
		q := []hop{{start, 0}}
		type claim struct {
			bo    string
			depth int
		}
		var found []claim
		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			if cur.depth >= 2 {
				continue
			}
			for _, e := range adj[cur.node] {
				next := e.TargetNodeID
				if next == cur.node {
					next = e.SourceNodeID
				}
				if _, ok := seen[next]; ok {
					continue
				}
				seen[next] = cur.depth + 1
				if owner, ok := nodeToBO[next]; ok && owner != boID {
					found = append(found, claim{bo: owner, depth: cur.depth + 1})
				}
				q = append(q, hop{next, cur.depth + 1})
			}
		}
		best := map[string]int{}
		for _, c := range found {
			if d, ok := best[c.bo]; !ok || c.depth < d {
				best[c.bo] = c.depth
			}
		}
		ids := make([]string, 0, len(best))
		minD := 99
		for _, d := range best {
			if d < minD {
				minD = d
			}
		}
		for id, d := range best {
			if d == minD {
				ids = append(ids, id)
			}
		}
		if len(ids) > 1 {
			logging.GetLogger().Sugar().Warnf("ambiguous related-table claimants for node %s: %v", start, ids)
		}
		return ids
	}

	seen := map[string]bool{}
	var out []RelationshipResult
	emit := func(r RelationshipResult) {
		key := r.TargetObjectID + "|" + r.JoinCondition + "|" + r.Kind
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, r)
	}

	for _, e := range edges {
		if e.SourceNodeID != drivingNode && e.TargetNodeID != drivingNode {
			continue
		}
		if e.SourceNodeID == e.TargetNodeID {
			continue
		}
		drivingIsParent := e.TargetNodeID == drivingNode
		other := e.SourceNodeID
		otherName, otherPath := e.SourceName, e.SourcePath
		childTable, parentTable := e.SourceName, e.TargetName
		if !drivingIsParent {
			other = e.TargetNodeID
			otherName, otherPath = e.TargetName, e.TargetPath
		}
		join, cols := parseJoinColumns(e.Properties, childTable, parentTable)
		card := storedCardinality(e.Properties)
		if drivingIsParent {
			card = flipCardinality(card)
		}

		owner, isDriving := nodeToBO[other]
		if isDriving && owner == boID {
			continue
		}
		if isDriving {
			meta := bos[owner]
			name := meta.Name
			if name == "" {
				name = otherName
			}
			emit(RelationshipResult{
				ID: e.ID, RelatedObjectName: name, TargetObjectID: owner,
				RelationshipType: "foreign_key", Cardinality: card, JoinCondition: join, JoinColumns: cols,
				SourceDriverTable: e.SourceName, TargetDriverTable: e.TargetName,
				Kind: "bo", SourceQualifiedPath: e.SourcePath, TargetQualifiedPath: e.TargetPath,
				Description: otherPath,
			})
			continue
		}

		claimants := claimantsAt(other)
		if len(claimants) == 1 {
			meta := bos[claimants[0]]
			name := meta.Name
			if name == "" {
				name = otherName
			}
			nn := card
			// Junction: other table also FKs to the claimant's driving node.
			claimantDriving := ""
			for node, bo := range nodeToBO {
				if bo == claimants[0] {
					claimantDriving = node
					break
				}
			}
			if claimantDriving != "" {
				for _, e2 := range adj[other] {
					if (e2.SourceNodeID == other && e2.TargetNodeID == claimantDriving) ||
						(e2.TargetNodeID == other && e2.SourceNodeID == claimantDriving) {
						nn = "N:N"
						break
					}
				}
			}
			emit(RelationshipResult{
				ID: e.ID, RelatedObjectName: name, TargetObjectID: claimants[0],
				RelationshipType: "foreign_key", Cardinality: nn, JoinCondition: join, JoinColumns: cols,
				SourceDriverTable: e.SourceName, TargetDriverTable: e.TargetName,
				Kind: "bo", LinkTable: otherName, SourceQualifiedPath: e.SourcePath, TargetQualifiedPath: e.TargetPath,
				Description: otherPath,
			})
			continue
		}
		if len(claimants) == 0 {
			emit(RelationshipResult{
				ID: e.ID, RelatedObjectName: otherName, TargetObjectID: boID,
				RelationshipType: "foreign_key", Cardinality: card, JoinCondition: join, JoinColumns: cols,
				SourceDriverTable: e.SourceName, TargetDriverTable: e.TargetName,
				Kind: "relatedTable", LinkTable: otherName,
				SourceQualifiedPath: e.SourcePath, TargetQualifiedPath: e.TargetPath,
				Description: otherPath,
			})
		}
	}

	return out, touching, nil
}

// legacyCatalogRelationships is the pre-scanner-backfill path: catalog edges
// that already have a non-related_to relationship_type, plus information_schema
// join fill-in. Only used when the driving node has zero foreign_key edges.
func (s *BusinessObjectService) legacyCatalogRelationships(ctx context.Context, tenantID, boID, drivingNode string, response *BORelationshipsResponse) {
	var rows []RelationshipResult
	err := s.db.SelectContext(ctx, &rows, `
		SELECT DISTINCT
			e.id::text as id,
			CASE
				WHEN e.source_node_id = $1::uuid OR (e.properties->>'source_bo_id') = $2 THEN COALESCE(t.node_name, t.qualified_path, e.properties->>'target_bo_id', 'Related BO')
				ELSE COALESCE(src.node_name, src.qualified_path, e.properties->>'source_bo_id', 'Related BO')
			END as related_object_name,
			CASE
				WHEN e.source_node_id = $1::uuid OR (e.properties->>'source_bo_id') = $2 THEN COALESCE(e.properties->>'target_bo_id', e.target_node_id::text, '')
				ELSE COALESCE(e.properties->>'source_bo_id', e.source_node_id::text, '')
			END as target_object_id,
			COALESCE(e.relationship_type, 'RELATED_TO') as relationship_type,
			COALESCE(e.properties->>'cardinality', '1:N') as cardinality,
			COALESCE(e.properties->>'description', t.qualified_path, src.qualified_path, '') as description,
			COALESCE(e.properties->>'join_condition', e.properties->>'description', '') as join_condition,
			COALESCE(src.node_name, '') as source_driver_table,
			COALESCE(t.node_name, '') as target_driver_table,
			COALESCE(src.qualified_path, '') as source_qualified_path,
			COALESCE(t.qualified_path, '') as target_qualified_path
		FROM catalog_edge e
		LEFT JOIN catalog_node src ON e.source_node_id = src.id
		LEFT JOIN catalog_node t ON e.target_node_id = t.id
		WHERE (
			e.source_node_id = $1::uuid OR e.target_node_id = $1::uuid
			OR e.source_node_id = $2::uuid OR e.target_node_id = $2::uuid
			OR (e.properties->>'source_bo_id') = $2 OR (e.properties->>'target_bo_id') = $2
		)
		AND e.relationship_type NOT IN (
			'related_to', 'MAPS_TO', 'USES_SEMANTIC_TERM', 'BACKED_BY_TERM',
			'HAS_FIELD', 'has_context', 'member_of', 'depends_on',
			'contains_field', 'contains_endpoint', 'contains_resource'
		)
	`, drivingNode, boID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("legacy related objects for BO %s: %v", boID, err)
		return
	}
	nodeToBO, bos, mapErr := s.drivingNodeToBO(ctx, tenantID)
	if mapErr != nil {
		nodeToBO = map[string]string{}
		bos = map[string]boMeta{}
	}
	for i := range rows {
		rel := &rows[i]
		if owner, ok := nodeToBO[rel.TargetObjectID]; ok {
			rel.TargetObjectID = owner
			if m, ok := bos[owner]; ok && m.Name != "" {
				rel.RelatedObjectName = m.Name
			}
		}
		if rel.Kind == "" {
			rel.Kind = "bo"
		}
		if rel.JoinCondition == "" {
			lowerType := strings.ToLower(rel.RelationshipType)
			if lowerType == "foreign_key" || lowerType == "belongs_to" {
				srcSchema, srcTable := qualifiedPathToSchemaTable(rel.SourceQualifiedPath)
				tgtSchema, tgtTable := qualifiedPathToSchemaTable(rel.TargetQualifiedPath)
				if srcTable != "" && tgtTable != "" {
					if fk, ok := s.resolveRealForeignKey(ctx, srcSchema, srcTable, tgtSchema, tgtTable); ok {
						rel.JoinCondition = fk
						if rel.Cardinality == "" {
							rel.Cardinality = "1:N"
						}
					}
				}
			}
		}
	}
	response.RelatedObjects = rows
}
