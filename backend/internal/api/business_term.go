package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TreeNode represents a node in our final JSON structure.
type TreeNode struct {
	NodeID     string      `json:"node_id"`
	NodeTypeID string      `json:"node_type_id"`
	NodeName   string      `json:"node_name"`
	Children   []*TreeNode `json:"children,omitempty"`
}

// findChildren is the core recursive function that builds the node tree.
func findChildren(db *sql.DB, sourceNodeID, edgeTypeID, targetNodeTypeID string, visited map[string]bool) (*TreeNode, error) {
	// Cycle Detection
	if visited[sourceNodeID] {
		return nil, nil
	}
	visited[sourceNodeID] = true

	// Log visiting node for traversal visibility
	log.Printf("findChildren: visiting node %s (edgeType=%s, targetNodeType=%s)", sourceNodeID, edgeTypeID, targetNodeTypeID)

	// Fetch Node Details
	var node TreeNode
	queryNode := `SELECT id, node_type_id, node_name FROM catalog_node WHERE id = $1`
	err := db.QueryRow(queryNode, sourceNodeID).Scan(&node.NodeID, &node.NodeTypeID, &node.NodeName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Warning: Node with ID %s not found.", sourceNodeID)
			return nil, nil
		}
		log.Printf("Error querying for node %s: %v", sourceNodeID, err)
		return nil, err
	}

	// Base Case
	if node.NodeTypeID == targetNodeTypeID {
		return &node, nil
	}

	// Recursive Step
	queryEdges := `SELECT target_node_id FROM catalog_edge WHERE source_node_id = $1 AND edge_type_id = $2`
	rows, err := db.Query(queryEdges, sourceNodeID, edgeTypeID)
	if err != nil {
		log.Printf("Error querying for edges from source %s: %v", sourceNodeID, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var targetNodeID string
		if err := rows.Scan(&targetNodeID); err != nil {
			log.Printf("Error scanning target node ID: %v", err)
			continue
		}

		childNode, err := findChildren(db, targetNodeID, edgeTypeID, targetNodeTypeID, visited)
		if err != nil {
			return nil, err
		}

		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	return &node, nil
}

// findNodeByID removed — traversal helpers consolidated in GetBusinessTerm to keep
// the file focused and avoid an unused symbol.

// findPath returns the path from the provided root to the node with targetID as
// a slice of TreeNode pointers. It returns the path and true if found.
func findPath(node *TreeNode, targetID string, acc []*TreeNode) ([]*TreeNode, bool) {
	if node == nil {
		return nil, false
	}
	acc = append(acc, node)
	if node.NodeID == targetID {
		return acc, true
	}
	for _, c := range node.Children {
		if p, ok := findPath(c, targetID, acc); ok {
			return p, true
		}
	}
	return nil, false
}

func GetBusinessTerm(db *sql.DB, columnID, edgeTypeID, targetNodeID string) (string, error) {
	// First, get the node_type_id of the target node
	var targetNodeTypeID string
	err := db.QueryRow(`SELECT node_type_id FROM catalog_node WHERE id = $1`, targetNodeID).Scan(&targetNodeTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("GetBusinessTerm: target node id %s not found", targetNodeID)
			// Not an internal error — no business term can be resolved if the
			// target node doesn't exist. Return empty term and no error so
			// caller can respond with a JSON payload rather than a 500.
			return "", nil
		}
		return "", err
	}

	visited := make(map[string]bool)
	root, err := findChildren(db, columnID, edgeTypeID, targetNodeTypeID, visited)
	if err != nil {
		return "", err
	}
	if root == nil {
		return "", nil
	}

	// Log the full traversal path (if any) from the column root to the
	// requested target node so developers can inspect how the graph was
	// explored.
	if path, ok := findPath(root, targetNodeID, nil); ok {
		// Build a readable path string: Name (id) -> Name (id) ...
		parts := make([]string, 0, len(path))
		for _, n := range path {
			parts = append(parts, n.NodeName+" ("+n.NodeID+")")
		}
		log.Printf("GetBusinessTerm: traversal path: %s", strings.Join(parts, " -> "))
		return path[len(path)-1].NodeName, nil
	}

	// If we didn't find the target in the traversal, log the root tree for
	// debugging so developers can inspect available children.
	b, _ := json.MarshalIndent(root, "", "  ")
	log.Printf("GetBusinessTerm: traversal did not find target %s; root tree:\n%s", targetNodeID, string(b))
	return "", nil
}

// BusinessTermSuggestion represents a suggested business term with metadata
type BusinessTermSuggestion struct {
	BusinessTerm   string   `json:"business_term"`
	Description    string   `json:"description"`
	Categories     []string `json:"categories"`
	Confidence     float64  `json:"confidence"`
	SemanticTerm   string   `json:"semantic_term"`
	DatabaseColumn string   `json:"database_column"`
}

