package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/dynamic"
)

// DynamicParameterHandler handles dynamic parameter operations
type DynamicParameterHandler struct {
	db *sql.DB
}

// NewDynamicParameterHandler creates a new dynamic parameter handler
func NewDynamicParameterHandler(db *sql.DB) *DynamicParameterHandler {
	return &DynamicParameterHandler{db: db}
}

// GetAvailableValues returns available values for a dynamic parameter
func (h *DynamicParameterHandler) GetAvailableValues(w http.ResponseWriter, r *http.Request) {
	paramType := chi.URLParam(r, "type")
	paramName := chi.URLParam(r, "name")

	if paramType == "" || paramName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Parameter type and name are required",
		})
		return
	}

	values, err := h.fetchAvailableValues(paramType, paramName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch available values",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"parameter": paramName,
		"type":      paramType,
		"values":    values,
	})
}

// fetchAvailableValues fetches available values based on parameter type and name
func (h *DynamicParameterHandler) fetchAvailableValues(paramType, paramName string) ([]string, error) {
	var query string
	var args []interface{}

	switch paramType {
	case "dimension":
		switch paramName {
		case "city":
			query = "SELECT DISTINCT city FROM clickstream WHERE city IS NOT NULL ORDER BY city"
		case "region":
			query = "SELECT DISTINCT region FROM clickstream WHERE region IS NOT NULL ORDER BY region"
		case "country":
			query = "SELECT DISTINCT country FROM clickstream WHERE country IS NOT NULL ORDER BY country"
		case "device_type":
			query = "SELECT DISTINCT device_type FROM clickstream WHERE device_type IS NOT NULL ORDER BY device_type"
		case "status":
			query = "SELECT DISTINCT status FROM orders WHERE status IS NOT NULL ORDER BY status"
		case "category":
			query = "SELECT DISTINCT category FROM products WHERE category IS NOT NULL ORDER BY category"
		default:
			return nil, fmt.Errorf("unknown dimension parameter: %s", paramName)
		}

	case "time_range":
		switch paramName {
		case "period":
			return []string{"1d", "7d", "30d", "90d", "1y"}, nil
		case "granularity":
			return []string{"hour", "day", "week", "month", "quarter", "year"}, nil
		default:
			return nil, fmt.Errorf("unknown time_range parameter: %s", paramName)
		}

	case "filter":
		switch paramName {
		case "active_only":
			return []string{"true", "false"}, nil
		case "premium_only":
			return []string{"true", "false"}, nil
		default:
			return nil, fmt.Errorf("unknown filter parameter: %s", paramName)
		}

	default:
		return nil, fmt.Errorf("unknown parameter type: %s", paramType)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

// GetParameterSchema returns the schema for dynamic parameters
func (h *DynamicParameterHandler) GetParameterSchema(w http.ResponseWriter, r *http.Request) {
	schema := map[string]interface{}{
		"dimensions": map[string]interface{}{
			"city": map[string]interface{}{
				"type":        "string",
				"description": "Geographic city for filtering",
				"required":    false,
				"source":      "clickstream.city",
			},
			"region": map[string]interface{}{
				"type":        "string",
				"description": "Geographic region for filtering",
				"required":    false,
				"source":      "clickstream.region",
			},
			"country": map[string]interface{}{
				"type":        "string",
				"description": "Country for filtering",
				"required":    false,
				"source":      "clickstream.country",
			},
			"device_type": map[string]interface{}{
				"type":        "string",
				"description": "Device type for filtering",
				"required":    false,
				"source":      "clickstream.device_type",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Order status for filtering",
				"required":    false,
				"source":      "orders.status",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Product category for filtering",
				"required":    false,
				"source":      "products.category",
			},
		},
		"time_ranges": map[string]interface{}{
			"period": map[string]interface{}{
				"type":        "string",
				"description": "Time period for analysis",
				"required":    false,
				"options":     []string{"1d", "7d", "30d", "90d", "1y"},
				"default":     "30d",
			},
			"granularity": map[string]interface{}{
				"type":        "string",
				"description": "Time granularity for grouping",
				"required":    false,
				"options":     []string{"hour", "day", "week", "month", "quarter", "year"},
				"default":     "day",
			},
		},
		"filters": map[string]interface{}{
			"active_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter for active records only",
				"required":    false,
				"default":     true,
			},
			"premium_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter for premium users only",
				"required":    false,
				"default":     false,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// DynamicMeasureHandler handles dynamic measure operations
type DynamicMeasureHandler struct {
	db *sql.DB
}

// NewDynamicMeasureHandler creates a new dynamic measure handler
func NewDynamicMeasureHandler(db *sql.DB) *DynamicMeasureHandler {
	return &DynamicMeasureHandler{db: db}
}

// GenerateDynamicMeasures generates dynamic measures based on database enums
func (h *DynamicMeasureHandler) GenerateDynamicMeasures(w http.ResponseWriter, r *http.Request) {
	sourceTable := r.URL.Query().Get("table")
	sourceColumn := r.URL.Query().Get("column")

	if sourceTable == "" || sourceColumn == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "table and column query parameters are required",
		})
		return
	}

	measures, err := h.generateMeasuresFromEnum(sourceTable, sourceColumn)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to generate dynamic measures",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"source":   fmt.Sprintf("%s.%s", sourceTable, sourceColumn),
		"measures": measures,
	})
}

// generateMeasuresFromEnum generates measures based on distinct values in a column
func (h *DynamicMeasureHandler) generateMeasuresFromEnum(table, column string) ([]dynamic.DynamicMeasure, error) {
	// Get distinct values from the source column
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s", column, table, column, column)
	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	// Generate dynamic measures for each value
	var measures []dynamic.DynamicMeasure
	for _, value := range values {
		measure := dynamic.DynamicMeasure{
			Name: fmt.Sprintf("total_%s_%s", strings.ToLower(value), strings.ToLower(column)),
			Type: "count",
			SQL:  fmt.Sprintf("CASE WHEN %s = '%s' THEN 1 ELSE 0 END", column, value),
			Meta: map[string]interface{}{
				"source_table":  table,
				"source_column": column,
				"filter_value":  value,
				"generated_at":  time.Now().Format(time.RFC3339),
			},
		}
		measures = append(measures, measure)
	}

	return measures, nil
}

