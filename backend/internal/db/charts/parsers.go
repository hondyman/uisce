package charts

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// createSemanticDataFromYourFormat generates semantic data from a predefined format
func createSemanticDataFromYourFormat() *SemanticLineageChart {
	semanticData := `Category Name	BusinessTermToSemanticTerm	2edcfc55-cc2c-4f14-aa30-4cef4956cfef	CategoryName
CategoryName	SemanticTermToSemanticColumn	10946b9c-f21c-45d5-891a-daaac8acfd58	CategoryName_public_categories_category_name
CategoryName_public_categories_category_name	SemanticColumnToDbColumn	6023bf4d-d8e0-414b-87f5-334ef05d17ea	category_name
Customer ID	BusinessTermToSemanticTerm	038b20f3-9f63-4cd5-b86f-6cfc5c850835	CustomerIdentifier
CustomerIdentifier	SemanticTermToSemanticColumn	b794777d-2565-4c56-a897-c6ec3fa95825	CustomerIdentifier_public_customers_customer_id
CustomerIdentifier_public_customers_customer_id	SemanticColumnToDbColumn	6c023a12-9a7c-490d-9a90-05c46c56360f	customer_id
Company Name	BusinessTermToSemanticTerm	97058882-6730-41e1-b0c8-8af6508ca5f3	EntityName
EntityName	SemanticTermToSemanticColumn	7d7e46d6-8525-48cc-84a9-135caf30cf74	EntityName_public_customers_company_name
EntityName_public_customers_company_name	SemanticColumnToDbColumn	f731e8af-96a3-48ee-87d2-4126e0db7d27	company_name
Contact Name	BusinessTermToSemanticTerm	feb23c8b-c5aa-4cb4-a154-26aa6c75ac8c	ContactName
ContactName	SemanticTermToSemanticColumn	2060930e-1fa2-47ef-a7c8-efbf025597b5	ContactName_public_customers_contact_name
Contact Title	BusinessTermToSemanticTerm	f4748543-bbac-49d8-af6b-e38d1da87032	ContactTitle
ContactTitle	SemanticTermToSemanticColumn	3b16e1f1-34b7-49f6-90fa-f17ed86823f3	ContactTitle_public_customers_contact_title
ContactTitle_public_customers_contact_title	SemanticColumnToDbColumn	6ce7d55e-62ae-4ead-ab54-4d671744db74	contact_title
Employee ID	BusinessTermToSemanticTerm	b2ec498c-8b91-4a74-8a25-5dc6c2c3d5a8	EmployeeIdentifier
EmployeeIdentifier	SemanticTermToSemanticColumn	dd52f42b-87ea-4f85-83bb-bd27e1c409e9	EmployeeIdentifier_public_orders_employee_id
EmployeeIdentifier_public_orders_employee_id	SemanticColumnToDbColumn	d16aa18c-88ba-4bfa-897c-80351daef43c	employee_id`

	return ParseSemanticLineageData(semanticData)
}

