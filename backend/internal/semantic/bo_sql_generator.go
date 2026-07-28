package semantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\${([^}]+)}`)

// Level 1: Domain (e.g., "Financials", "Operations")
type Domain struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Entities map[string]*Entity `json:"entities"`
}

// Level 2: Entity / Business Object (e.g., "Invoice", "Customer")
type Entity struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	PhysicalSchema string                `json:"physical_schema"` // e.g., "analytics"
	PhysicalTable  string                `json:"physical_table"`  // e.g., "invoices"
	Attributes     map[string]*Attribute `json:"attributes"`
	Edges          []*Edge               `json:"edges"` // Relationships to other entities
	DefaultFilters []string              `json:"default_filters"`
}

// Level 3: Attribute (e.g., "total_amount", "customer_name")
type AttributeType string

const (
	Dimension         AttributeType = "DIMENSION"  // Group by fields
	Measure           AttributeType = "MEASURE"    // Aggregated fields
	CalculatedMeasure AttributeType = "CALCULATED" // Derived via expression
)

type Attribute struct {
	ID             string        `json:"id"`
	EntityID       string        `json:"entity_id"`
	Name           string        `json:"name"`
	Type           AttributeType `json:"type"`
	PhysicalColumn string        `json:"physical_column"`
	AggFunction    string        `json:"agg_function"` // e.g., "SUM", "AVG", "COUNT"
	Expression     string        `json:"expression"`   // For CALCULATED measures e.g. "${Revenue} - ${Cost}"
	Format         string        `json:"format"`       // e.g. "currency", "percentage"
	Description    string        `json:"description"`  // For AI agent grounding
}

// The Graph Edge: Defines how Entities JOIN together
type Edge struct {
	TargetEntityID string `json:"target_entity_id"`
	RoleName       string `json:"role_name"`      // Role-playing dimension alias (e.g., "ShipDate")
	JoinType       string `json:"join_type"`      // "INNER", "LEFT"
	JoinCondition  string `json:"join_condition"` // e.g., "source.customer_id = target.id"
}

type SemanticFilter struct {
	Attribute string `json:"attribute"` // "Entity.Attribute"
	Operator  string `json:"operator"`  // "=", ">=", "<=", "LIKE"
	Value     string `json:"value"`
}

type SemanticRequest struct {
	Domain     string           `json:"domain"`
	Dimensions []string         `json:"dimensions"` // e.g., ["Customer.Region", "Order.OrderMonth"]
	Measures   []string         `json:"measures"`   // e.g., ["OrderDetail.GrossRevenue", "OrderDetail.GrossProfit"]
	Filters    []SemanticFilter `json:"filters"`
}

type GeneratorContext struct {
	TenantKey string
	Dialect   string // "DATAFUSION" (Iceberg) or "STARROCKS"
	Graph     *Domain
}

// ResolveCalculatedMeasures builds a topological sort DAG for derived metrics and compiles their expressions
func (g *GeneratorContext) ResolveCalculatedMeasures(requestedMeasures []string) ([]string, map[string]string, error) {
	exprMap := make(map[string]string)
	adjacency := make(map[string][]string)
	inDegree := make(map[string]int)
	attributeRegistry := make(map[string]*Attribute)

	for _, entity := range g.Graph.Entities {
		for _, attr := range entity.Attributes {
			if attr.Type == Measure || attr.Type == CalculatedMeasure {
				fullName := fmt.Sprintf("%s.%s", entity.Name, attr.Name)
				attributeRegistry[fullName] = attr
				inDegree[fullName] = 0
			}
		}
	}

	for _, measureName := range requestedMeasures {
		attr, exists := attributeRegistry[measureName]
		if !exists {
			return nil, nil, fmt.Errorf("measure %s not found in semantic registry", measureName)
		}

		if attr.Type == CalculatedMeasure {
			matches := placeholderRegex.FindAllStringSubmatch(attr.Expression, -1)
			for _, match := range matches {
				depName := match[1]
				if _, ok := attributeRegistry[depName]; !ok {
					return nil, nil, fmt.Errorf("unresolved dependency %s in calculated measure %s", depName, measureName)
				}
				adjacency[depName] = append(adjacency[depName], measureName)
				inDegree[measureName]++
			}
		}
	}

	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var sortedOrder []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sortedOrder = append(sortedOrder, curr)

		for _, neighbor := range adjacency[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	compiledExpressions := make(map[string]string)
	for _, name := range sortedOrder {
		attr, exists := attributeRegistry[name]
		if !exists {
			continue
		}
		entityName := strings.Split(name, ".")[0]

		if attr.Type == Measure {
			aggFunc := attr.AggFunction
			if aggFunc == "" {
				aggFunc = "SUM"
			}
			compiledExpressions[name] = fmt.Sprintf("%s(%s.%s)", aggFunc, entityName, attr.PhysicalColumn)
		} else if attr.Type == CalculatedMeasure {
			sqlExpr := attr.Expression
			for _, match := range placeholderRegex.FindAllStringSubmatch(attr.Expression, -1) {
				depKey := match[1]
				depSQL := compiledExpressions[depKey]
				sqlExpr = strings.ReplaceAll(sqlExpr, fmt.Sprintf("${%s}", depKey), fmt.Sprintf("(%s)", depSQL))
			}
			compiledExpressions[name] = sqlExpr
		}
	}

	return sortedOrder, compiledExpressions, nil
}

// GenerateSQL compiles a semantic request into dialect-aware physical SQL
func (g *GeneratorContext) GenerateSQL(req SemanticRequest) (string, error) {
	if g.Graph == nil {
		return "", fmt.Errorf("semantic graph not loaded")
	}

	requiredEntities, err := g.extractRequiredEntities(req)
	if err != nil {
		return "", err
	}
	if len(requiredEntities) == 0 {
		return "", fmt.Errorf("no entities found in semantic request")
	}

	requiredEntityIDs := make([]string, len(requiredEntities))
	for i, e := range requiredEntities {
		requiredEntityIDs[i] = e.ID
	}

	joinPaths, err := g.findShortestJoinPath(requiredEntityIDs)
	if err != nil {
		return "", fmt.Errorf("no valid relationship path found: %w", err)
	}

	selects := []string{}
	groupBys := []string{}

	for _, dim := range req.Dimensions {
		attr, err := g.resolveAttribute(dim)
		if err != nil {
			return "", err
		}
		entity := g.Graph.Entities[attr.EntityID]
		tableAlias := entity.Name

		colExpr := fmt.Sprintf("%s.%s", tableAlias, attr.PhysicalColumn)
		selects = append(selects, fmt.Sprintf("%s AS %s", colExpr, sanitizeAlias(dim)))
		groupBys = append(groupBys, colExpr)
	}

	_, compiledMeasures, err := g.ResolveCalculatedMeasures(req.Measures)
	if err != nil {
		return "", err
	}

	for _, meas := range req.Measures {
		expr, exists := compiledMeasures[meas]
		if !exists {
			attr, err := g.resolveAttribute(meas)
			if err != nil {
				return "", err
			}
			entity := g.Graph.Entities[attr.EntityID]
			aggFunc := attr.AggFunction
			if aggFunc == "" {
				aggFunc = "SUM"
			}
			expr = fmt.Sprintf("%s(%s.%s)", aggFunc, entity.Name, attr.PhysicalColumn)
		}
		selects = append(selects, fmt.Sprintf("%s AS %s", expr, sanitizeAlias(meas)))
	}

	baseEntity := requiredEntities[0]
	fromClause := fmt.Sprintf("FROM %s %s", g.formatTableName(baseEntity), baseEntity.Name)

	joinClauses := []string{}
	for _, edge := range joinPaths {
		targetEntity, exists := g.Graph.Entities[edge.TargetEntityID]
		if !exists {
			continue
		}
		alias := targetEntity.Name
		if edge.RoleName != "" {
			alias = edge.RoleName
		}
		joinStmt := fmt.Sprintf("%s JOIN %s %s ON %s",
			edge.JoinType,
			g.formatTableName(targetEntity),
			alias,
			edge.JoinCondition)
		joinClauses = append(joinClauses, joinStmt)
	}

	wheres := []string{}
	for _, filter := range req.Filters {
		attr, err := g.resolveAttribute(filter.Attribute)
		if err != nil {
			return "", err
		}
		entity := g.Graph.Entities[attr.EntityID]
		tableAlias := entity.Name

		wheres = append(wheres, fmt.Sprintf("%s.%s %s '%s'", tableAlias, attr.PhysicalColumn, filter.Operator, filter.Value))
	}

	for _, entity := range requiredEntities {
		for _, df := range entity.DefaultFilters {
			wheres = append(wheres, df)
		}
	}

	sql := fmt.Sprintf("SELECT\n  %s\n%s\n", strings.Join(selects, ",\n  "), fromClause)
	if len(joinClauses) > 0 {
		sql += strings.Join(joinClauses, "\n") + "\n"
	}
	if len(wheres) > 0 {
		sql += "WHERE " + strings.Join(wheres, " AND ") + "\n"
	}
	if len(groupBys) > 0 {
		sql += "GROUP BY " + strings.Join(groupBys, ", ") + "\n"
	}

	return sql, nil
}

func (g *GeneratorContext) formatTableName(e *Entity) string {
	schema := e.PhysicalSchema
	if schema == "" {
		schema = "analytics"
	}
	if g.Dialect == "DATAFUSION" {
		return fmt.Sprintf(`"%s"."%s"."%s"`, g.TenantKey, schema, e.PhysicalTable)
	} else if g.Dialect == "STARROCKS" {
		return fmt.Sprintf(`%s.%s`, schema, e.PhysicalTable)
	}
	return e.PhysicalTable
}

func sanitizeAlias(s string) string {
	return strings.ReplaceAll(s, ".", "_")
}

func (g *GeneratorContext) extractRequiredEntities(req SemanticRequest) ([]*Entity, error) {
	entityMap := make(map[string]*Entity)
	allRef := append([]string{}, req.Dimensions...)
	allRef = append(allRef, req.Measures...)
	for _, f := range req.Filters {
		allRef = append(allRef, f.Attribute)
	}

	for _, ref := range allRef {
		attr, err := g.resolveAttribute(ref)
		if err != nil {
			return nil, err
		}
		if entity, exists := g.Graph.Entities[attr.EntityID]; exists {
			entityMap[entity.ID] = entity
		}
	}

	result := make([]*Entity, 0, len(entityMap))
	for _, e := range entityMap {
		result = append(result, e)
	}
	return result, nil
}

func (g *GeneratorContext) resolveAttribute(ref string) (*Attribute, error) {
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid attribute reference format '%s' (expected Entity.Attribute)", ref)
	}
	entityName, attrName := parts[0], parts[1]

	for _, entity := range g.Graph.Entities {
		if strings.EqualFold(entity.Name, entityName) {
			if attr, exists := entity.Attributes[attrName]; exists {
				return attr, nil
			}
		}
	}
	return nil, fmt.Errorf("attribute '%s' not found in domain graph", ref)
}

func (g *GeneratorContext) findShortestJoinPath(requiredEntityIDs []string) ([]*Edge, error) {
	if len(requiredEntityIDs) <= 1 {
		return []*Edge{}, nil
	}

	rootID := requiredEntityIDs[0]
	targetIDs := make(map[string]bool)
	for _, id := range requiredEntityIDs[1:] {
		targetIDs[id] = true
	}

	type QueueNode struct {
		EntityID string
		Path     []*Edge
	}

	visited := make(map[string]bool)
	queue := []QueueNode{{EntityID: rootID, Path: []*Edge{}}}
	visited[rootID] = true

	var collectedEdges []*Edge
	edgesMap := make(map[string]bool)

	for len(queue) > 0 && len(targetIDs) > 0 {
		curr := queue[0]
		queue = queue[1:]

		entity := g.Graph.Entities[curr.EntityID]
		if entity == nil {
			continue
		}

		for _, edge := range entity.Edges {
			if targetIDs[edge.TargetEntityID] {
				fullPath := append([]*Edge{}, curr.Path...)
				fullPath = append(fullPath, edge)

				for _, e := range fullPath {
					edgeKey := fmt.Sprintf("%s->%s:%s", curr.EntityID, e.TargetEntityID, e.JoinCondition)
					if !edgesMap[edgeKey] {
						edgesMap[edgeKey] = true
						collectedEdges = append(collectedEdges, e)
					}
				}
				delete(targetIDs, edge.TargetEntityID)
			}

			if !visited[edge.TargetEntityID] {
				visited[edge.TargetEntityID] = true
				nextPath := append([]*Edge{}, curr.Path...)
				nextPath = append(nextPath, edge)
				queue = append(queue, QueueNode{EntityID: edge.TargetEntityID, Path: nextPath})
			}
		}
	}

	if len(targetIDs) > 0 {
		return nil, fmt.Errorf("unreachable entities in semantic graph traversal: missing join path")
	}

	return collectedEdges, nil
}

func LoadSemanticGraphFromDB(ctx context.Context, db *sql.DB, tenantID string) (*Domain, error) {
	domain := &Domain{
		ID:       tenantID,
		Name:     fmt.Sprintf("Domain-%s", tenantID),
		Entities: make(map[string]*Entity),
	}

	if db == nil {
		return domain, nil
	}

	nodeRows, err := db.QueryContext(ctx, `
		SELECT id, name, type, physical_schema, physical_table, default_filters
		FROM catalog_node
		WHERE tenant_id = $1 AND type = 'ENTITY'
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog_node entities: %w", err)
	}
	defer nodeRows.Close()

	entityMap := make(map[string]*Entity)

	for nodeRows.Next() {
		var id, name, nodeType string
		var physSchema, physTable sql.NullString
		var defaultFiltersJSON []byte

		if err := nodeRows.Scan(&id, &name, &nodeType, &physSchema, &physTable, &defaultFiltersJSON); err != nil {
			return nil, err
		}

		var filters []string
		if len(defaultFiltersJSON) > 0 {
			_ = json.Unmarshal(defaultFiltersJSON, &filters)
		}

		entityMap[id] = &Entity{
			ID:             id,
			Name:           name,
			PhysicalSchema: physSchema.String,
			PhysicalTable:  physTable.String,
			Attributes:     make(map[string]*Attribute),
			Edges:          []*Edge{},
			DefaultFilters: filters,
		}
	}

	attrRows, err := db.QueryContext(ctx, `
		SELECT id, name, parent_id, physical_column, agg_function, expression, format, description,
		       CASE 
		           WHEN expression IS NOT NULL AND expression != '' THEN 'CALCULATED'
		           WHEN agg_function IS NOT NULL THEN 'MEASURE' 
		           ELSE 'DIMENSION' 
		       END as attr_type
		FROM catalog_node
		WHERE tenant_id = $1 AND type = 'ATTRIBUTE'
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	for attrRows.Next() {
		var id, name string
		var parentID, physCol, aggFunc, expr, format, desc sql.NullString
		var attrTypeStr string

		if err := attrRows.Scan(&id, &name, &parentID, &physCol, &aggFunc, &expr, &format, &desc, &attrTypeStr); err != nil {
			return nil, err
		}

		if parentID.Valid {
			if entity, exists := entityMap[parentID.String]; exists {
				entity.Attributes[name] = &Attribute{
					ID:             id,
					EntityID:       parentID.String,
					Name:           name,
					Type:           AttributeType(attrTypeStr),
					PhysicalColumn: physCol.String,
					AggFunction:    aggFunc.String,
					Expression:     expr.String,
					Format:         format.String,
					Description:    desc.String,
				}
			}
		}
	}

	domain.Entities = entityMap

	edgeRows, err := db.QueryContext(ctx, `
		SELECT source_entity_id, target_entity_id, join_type, join_condition, role_name
		FROM catalog_edge
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog_edge: %w", err)
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var sourceID, targetID, joinType, joinCond string
		var roleName sql.NullString
		if err := edgeRows.Scan(&sourceID, &targetID, &joinType, &joinCond, &roleName); err != nil {
			return nil, err
		}

		if sourceEntity, exists := domain.Entities[sourceID]; exists {
			sourceEntity.Edges = append(sourceEntity.Edges, &Edge{
				TargetEntityID: targetID,
				RoleName:       roleName.String,
				JoinType:       joinType,
				JoinCondition:  joinCond,
			})
		}
	}

	return domain, nil
}