// GetDynamicMeasureCatalog returns the catalog of available dynamic measures
func (h *DynamicMeasureHandler) GetDynamicMeasureCatalog(w http.ResponseWriter, r *http.Request) {
	// This would typically query the catalog_node table
	// For now, return a sample catalog
	catalog := []map[string]interface{}{
		{
			"id":            "status_measures",
			"name":          "Order Status Measures",
			"source_table":  "orders",
			"source_column": "status",
			"measures": []string{
				"total_processing_orders",
				"total_shipped_orders",
				"total_completed_orders",
			},
			"last_updated": time.Now().Format(time.RFC3339),
			"golden_path":  true,
		},
		{
			"id":            "device_measures",
			"name":          "Device Type Measures",
			"source_table":  "clickstream",
			"source_column": "device_type",
			"measures": []string{
				"total_mobile_device_type",
				"total_desktop_device_type",
				"total_tablet_device_type",
			},
			"last_updated": time.Now().Format(time.RFC3339),
			"golden_path":  true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"catalog": catalog,
	})
}

// ValidateDynamicMeasure validates a dynamic measure definition
func (h *DynamicMeasureHandler) ValidateDynamicMeasure(w http.ResponseWriter, r *http.Request) {
	var measure dynamic.DynamicMeasure
	if err := json.NewDecoder(r.Body).Decode(&measure); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid measure format",
			"details": err.Error(),
		})
		return
	}

	// Basic validation
	if measure.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Measure name is required",
		})
		return
	}

	if measure.SQL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Measure SQL is required",
		})
		return
	}

	// Check for SQL injection patterns (basic check)
	dangerousPatterns := []string{"DROP", "DELETE", "UPDATE", "INSERT", "EXEC", "EXECUTE"}
	sqlUpper := strings.ToUpper(measure.SQL)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(sqlUpper, pattern) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Measure SQL contains potentially dangerous patterns",
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":        true,
		"measure":      measure,
		"validated_at": time.Now().Format(time.RFC3339),
	})
}

// DynamicDimensionHandler handles dynamic dimension operations
type DynamicDimensionHandler struct {
	db *sql.DB
}

// NewDynamicDimensionHandler creates a new dynamic dimension handler
func NewDynamicDimensionHandler(db *sql.DB) *DynamicDimensionHandler {
	return &DynamicDimensionHandler{db: db}
}

// GetDynamicDimensions returns available dynamic dimensions
func (h *DynamicDimensionHandler) GetDynamicDimensions(w http.ResponseWriter, r *http.Request) {
	dimensions := []map[string]interface{}{
		{
			"name":        "advisor_id",
			"type":        "number",
			"description": "Financial advisor identifier",
			"source":      "transactions.advisor_id",
			"cardinality": "high",
			"usage":       "segmentation",
		},
		{
			"name":        "fund_type",
			"type":        "string",
			"description": "Type of investment fund",
			"source":      "transactions.fund_type",
			"cardinality": "low",
			"usage":       "filtering",
		},
		{
			"name":        "client_segment",
			"type":        "string",
			"description": "Client segmentation category",
			"source":      "clients.segment",
			"cardinality": "low",
			"usage":       "filtering",
		},
		{
			"name":        "risk_profile",
			"type":        "string",
			"description": "Client risk profile",
			"source":      "clients.risk_profile",
			"cardinality": "low",
			"usage":       "filtering",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dimensions": dimensions,
	})
}

// GetDimensionValues returns available values for a dynamic dimension
func (h *DynamicDimensionHandler) GetDimensionValues(w http.ResponseWriter, r *http.Request) {
	dimensionName := chi.URLParam(r, "dimension")

	if dimensionName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Dimension name is required",
		})
		return
	}

	values, err := h.fetchDimensionValues(dimensionName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch dimension values",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dimension": dimensionName,
		"values":    values,
	})
}

// fetchDimensionValues fetches available values for a dynamic dimension
func (h *DynamicDimensionHandler) fetchDimensionValues(dimensionName string) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	switch dimensionName {
	case "advisor_id":
		query = "SELECT DISTINCT advisor_id as value, COUNT(*) as count FROM transactions WHERE advisor_id IS NOT NULL GROUP BY advisor_id ORDER BY advisor_id"
	case "fund_type":
		query = "SELECT DISTINCT fund_type as value, COUNT(*) as count FROM transactions WHERE fund_type IS NOT NULL GROUP BY fund_type ORDER BY count DESC"
	case "client_segment":
		query = "SELECT DISTINCT segment as value, COUNT(*) as count FROM clients WHERE segment IS NOT NULL GROUP BY segment ORDER BY count DESC"
	case "risk_profile":
		query = "SELECT DISTINCT risk_profile as value, COUNT(*) as count FROM clients WHERE risk_profile IS NOT NULL GROUP BY risk_profile ORDER BY count DESC"
	default:
		return nil, fmt.Errorf("unknown dimension: %s", dimensionName)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []map[string]interface{}
	for rows.Next() {
		var value string
		var count int
		if err := rows.Scan(&value, &count); err != nil {
			return nil, err
		}
		values = append(values, map[string]interface{}{
			"value": value,
			"count": count,
			"label": fmt.Sprintf("%s (%d)", value, count),
		})
	}

	return values, nil
}

// ScopedFilterHandler handles scoped filter operations
type ScopedFilterHandler struct {
	db *sql.DB
}

// NewScopedFilterHandler creates a new scoped filter handler
func NewScopedFilterHandler(db *sql.DB) *ScopedFilterHandler {
	return &ScopedFilterHandler{db: db}
}