// ParseSemanticLineageData parses tab-separated semantic data
func ParseSemanticLineageData(data string) *SemanticLineageChart {
	chart := &SemanticLineageChart{
		BusinessTerms:   []SemanticNode{},
		SemanticTerms:   []SemanticNode{},
		SemanticColumns: []SemanticNode{},
		DatabaseColumns: []SemanticNode{},
		Edges:           []SemanticEdge{},
		Viewport:        map[string]interface{}{"x": 0, "y": 0, "zoom": 1},
		Metadata:        map[string]interface{}{},
	}

	businessTerms := make(map[string]SemanticNode)
	semanticTerms := make(map[string]SemanticNode)
	semanticColumns := make(map[string]SemanticNode)
	databaseColumns := make(map[string]SemanticNode)

	businessTermIDs := make(map[string]uuid.UUID)
	semanticTermIDs := make(map[string]uuid.UUID)
	semanticColumnIDs := make(map[string]uuid.UUID)
	databaseColumnIDs := make(map[string]uuid.UUID)

	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}

		sourceNode := parts[0]
		relationshipType := parts[1]
		edgeID := parts[2]
		targetNode := parts[3]

		var edgeUUID uuid.UUID
		var err error
		if edgeUUID, err = uuid.Parse(edgeID); err != nil {
			edgeUUID = uuid.New()
		}

		switch relationshipType {
		case "BusinessTermToSemanticTerm":
			if _, exists := businessTerms[sourceNode]; !exists {
				nodeID := uuid.New()
				businessTerms[sourceNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      sourceNode,
					NodeType:      "business_term",
					Description:   fmt.Sprintf("Business term: %s", sourceNode),
					QualifiedPath: fmt.Sprintf("business.%s", strings.ReplaceAll(strings.ToLower(sourceNode), " ", "_")),
					Properties:    map[string]interface{}{"domain": "Business"},
				}
				businessTermIDs[sourceNode] = nodeID
			}

			if _, exists := semanticTerms[targetNode]; !exists {
				nodeID := uuid.New()
				semanticTerms[targetNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      targetNode,
					NodeType:      "semantic_term",
					Description:   fmt.Sprintf("Semantic term: %s", targetNode),
					QualifiedPath: fmt.Sprintf("semantic.%s", strings.ToLower(targetNode)),
					Properties:    map[string]interface{}{"category": "Semantic"},
				}
				semanticTermIDs[targetNode] = nodeID
			}

			chart.Edges = append(chart.Edges, SemanticEdge{
				ID:               edgeUUID,
				SourceID:         businessTermIDs[sourceNode],
				TargetID:         semanticTermIDs[targetNode],
				EdgeType:         "defines",
				RelationshipType: "defines",
				Properties:       map[string]interface{}{},
			})

		case "SemanticTermToSemanticColumn":
			if _, exists := semanticTerms[sourceNode]; !exists {
				nodeID := uuid.New()
				semanticTerms[sourceNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      sourceNode,
					NodeType:      "semantic_term",
					Description:   fmt.Sprintf("Semantic term: %s", sourceNode),
					QualifiedPath: fmt.Sprintf("semantic.%s", strings.ToLower(sourceNode)),
					Properties:    map[string]interface{}{"category": "Semantic"},
				}
				semanticTermIDs[sourceNode] = nodeID
			}

			if _, exists := semanticColumns[targetNode]; !exists {
				nodeID := uuid.New()
				semanticColumns[targetNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      targetNode,
					NodeType:      "semantic_column",
					Description:   fmt.Sprintf("Semantic column: %s", targetNode),
					QualifiedPath: fmt.Sprintf("semantic.columns.%s", strings.ToLower(targetNode)),
					Properties:    map[string]interface{}{"dataType": "Unknown"},
				}
				semanticColumnIDs[targetNode] = nodeID
			}

			chart.Edges = append(chart.Edges, SemanticEdge{
				ID:               edgeUUID,
				SourceID:         semanticTermIDs[sourceNode],
				TargetID:         semanticColumnIDs[targetNode],
				EdgeType:         "implements",
				RelationshipType: "implements",
				Properties:       map[string]interface{}{},
			})

		case "SemanticColumnToDbColumn":
			if _, exists := semanticColumns[sourceNode]; !exists {
				nodeID := uuid.New()
				semanticColumns[sourceNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      sourceNode,
					NodeType:      "semantic_column",
					Description:   fmt.Sprintf("Semantic column: %s", sourceNode),
					QualifiedPath: fmt.Sprintf("semantic.columns.%s", strings.ToLower(sourceNode)),
					Properties:    map[string]interface{}{"dataType": "Unknown"},
				}
				semanticColumnIDs[sourceNode] = nodeID
			}

			if _, exists := databaseColumns[targetNode]; !exists {
				nodeID := uuid.New()

				// Enhanced database column with qualified path
				var qualifiedPath string
				if strings.Contains(sourceNode, "_public_") {
					// Extract schema.table.column from semantic column name
					parts := strings.Split(sourceNode, "_public_")
					if len(parts) == 2 {
						tablePart := strings.Replace(parts[1], "_"+targetNode, "", 1)
						qualifiedPath = fmt.Sprintf("public.%s.%s", tablePart, targetNode)
					} else {
						qualifiedPath = fmt.Sprintf("public.unknown.%s", targetNode)
					}
				} else {
					qualifiedPath = fmt.Sprintf("unknown.unknown.%s", targetNode)
				}

				databaseColumns[targetNode] = SemanticNode{
					ID:            nodeID,
					NodeName:      targetNode,
					NodeType:      "database_column",
					Description:   fmt.Sprintf("Database column: %s", qualifiedPath),
					QualifiedPath: qualifiedPath,
					Properties: map[string]interface{}{
						"schema": strings.Split(qualifiedPath, ".")[0],
						"table":  strings.Split(qualifiedPath, ".")[1],
						"column": targetNode,
					},
				}
				databaseColumnIDs[targetNode] = nodeID
			}

			chart.Edges = append(chart.Edges, SemanticEdge{
				ID:               edgeUUID,
				SourceID:         semanticColumnIDs[sourceNode],
				TargetID:         databaseColumnIDs[targetNode],
				EdgeType:         "maps_to",
				RelationshipType: "maps_to",
				Properties:       map[string]interface{}{},
			})
		}
	}

	// Convert maps to slices
	for _, node := range businessTerms {
		chart.BusinessTerms = append(chart.BusinessTerms, node)
	}
	for _, node := range semanticTerms {
		chart.SemanticTerms = append(chart.SemanticTerms, node)
	}
	for _, node := range semanticColumns {
		chart.SemanticColumns = append(chart.SemanticColumns, node)
	}
	for _, node := range databaseColumns {
		chart.DatabaseColumns = append(chart.DatabaseColumns, node)
	}

	chart.Metadata["semanticEdgeCount"] = len(chart.Edges)
	chart.Metadata["totalNodes"] = len(chart.BusinessTerms) + len(chart.SemanticTerms) + len(chart.SemanticColumns) + len(chart.DatabaseColumns)
	chart.Metadata["generatedAt"] = time.Now().Format(time.RFC3339)
	chart.Metadata["chartType"] = "semantic_lineage"

	log.Printf("Parsed semantic data: %d business terms, %d semantic terms, %d semantic columns, %d database columns, %d edges",
		len(chart.BusinessTerms), len(chart.SemanticTerms), len(chart.SemanticColumns), len(chart.DatabaseColumns), len(chart.Edges))

	return chart
}