// GenerateBusinessTermSuggestionsRequest represents the request payload
type GenerateBusinessTermSuggestionsRequest struct {
	SemanticTerms   []string `json:"semantic_terms"`
	DatabaseColumns []string `json:"database_columns"`
	Limit           int      `json:"limit,omitempty"`
}

// generateBusinessTermSuggestions generates AI-powered business term suggestions
func generateBusinessTermSuggestions(semanticTerms []string, databaseColumns []string, limit int) []BusinessTermSuggestion {
	if limit <= 0 {
		limit = 5
	}

	suggestions := []BusinessTermSuggestion{}

	// Create mappings between semantic terms and database columns
	termColumnMap := make(map[string][]string)
	for i, semanticTerm := range semanticTerms {
		if i < len(databaseColumns) {
			termColumnMap[semanticTerm] = append(termColumnMap[semanticTerm], databaseColumns[i])
		}
	}

	// Generate suggestions for each semantic term
	for semanticTerm, columns := range termColumnMap {
		// Use the first column for context
		column := columns[0]

		// Simple rule-based suggestions (in a real implementation, this would use AI/ML)
		suggestion := generateSuggestionForTerm(semanticTerm, column)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	// Limit the results
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions
}

// generateSuggestionForTerm creates a business term suggestion for a semantic term and column
func generateSuggestionForTerm(semanticTerm, databaseColumn string) *BusinessTermSuggestion {
	semanticTerm = strings.ToUpper(strings.TrimSpace(semanticTerm))
	databaseColumn = strings.ToUpper(strings.TrimSpace(databaseColumn))

	// Simple pattern matching for common business terms
	// In a real implementation, this would use machine learning models
	switch {
	case strings.Contains(semanticTerm, "CUSTOMER") || strings.Contains(databaseColumn, "CUST"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Customer Information",
			Description:    "Core customer data including identification and contact details",
			Categories:     []string{"Customer Management", "CRM", "Personal Data"},
			Confidence:     0.85,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	case strings.Contains(semanticTerm, "ACCOUNT") || strings.Contains(databaseColumn, "ACCT"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Account Details",
			Description:    "Financial account information and identifiers",
			Categories:     []string{"Financial Services", "Banking", "Account Management"},
			Confidence:     0.82,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	case strings.Contains(semanticTerm, "TRANSACTION") || strings.Contains(databaseColumn, "TXN"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Transaction Records",
			Description:    "Financial transaction data and processing information",
			Categories:     []string{"Financial Services", "Transaction Processing", "Audit Trail"},
			Confidence:     0.88,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	case strings.Contains(semanticTerm, "AMOUNT") || strings.Contains(databaseColumn, "AMT"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Monetary Values",
			Description:    "Financial amounts, balances, and monetary calculations",
			Categories:     []string{"Financial Services", "Accounting", "Monetary Data"},
			Confidence:     0.80,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	case strings.Contains(semanticTerm, "DATE") || strings.Contains(databaseColumn, "DATE"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Temporal Information",
			Description:    "Date and time related data for tracking and analysis",
			Categories:     []string{"Time Series", "Analytics", "Reporting"},
			Confidence:     0.75,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	case strings.Contains(semanticTerm, "NAME") || strings.Contains(databaseColumn, "NAME"):
		return &BusinessTermSuggestion{
			BusinessTerm:   "Entity Names",
			Description:    "Names and identifiers for people, organizations, or entities",
			Categories:     []string{"Identity", "Reference Data", "Entity Management"},
			Confidence:     0.78,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	default:
		// Generic fallback suggestion
		tc := cases.Title(language.Und)
		return &BusinessTermSuggestion{
			BusinessTerm:   tc.String(strings.ToLower(semanticTerm)),
			Description:    fmt.Sprintf("Business context for %s data field", semanticTerm),
			Categories:     []string{"General", "Reference Data"},
			Confidence:     0.60,
			SemanticTerm:   semanticTerm,
			DatabaseColumn: databaseColumn,
		}
	}
}

func (s *Server) generateBusinessTermSuggestions(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	log.Printf("generateBusinessTermSuggestions handler called: tenant_id=%s datasource_id=%s remote=%s",
		tenantID, datasourceID, r.RemoteAddr)

	// Parse request body
	var req GenerateBusinessTermSuggestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("generateBusinessTermSuggestions: failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.SemanticTerms) == 0 {
		log.Printf("generateBusinessTermSuggestions: no semantic terms provided")
		http.Error(w, "At least one semantic term is required", http.StatusBadRequest)
		return
	}

	if len(req.DatabaseColumns) == 0 {
		log.Printf("generateBusinessTermSuggestions: no database columns provided")
		http.Error(w, "At least one database column is required", http.StatusBadRequest)
		return
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Generate suggestions
	suggestions := generateBusinessTermSuggestions(req.SemanticTerms, req.DatabaseColumns, req.Limit)

	// Log the number of suggestions generated
	log.Printf("generateBusinessTermSuggestions: generated %d suggestions for %d semantic terms",
		len(suggestions), len(req.SemanticTerms))

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}