// GetScopedFilters returns available scoped filters
func (h *ScopedFilterHandler) GetScopedFilters(w http.ResponseWriter, r *http.Request) {
	filters := []map[string]interface{}{
		{
			"name":        "high_value_clients",
			"description": "Clients with AUM > $1M",
			"type":        "boolean",
			"sql":         "aum > 1000000",
			"source":      "clients.aum",
			"category":    "value",
		},
		{
			"name":        "active_traders",
			"description": "Clients with transactions in last 30 days",
			"type":        "boolean",
			"sql":         "last_transaction_date > CURRENT_DATE - INTERVAL '30 days'",
			"source":      "clients.last_transaction_date",
			"category":    "activity",
		},
		{
			"name":        "premium_accounts",
			"description": "Premium account tier clients",
			"type":        "boolean",
			"sql":         "account_tier = 'premium'",
			"source":      "clients.account_tier",
			"category":    "tier",
		},
		{
			"name":        "new_clients",
			"description": "Clients acquired in last 90 days",
			"type":        "boolean",
			"sql":         "acquisition_date > CURRENT_DATE - INTERVAL '90 days'",
			"source":      "clients.acquisition_date",
			"category":    "recency",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"filters": filters,
	})
}

// ApplyScopedFilter applies a scoped filter to a query
func (h *ScopedFilterHandler) ApplyScopedFilter(w http.ResponseWriter, r *http.Request) {
	var request struct {
		BaseQuery  string                 `json:"base_query" binding:"required"`
		FilterName string                 `json:"filter_name" binding:"required"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	filterSQL, err := h.getFilterSQL(request.FilterName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Unknown filter",
			"details": err.Error(),
		})
		return
	}

	// Apply parameters to filter SQL if provided
	for key, value := range request.Parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		filterSQL = strings.ReplaceAll(filterSQL, placeholder, fmt.Sprintf("%v", value))
	}

	modifiedQuery := fmt.Sprintf("%s WHERE %s", request.BaseQuery, filterSQL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original_query": request.BaseQuery,
		"filter_applied": request.FilterName,
		"filter_sql":     filterSQL,
		"modified_query": modifiedQuery,
	})
}

// getFilterSQL returns the SQL for a scoped filter
func (h *ScopedFilterHandler) getFilterSQL(filterName string) (string, error) {
	filterMap := map[string]string{
		"high_value_clients": "aum > 1000000",
		"active_traders":     "last_transaction_date > CURRENT_DATE - INTERVAL '30 days'",
		"premium_accounts":   "account_tier = 'premium'",
		"new_clients":        "acquisition_date > CURRENT_DATE - INTERVAL '90 days'",
	}

	filterSQL, exists := filterMap[filterName]
	if !exists {
		return "", fmt.Errorf("filter '%s' not found", filterName)
	}

	return filterSQL, nil
}

// CohortFilterHandler handles scoped cohort filter operations
type CohortFilterHandler struct {
	db *sql.DB
}

// NewCohortFilterHandler creates a new cohort filter handler
func NewCohortFilterHandler(db *sql.DB) *CohortFilterHandler {
	return &CohortFilterHandler{db: db}
}

// GetCohorts returns available behavioral cohorts
func (h *CohortFilterHandler) GetCohorts(w http.ResponseWriter, r *http.Request) {
	cohorts := []map[string]interface{}{
		{
			"name":           "high_tenure_clients",
			"description":    "Clients with tenure > 5 years",
			"type":           "behavioral",
			"sql":            "DATEDIFF('year', acquisition_date, CURRENT_DATE) > 5",
			"source":         "clients.acquisition_date",
			"estimated_size": "medium",
		},
		{
			"name":           "high_risk_clients",
			"description":    "Clients with risk profile = high",
			"type":           "behavioral",
			"sql":            "risk_profile = 'high'",
			"source":         "clients.risk_profile",
			"estimated_size": "small",
		},
		{
			"name":           "premium_fund_holders",
			"description":    "Clients holding premium fund types",
			"type":           "domain",
			"sql":            "fund_type IN ('premium', 'institutional', 'private_equity')",
			"source":         "accounts.fund_type",
			"estimated_size": "medium",
		},
		{
			"name":           "active_traders",
			"description":    "Clients with > 10 transactions in last 30 days",
			"type":           "behavioral",
			"sql":            "transaction_count_30d > 10",
			"source":         "client_metrics.transaction_count_30d",
			"estimated_size": "small",
		},
		{
			"name":           "large_accounts",
			"description":    "Accounts with AUM > $5M",
			"type":           "domain",
			"sql":            "aum > 5000000",
			"source":         "accounts.aum",
			"estimated_size": "small",
		},
		{
			"name":           "new_clients_q4",
			"description":    "Clients acquired in Q4 2024",
			"type":           "temporal",
			"sql":            "acquisition_date >= '2024-10-01' AND acquisition_date < '2025-01-01'",
			"source":         "clients.acquisition_date",
			"estimated_size": "medium",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cohorts": cohorts,
	})
}

// GetCohortValues returns available values for a cohort filter
func (h *CohortFilterHandler) GetCohortValues(w http.ResponseWriter, r *http.Request) {
	cohortName := chi.URLParam(r, "cohort")

	if cohortName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Cohort name is required",
		})
		return
	}

	values, err := h.fetchCohortValues(cohortName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch cohort values",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cohort": cohortName,
		"values": values,
	})
}

// fetchCohortValues fetches available values for a cohort filter
func (h *CohortFilterHandler) fetchCohortValues(cohortName string) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	switch cohortName {
	case "high_tenure_clients":
		query = "SELECT DISTINCT client_id as value, DATEDIFF('year', acquisition_date, CURRENT_DATE) as tenure_years FROM clients WHERE DATEDIFF('year', acquisition_date, CURRENT_DATE) > 5 ORDER BY tenure_years DESC LIMIT 100"
	case "high_risk_clients":
		query = "SELECT DISTINCT client_id as value, risk_profile FROM clients WHERE risk_profile = 'high' ORDER BY client_id LIMIT 100"
	case "premium_fund_holders":
		query = "SELECT DISTINCT client_id as value, fund_type FROM accounts WHERE fund_type IN ('premium', 'institutional', 'private_equity') ORDER BY fund_type LIMIT 100"
	case "active_traders":
		query = "SELECT DISTINCT client_id as value, transaction_count_30d FROM client_metrics WHERE transaction_count_30d > 10 ORDER BY transaction_count_30d DESC LIMIT 100"
	case "large_accounts":
		query = "SELECT DISTINCT account_id as value, aum FROM accounts WHERE aum > 5000000 ORDER BY aum DESC LIMIT 100"
	case "new_clients_q4":
		query = "SELECT DISTINCT client_id as value, acquisition_date FROM clients WHERE acquisition_date >= '2024-10-01' AND acquisition_date < '2025-01-01' ORDER BY acquisition_date DESC LIMIT 100"
	default:
		return nil, fmt.Errorf("unknown cohort: %s", cohortName)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []map[string]interface{}
	for rows.Next() {
		var value string
		var metadata interface{}
		if err := rows.Scan(&value, &metadata); err != nil {
			return nil, err
		}
		values = append(values, map[string]interface{}{
			"value":    value,
			"metadata": metadata,
			"label":    fmt.Sprintf("%s (%v)", value, metadata),
		})
	}

	return values, nil
}

// ApplyCohortFilter applies a cohort filter to a query
func (h *CohortFilterHandler) ApplyCohortFilter(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CohortName string                 `json:"cohort_name" binding:"required"`
		BaseQuery  string                 `json:"base_query" binding:"required"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	cohortSQL, err := h.getCohortSQL(request.CohortName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Unknown cohort",
			"details": err.Error(),
		})
		return
	}

	// Apply parameters to cohort SQL if provided
	for key, value := range request.Parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		cohortSQL = strings.ReplaceAll(cohortSQL, placeholder, fmt.Sprintf("%v", value))
	}

	modifiedQuery := fmt.Sprintf("%s WHERE %s", request.BaseQuery, cohortSQL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original_query": request.BaseQuery,
		"cohort_applied": request.CohortName,
		"cohort_sql":     cohortSQL,
		"modified_query": modifiedQuery,
	})
}