// createSampleTechnicalData generates sample technical data for testing
func createSampleTechnicalData(chart *TechnicalLineageChart) {
	customersID := uuid.New().String()
	ordersID := uuid.New().String()

	customerColumns := []map[string]interface{}{
		{
			"name":          "id",
			"type":          "integer",
			"isCore":        true,
			"nullable":      false,
			"schema":        "public",
			"table":         "customers",
			"qualifiedPath": "public.customers.id",
		},
		{
			"name":          "company_name",
			"type":          "varchar(100)",
			"isCore":        false,
			"nullable":      true,
			"schema":        "public",
			"table":         "customers",
			"qualifiedPath": "public.customers.company_name",
		},
		{
			"name":          "contact_name",
			"type":          "varchar(50)",
			"isCore":        false,
			"nullable":      true,
			"schema":        "public",
			"table":         "customers",
			"qualifiedPath": "public.customers.contact_name",
		},
		{
			"name":          "created_at",
			"type":          "timestamp",
			"isCore":        false,
			"nullable":      false,
			"schema":        "public",
			"table":         "customers",
			"qualifiedPath": "public.customers.created_at",
		},
	}

	orderColumns := []map[string]interface{}{
		{
			"name":          "id",
			"type":          "integer",
			"isCore":        true,
			"nullable":      false,
			"schema":        "public",
			"table":         "orders",
			"qualifiedPath": "public.orders.id",
		},
		{
			"name":          "customer_id",
			"type":          "integer",
			"isCore":        false,
			"nullable":      false,
			"schema":        "public",
			"table":         "orders",
			"qualifiedPath": "public.orders.customer_id",
		},
		{
			"name":          "order_date",
			"type":          "date",
			"isCore":        false,
			"nullable":      false,
			"schema":        "public",
			"table":         "orders",
			"qualifiedPath": "public.orders.order_date",
		},
		{
			"name":          "total_amount",
			"type":          "decimal(10,2)",
			"isCore":        false,
			"nullable":      true,
			"schema":        "public",
			"table":         "orders",
			"qualifiedPath": "public.orders.total_amount",
		},
	}

	chart.Nodes = []ReactFlowNode{
		{
			ID:       customersID,
			Type:     "table",
			Position: map[string]float64{"x": 0, "y": 0},
			Data: map[string]interface{}{
				"label":         "customers",
				"tableName":     "public.customers",
				"schemaName":    "public",
				"schema":        "public",
				"nodeType":      "table",
				"isCore":        false,
				"columns":       customerColumns,
				"qualifiedPath": "public.customers",
				"description":   "Table: public.customers",
				"columnCount":   len(customerColumns),
			},
		},
		{
			ID:       ordersID,
			Type:     "table",
			Position: map[string]float64{"x": 250, "y": 0},
			Data: map[string]interface{}{
				"label":         "orders",
				"tableName":     "public.orders",
				"schemaName":    "public",
				"schema":        "public",
				"nodeType":      "table",
				"isCore":        false,
				"columns":       orderColumns,
				"qualifiedPath": "public.orders",
				"description":   "Table: public.orders",
				"columnCount":   len(orderColumns),
			},
		},
	}

	chart.Edges = []ReactFlowEdge{
		{
			ID:     uuid.New().String(),
			Source: ordersID,
			Target: customersID,
			Type:   "smoothstep",
			Label:  "customer_id -> id",
			Data: map[string]interface{}{
				"relationshipType": "foreign_key",
				"sourceColumn":     "customer_id",
				"targetColumn":     "id",
			},
		},
	}

	chart.Metadata["databaseEdgeCount"] = 1
	chart.Metadata["totalNodes"] = 2
}