// getCohortSQL returns the SQL for a cohort filter
func (h *CohortFilterHandler) getCohortSQL(cohortName string) (string, error) {
	cohortMap := map[string]string{
		"high_tenure_clients":  "DATEDIFF('year', acquisition_date, CURRENT_DATE) > 5",
		"high_risk_clients":    "risk_profile = 'high'",
		"premium_fund_holders": "fund_type IN ('premium', 'institutional', 'private_equity')",
		"active_traders":       "transaction_count_30d > 10",
		"large_accounts":       "aum > 5000000",
		"new_clients_q4":       "acquisition_date >= '2024-10-01' AND acquisition_date < '2025-01-01'",
	}

	cohortSQL, exists := cohortMap[cohortName]
	if !exists {
		return "", fmt.Errorf("cohort '%s' not found", cohortName)
	}

	return cohortSQL, nil
}

// LineageVisualizationHandler handles lineage-aware operations for visualization
type LineageVisualizationHandler struct {
	db *sql.DB
}

// NewLineageVisualizationHandler creates a new lineage visualization handler
func NewLineageVisualizationHandler(db *sql.DB) *LineageVisualizationHandler {
	return &LineageVisualizationHandler{db: db}
}

// GetLineage returns lineage information for a metric or dimension
func (h *LineageVisualizationHandler) GetLineage(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	if nodeID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Node ID is required",
		})
		return
	}

	lineage, err := h.fetchLineage(nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch lineage",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lineage)
}

// fetchLineage fetches lineage information for a node
func (h *LineageVisualizationHandler) fetchLineage(nodeID string) (map[string]interface{}, error) {
	// Query the catalog for the node
	var nodeType, name string
	var description, schemaDef sql.NullString
	err := h.db.QueryRow(`
		SELECT node_type, name, description, schema_def
		FROM public.catalog_node
		WHERE node_id = $1
	`, nodeID).Scan(&nodeType, &name, &description, &schemaDef)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// Parse schema definition to extract lineage information
	var schemaData map[string]interface{}
	if !schemaDef.Valid || schemaDef.String == "" {
		return nil, fmt.Errorf("no schema definition available for node %s", nodeID)
	}
	if err := json.Unmarshal([]byte(schemaDef.String), &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Extract source tables from lineage
	sourceTables := []string{}
	if lineage, ok := schemaData["lineage"].(map[string]interface{}); ok {
		if sources, ok := lineage["source_tables"].([]interface{}); ok {
			for _, source := range sources {
				if sourceStr, ok := source.(string); ok {
					sourceTables = append(sourceTables, sourceStr)
				}
			}
		}
	}

	// Find downstream consumers (other nodes that reference this node)
	downstreamConsumers := []string{}
	rows, err := h.db.Query(`
		SELECT node_id, name
		FROM public.catalog_node
		WHERE schema_def::text LIKE $1
	`, "%"+nodeID+"%")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var consumerID, consumerName string
			rows.Scan(&consumerID, &consumerName)
			if consumerID != nodeID {
				downstreamConsumers = append(downstreamConsumers, consumerName)
			}
		}
	}

	// Build lineage graph data
	lineageData := map[string]interface{}{
		"node_id":              nodeID,
		"node_type":            nodeType,
		"name":                 name,
		"description":          description,
		"source_tables":        sourceTables,
		"downstream_consumers": downstreamConsumers,
		"upstream_transformations": []string{
			"Data validation",
			"Type conversion",
			"Aggregation",
			"Join operations",
		},
		"data_quality_checks": []string{
			"Completeness check",
			"Accuracy validation",
			"Freshness monitoring",
			"Consistency verification",
		},
	}

	return lineageData, nil
}

// GetLineageGraph returns graph data for visualization
func (h *LineageVisualizationHandler) GetLineageGraph(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	if nodeID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Node ID is required",
		})
		return
	}

	lineage, err := h.fetchLineage(nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch lineage",
			"details": err.Error(),
		})
		return
	}

	// Convert to Cytoscape format
	elements := []map[string]interface{}{}

	// Add source tables as nodes
	for _, source := range lineage["source_tables"].([]string) {
		elements = append(elements, map[string]interface{}{
			"data": map[string]interface{}{
				"id":    source,
				"label": source,
				"type":  "source_table",
			},
		})
	}

	// Add the main node
	elements = append(elements, map[string]interface{}{
		"data": map[string]interface{}{
			"id":    lineage["node_id"],
			"label": lineage["name"],
			"type":  lineage["node_type"],
		},
	})

	// Add downstream consumers
	for _, consumer := range lineage["downstream_consumers"].([]string) {
		elements = append(elements, map[string]interface{}{
			"data": map[string]interface{}{
				"id":    strings.ReplaceAll(consumer, " ", "_"),
				"label": consumer,
				"type":  "consumer",
			},
		})
	}

	// Add edges from sources to main node
	for _, source := range lineage["source_tables"].([]string) {
		elements = append(elements, map[string]interface{}{
			"data": map[string]interface{}{
				"source": source,
				"target": lineage["node_id"],
				"type":   "transformation",
			},
		})
	}

	// Add edges from main node to consumers
	for _, consumer := range lineage["downstream_consumers"].([]string) {
		elements = append(elements, map[string]interface{}{
			"data": map[string]interface{}{
				"source": lineage["node_id"],
				"target": strings.ReplaceAll(consumer, " ", "_"),
				"type":   "consumption",
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"elements": elements,
		"lineage":  lineage,
	})
}

// StewardApprovalHandler handles steward approval workflow operations
type StewardApprovalHandler struct {
	db *sql.DB
}

// NewStewardApprovalHandler creates a new steward approval handler
func NewStewardApprovalHandler(db *sql.DB) *StewardApprovalHandler {
	return &StewardApprovalHandler{db: db}
}

// GetPendingApprovals returns metrics pending steward approval
func (h *StewardApprovalHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	stewardUser := r.URL.Query().Get("steward")
	if stewardUser == "" {
		stewardUser = "system"
	}

	rows, err := h.db.Query(`
		SELECT node_id, node_type, name, description, schema_def, created_at, review_status
		FROM public.catalog_node
		WHERE review_status IN ('draft', 'pending_review')
		AND (steward_group = $1 OR steward_group IS NULL)
		ORDER BY created_at DESC
	`, stewardUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch pending approvals",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var approvals []map[string]interface{}
	for rows.Next() {
		var nodeID, nodeType, name, reviewStatus string
		var createdAt time.Time
		var description, schemaDef sql.NullString
		rows.Scan(&nodeID, &nodeType, &name, &description, &schemaDef, &createdAt, &reviewStatus)

		desc := ""
		if description.Valid {
			desc = description.String
		}
		schema := ""
		if schemaDef.Valid {
			schema = schemaDef.String
		}
		approvals = append(approvals, map[string]interface{}{
			"node_id":       nodeID,
			"node_type":     nodeType,
			"name":          name,
			"description":   desc,
			"schema_def":    schema,
			"created_at":    createdAt,
			"review_status": reviewStatus,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_approvals": approvals,
		"steward":           stewardUser,
	})
}

// ApproveMetric approves a metric for production use
func (h *StewardApprovalHandler) ApproveMetric(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	var request struct {
		StewardUser string `json:"steward_user" binding:"required"`
		Comment     string `json:"comment,omitempty"`
		GoldenPath  bool   `json:"golden_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update the metric status
	_, err := h.db.Exec(`
		UPDATE public.catalog_node
		SET review_status = 'approved',
		    golden_path = $1,
		    updated_at = $2
		WHERE node_id = $3
	`, request.GoldenPath, time.Now().UTC(), nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to approve metric",
			"details": err.Error(),
		})
		return
	}

	// Add approval comment
	if request.Comment != "" {
		h.addReviewComment(nodeID, request.StewardUser, request.Comment, "approve")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Metric approved successfully",
		"node_id":     nodeID,
		"golden_path": request.GoldenPath,
	})
}

// RejectMetric rejects a metric with feedback
func (h *StewardApprovalHandler) RejectMetric(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	var request struct {
		StewardUser string `json:"steward_user" binding:"required"`
		Comment     string `json:"comment" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update the metric status
	_, err := h.db.Exec(`
		UPDATE public.catalog_node
		SET review_status = 'rejected',
		    updated_at = $1
		WHERE node_id = $2
	`, time.Now().UTC(), nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to reject metric",
			"details": err.Error(),
		})
		return
	}

	// Add rejection comment
	h.addReviewComment(nodeID, request.StewardUser, request.Comment, "reject")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Metric rejected with feedback",
		"node_id": nodeID,
	})
}

// FlagMetric flags a metric for additional review
func (h *StewardApprovalHandler) FlagMetric(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	var request struct {
		StewardUser string `json:"steward_user" binding:"required"`
		Comment     string `json:"comment" binding:"required"`
		Severity    string `json:"severity,omitempty"` // low, medium, high, critical
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Add flag comment
	h.addReviewComment(nodeID, request.StewardUser, request.Comment, "flag")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Metric flagged for review",
		"node_id":  nodeID,
		"severity": request.Severity,
	})
}

// AddReviewComment adds a review comment to a metric
func (h *StewardApprovalHandler) AddReviewComment(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")

	var request struct {
		StewardUser string `json:"steward_user" binding:"required"`
		Comment     string `json:"comment" binding:"required"`
		Action      string `json:"action,omitempty"` // comment, approve, reject, flag
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	h.addReviewComment(nodeID, request.StewardUser, request.Comment, request.Action)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Comment added successfully",
		"node_id": nodeID,
	})
}

// addReviewComment is a helper method to add review comments
func (h *StewardApprovalHandler) addReviewComment(nodeID, user, comment, action string) error {
	commentJSON := fmt.Sprintf(`{
		"user": "%s",
		"comment": "%s",
		"timestamp": "%s",
		"action": "%s"
	}`, user, comment, time.Now().UTC().Format(time.RFC3339), action)

	_, err := h.db.Exec(`
		UPDATE public.catalog_node
		SET review_comments = COALESCE(review_comments, '[]'::jsonb) || $1::jsonb,
		    updated_at = $2
		WHERE node_id = $3
	`, commentJSON, time.Now().UTC(), nodeID)

	return err
}

// DynamicUnionHandler handles dynamic union table operations
type DynamicUnionHandler struct {
	db *sql.DB
}

// NewDynamicUnionHandler creates a new dynamic union handler
func NewDynamicUnionHandler(db *sql.DB) *DynamicUnionHandler {
	return &DynamicUnionHandler{db: db}
}

// CreateDynamicUnion creates a new dynamic union table definition
func (h *DynamicUnionHandler) CreateDynamicUnion(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name         string            `json:"name" binding:"required"`
		Description  string            `json:"description"`
		SourceTables []string          `json:"source_tables" binding:"required"`
		UnionType    string            `json:"union_type"`
		TableAliases map[string]string `json:"table_aliases,omitempty"`
		Tags         []string          `json:"tags,omitempty"`
		Owner        string            `json:"owner" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.UnionType == "" {
		request.UnionType = "UNION ALL"
	}

	nodeID := fmt.Sprintf("union_%s_%d", strings.ToLower(strings.ReplaceAll(request.Name, " ", "_")), time.Now().Unix())

	// Generate union SQL
	unionSQL := h.generateUnionSQL(request.SourceTables, request.UnionType, request.TableAliases)

	// Create schema definition
	schemaDef := map[string]interface{}{
		"node_id":       nodeID,
		"node_type":     "dynamic_union",
		"name":          request.Name,
		"description":   request.Description,
		"source_tables": request.SourceTables,
		"union_type":    request.UnionType,
		"union_sql":     unionSQL,
		"table_aliases": request.TableAliases,
		"tags":          request.Tags,
		"owner":         request.Owner,
		"version":       "1.0.0",
		"golden_path":   false,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"review_status": "draft",
	}

	schemaJSON, err := json.Marshal(schemaDef)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to marshal schema",
			"details": err.Error(),
		})
		return
	}

	// Store in catalog
	_, err = h.db.Exec(`
		INSERT INTO public.catalog_node (
			node_id, node_type, name, description, schema_def, owner,
			version, golden_path, review_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, nodeID, "dynamic_union", request.Name, request.Description, string(schemaJSON),
		request.Owner, "1.0.0", false, "draft", time.Now().UTC(), time.Now().UTC())

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to create union",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Dynamic union created successfully",
		"node_id":   nodeID,
		"union_sql": unionSQL,
	})
}

// generateUnionSQL generates the SQL for union operations
func (h *DynamicUnionHandler) generateUnionSQL(tables []string, unionType string, aliases map[string]string) string {
	var sqlParts []string

	for i, table := range tables {
		alias := aliases[table]
		if alias == "" {
			alias = fmt.Sprintf("t%d", i+1)
		}

		sqlParts = append(sqlParts, fmt.Sprintf("SELECT *, '%s' AS source_table FROM %s AS %s", table, table, alias))

		if i < len(tables)-1 {
			sqlParts = append(sqlParts, unionType)
		}
	}

	return strings.Join(sqlParts, "\n")
}

// GetDynamicUnions returns all dynamic union definitions
func (h *DynamicUnionHandler) GetDynamicUnions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT node_id, name, description, schema_def, review_status, golden_path
		FROM public.catalog_node
		WHERE node_type = 'dynamic_union'
		ORDER BY created_at DESC
	`)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch unions",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var unions []map[string]interface{}
	for rows.Next() {
		var nodeID, name, reviewStatus string
		var goldenPath bool
		var description, schemaDef sql.NullString
		rows.Scan(&nodeID, &name, &description, &schemaDef, &reviewStatus, &goldenPath)

		var schemaData map[string]interface{}
		if schemaDef.Valid && schemaDef.String != "" {
			json.Unmarshal([]byte(schemaDef.String), &schemaData)
		}

		desc := ""
		if description.Valid {
			desc = description.String
		}

		unions = append(unions, map[string]interface{}{
			"node_id":       nodeID,
			"name":          name,
			"description":   desc,
			"schema_def":    schemaData,
			"review_status": reviewStatus,
			"golden_path":   goldenPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unions": unions,
	})
}

// StringTimeDimensionHandler handles string-based time dimension operations
type StringTimeDimensionHandler struct {
	db *sql.DB
}

// NewStringTimeDimensionHandler creates a new string time dimension handler
func NewStringTimeDimensionHandler(db *sql.DB) *StringTimeDimensionHandler {
	return &StringTimeDimensionHandler{db: db}
}

// CreateStringTimeDimension creates a new string-based time dimension
func (h *StringTimeDimensionHandler) CreateStringTimeDimension(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		SourceColumn      string `json:"source_column" binding:"required"`
		SourceTable       string `json:"source_table" binding:"required"`
		DateFormat        string `json:"date_format" binding:"required"`
		TimeFormat        string `json:"time_format,omitempty"`
		Timezone          string `json:"timezone"`
		ParsingFunction   string `json:"parsing_function"`
		FallbackValue     string `json:"fallback_value,omitempty"`
		Owner             string `json:"owner" binding:"required"`
		MaterializeColumn bool   `json:"materialize_column"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.Timezone == "" {
		request.Timezone = "UTC"
	}
	if request.ParsingFunction == "" {
		request.ParsingFunction = "TO_TIMESTAMP"
	}

	nodeID := fmt.Sprintf("time_dim_%s_%d", strings.ToLower(strings.ReplaceAll(request.Name, " ", "_")), time.Now().Unix())

	// Generate parsing SQL
	parsingSQL := h.generateTimeParsingSQL(request)

	// Create schema definition
	schemaDef := map[string]interface{}{
		"node_id":            nodeID,
		"node_type":          "string_time_dimension",
		"name":               request.Name,
		"description":        request.Description,
		"source_column":      request.SourceColumn,
		"source_table":       request.SourceTable,
		"date_format":        request.DateFormat,
		"time_format":        request.TimeFormat,
		"timezone":           request.Timezone,
		"parsing_function":   request.ParsingFunction,
		"parsing_sql":        parsingSQL,
		"fallback_value":     request.FallbackValue,
		"materialize_column": request.MaterializeColumn,
		"owner":              request.Owner,
		"version":            "1.0.0",
		"golden_path":        false,
		"created_at":         time.Now().UTC().Format(time.RFC3339),
		"review_status":      "draft",
	}

	schemaJSON, err := json.Marshal(schemaDef)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to marshal schema",
			"details": err.Error(),
		})
		return
	}

	// Store in catalog
	_, err = h.db.Exec(`
		INSERT INTO public.catalog_node (
			node_id, node_type, name, description, schema_def, owner,
			version, golden_path, review_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, nodeID, "string_time_dimension", request.Name, request.Description, string(schemaJSON),
		request.Owner, "1.0.0", false, "draft", time.Now().UTC(), time.Now().UTC())

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to create time dimension",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "String time dimension created successfully",
		"node_id":     nodeID,
		"parsing_sql": parsingSQL,
	})
}

// generateTimeParsingSQL generates SQL for parsing string dates
func (h *StringTimeDimensionHandler) generateTimeParsingSQL(request interface{}) string {
	req := request.(map[string]interface{})

	sourceColumn := req["source_column"].(string)
	dateFormat := req["date_format"].(string)
	timeFormat := req["time_format"].(string)
	timezone := req["timezone"].(string)
	parsingFunction := req["parsing_function"].(string)
	fallbackValue := req["fallback_value"].(string)

	var sql string

	switch parsingFunction {
	case "TO_TIMESTAMP":
		if timeFormat != "" {
			sql = fmt.Sprintf("TO_TIMESTAMP(%s, '%s %s', '%s')", sourceColumn, dateFormat, timeFormat, timezone)
		} else {
			sql = fmt.Sprintf("TO_TIMESTAMP(%s, '%s')", sourceColumn, dateFormat)
		}
	case "PARSE_TIMESTAMP":
		if timeFormat != "" {
			sql = fmt.Sprintf("PARSE_TIMESTAMP('%s %s', %s)", dateFormat, timeFormat, sourceColumn)
		} else {
			sql = fmt.Sprintf("PARSE_TIMESTAMP('%s', %s)", dateFormat, sourceColumn)
		}
	default:
		sql = fmt.Sprintf("TO_TIMESTAMP(%s, '%s')", sourceColumn, dateFormat)
	}

	if fallbackValue != "" {
		sql = fmt.Sprintf("COALESCE(%s, '%s')", sql, fallbackValue)
	}

	return sql
}

// CustomGranularityHandler handles custom time granularity operations
type CustomGranularityHandler struct {
	db *sql.DB
}

// NewCustomGranularityHandler creates a new custom granularity handler
func NewCustomGranularityHandler(db *sql.DB) *CustomGranularityHandler {
	return &CustomGranularityHandler{db: db}
}

// CreateCustomGranularity creates a new custom time granularity
func (h *CustomGranularityHandler) CreateCustomGranularity(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		Dimension    string `json:"dimension" binding:"required"`
		Interval     string `json:"interval" binding:"required"`
		OffsetDays   int    `json:"offset_days"`
		OffsetHours  int    `json:"offset_hours"`
		FiscalLabel  string `json:"fiscal_label,omitempty"`
		CalendarType string `json:"calendar_type"`
		WeekStartDay string `json:"week_start_day"`
		Owner        string `json:"owner" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.CalendarType == "" {
		request.CalendarType = "gregorian"
	}
	if request.WeekStartDay == "" {
		request.WeekStartDay = "monday"
	}

	nodeID := fmt.Sprintf("gran_%s_%d", strings.ToLower(strings.ReplaceAll(request.Name, " ", "_")), time.Now().Unix())

	// Generate granularity SQL
	granularitySQL := h.generateGranularitySQL(request)

	// Create schema definition
	schemaDef := map[string]interface{}{
		"node_id":         nodeID,
		"node_type":       "custom_granularity",
		"name":            request.Name,
		"description":     request.Description,
		"dimension":       request.Dimension,
		"interval":        request.Interval,
		"offset_days":     request.OffsetDays,
		"offset_hours":    request.OffsetHours,
		"fiscal_label":    request.FiscalLabel,
		"calendar_type":   request.CalendarType,
		"week_start_day":  request.WeekStartDay,
		"granularity_sql": granularitySQL,
		"owner":           request.Owner,
		"version":         "1.0.0",
		"golden_path":     false,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
		"review_status":   "draft",
	}

	schemaJSON, err := json.Marshal(schemaDef)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to marshal schema",
			"details": err.Error(),
		})
		return
	}

	// Store in catalog
	_, err = h.db.Exec(`
		INSERT INTO public.catalog_node (
			node_id, node_type, name, description, schema_def, owner,
			version, golden_path, review_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, nodeID, "custom_granularity", request.Name, request.Description, string(schemaJSON),
		request.Owner, "1.0.0", false, "draft", time.Now().UTC(), time.Now().UTC())

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to create granularity",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Custom granularity created successfully",
		"node_id":         nodeID,
		"granularity_sql": granularitySQL,
	})
}

// generateGranularitySQL generates SQL for custom time granularities
func (h *CustomGranularityHandler) generateGranularitySQL(request interface{}) string {
	req := request.(map[string]interface{})

	dimension := req["dimension"].(string)
	interval := req["interval"].(string)
	offsetDays := req["offset_days"].(int)
	offsetHours := req["offset_hours"].(int)
	calendarType := req["calendar_type"].(string)
	weekStartDay := req["week_start_day"].(string)

	var sql string

	switch calendarType {
	case "fiscal":
		// Fiscal year calculation with offset
		sql = fmt.Sprintf(`
			CASE
				WHEN EXTRACT(MONTH FROM %s + INTERVAL '%d days') >= 7
				THEN CONCAT('FY', EXTRACT(YEAR FROM %s + INTERVAL '%d days') + 1)
				ELSE CONCAT('FY', EXTRACT(YEAR FROM %s + INTERVAL '%d days'))
			END
		`, dimension, offsetDays, dimension, offsetDays, dimension, offsetDays)
	case "iso_week":
		sql = fmt.Sprintf("EXTRACT(ISOYEAR FROM %s) || '-W' || LPAD(EXTRACT(ISOWEEK FROM %s)::TEXT, 2, '0')", dimension, dimension)
	case "custom":
		if weekStartDay != "monday" {
			// Adjust for different week start days
			dayOffset := map[string]int{
				"sunday":    0,
				"monday":    1,
				"tuesday":   2,
				"wednesday": 3,
				"thursday":  4,
				"friday":    5,
				"saturday":  6,
			}
			offset := dayOffset[weekStartDay]
			sql = fmt.Sprintf(`
				DATE_TRUNC('week', %s + INTERVAL '%d days') + INTERVAL '%d days'
			`, dimension, offset, -offset)
		} else {
			sql = fmt.Sprintf("DATE_TRUNC('%s', %s)", interval, dimension)
		}
	default: // gregorian
		if offsetDays != 0 || offsetHours != 0 {
			sql = fmt.Sprintf("DATE_TRUNC('%s', %s + INTERVAL '%d days %d hours')", interval, dimension, offsetDays, offsetHours)
		} else {
			sql = fmt.Sprintf("DATE_TRUNC('%s', %s)", interval, dimension)
		}
	}

	return strings.TrimSpace(sql)
}

// GetCustomGranularities returns all custom granularity definitions
func (h *CustomGranularityHandler) GetCustomGranularities(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT node_id, name, description, schema_def, review_status, golden_path
		FROM public.catalog_node
		WHERE node_type = 'custom_granularity'
		ORDER BY created_at DESC
	`)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to fetch granularities",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var granularities []map[string]interface{}
	for rows.Next() {
		var nodeID, name, reviewStatus string
		var goldenPath bool
		var description, schemaDef sql.NullString
		rows.Scan(&nodeID, &name, &description, &schemaDef, &reviewStatus, &goldenPath)

		var schemaData map[string]interface{}
		if schemaDef.Valid && schemaDef.String != "" {
			json.Unmarshal([]byte(schemaDef.String), &schemaData)
		}

		desc := ""
		if description.Valid {
			desc = description.String
		}

		granularities = append(granularities, map[string]interface{}{
			"node_id":       nodeID,
			"name":          name,
			"description":   desc,
			"schema_def":    schemaData,
			"review_status": reviewStatus,
			"golden_path":   goldenPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"granularities": granularities,
	})
}

// RegisterDynamicHandlers registers all dynamic handlers
func RegisterDynamicHandlers(r chi.Router, db *sql.DB) {
	paramHandler := NewDynamicParameterHandler(db)
	measureHandler := NewDynamicMeasureHandler(db)
	dimHandler := NewDynamicDimensionHandler(db)
	filterHandler := NewScopedFilterHandler(db)
	cohortHandler := NewCohortFilterHandler(db)
	lineageHandler := NewLineageVisualizationHandler(db)
	approvalHandler := NewStewardApprovalHandler(db)
	unionHandler := NewDynamicUnionHandler(db)
	timeDimHandler := NewStringTimeDimensionHandler(db)
	granHandler := NewCustomGranularityHandler(db)

	r.Route("/dynamic", func(r chi.Router) {
		r.Route("/parameters", func(r chi.Router) {
			r.Get("/{type}/{name}", paramHandler.GetAvailableValues)
			r.Get("/schema", paramHandler.GetParameterSchema)
		})
		r.Route("/measures", func(r chi.Router) {
			r.Get("/generate", measureHandler.GenerateDynamicMeasures)
			r.Get("/catalog", measureHandler.GetDynamicMeasureCatalog)
			r.Post("/validate", measureHandler.ValidateDynamicMeasure)
		})
		r.Route("/dimensions", func(r chi.Router) {
			r.Get("/", dimHandler.GetDynamicDimensions)
			r.Get("/{dimension}/values", dimHandler.GetDimensionValues)
		})
		r.Route("/filters", func(r chi.Router) {
			r.Get("/", filterHandler.GetScopedFilters)
			r.Post("/apply", filterHandler.ApplyScopedFilter)
		})
		r.Route("/cohorts", func(r chi.Router) {
			r.Get("/", cohortHandler.GetCohorts)
			r.Get("/{cohort}/values", cohortHandler.GetCohortValues)
			r.Post("/apply", cohortHandler.ApplyCohortFilter)
		})
		r.Route("/lineage", func(r chi.Router) {
			r.Get("/{node_id}", lineageHandler.GetLineage)
			r.Get("/{node_id}/graph", lineageHandler.GetLineageGraph)
		})
		r.Route("/approvals", func(r chi.Router) {
			r.Get("/pending", approvalHandler.GetPendingApprovals)
			r.Post("/{node_id}/approve", approvalHandler.ApproveMetric)
			r.Post("/{node_id}/reject", approvalHandler.RejectMetric)
			r.Post("/{node_id}/flag", approvalHandler.FlagMetric)
			r.Post("/{node_id}/comment", approvalHandler.AddReviewComment)
		})
		r.Route("/unions", func(r chi.Router) {
			r.Post("/", unionHandler.CreateDynamicUnion)
			r.Get("/", unionHandler.GetDynamicUnions)
		})
		r.Route("/time-dimensions", func(r chi.Router) {
			r.Post("/", timeDimHandler.CreateStringTimeDimension)
		})
		r.Route("/granularities", func(r chi.Router) {
			r.Post("/", granHandler.CreateCustomGranularity)
			r.Get("/", granHandler.GetCustomGranularities)
		})
	})
}
