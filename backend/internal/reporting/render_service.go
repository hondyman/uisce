package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

type QueryResult struct {
	Columns           []string                 `json:"columns"`
	Rows              []map[string]interface{} `json:"rows"`
	RowCount          int                      `json:"rowCount"`
	GeneratedAt       string                   `json:"generatedAt"`
	BOKey             string                   `json:"boKey,omitempty"`
	SQLQuery          string                   `json:"sqlQuery,omitempty"`
	ParametersApplied map[string]interface{}   `json:"parametersApplied,omitempty"`
	Error             string                   `json:"error,omitempty"`
}

type ReportRenderService struct {
	db     *sqlx.DB
	boRepo *boresolver.PostgresBORepository
}

func NewReportRenderService(db *sqlx.DB, boRepo *boresolver.PostgresBORepository) *ReportRenderService {
	return &ReportRenderService{
		db:     db,
		boRepo: boRepo,
	}
}

type semanticBinding struct {
	BOPath         string   `json:"bo_path"`
	FieldAllowlist []string `json:"field_allowlist"`
	Measures       []struct {
		Field       string `json:"field"`
		Aggregation string `json:"aggregation"`
	} `json:"measures"`
	Dimensions []string `json:"dimensions"`
	Filters    []struct {
		Field     string      `json:"field"`
		Operator  string      `json:"operator"`
		Value     interface{} `json:"value"`
		Parameter string      `json:"parameter,omitempty"`
	} `json:"filters"`
}

type semanticQuery struct {
	DataBindings []semanticBinding `json:"data_bindings"`
}

func (s *ReportRenderService) RenderByKey(
	ctx context.Context,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	reportKey string,
	parameters json.RawMessage,
) (*QueryResult, error) {
	repo := NewRepository(s.db)
	def, err := repo.GetDefinitionByKey(ctx, tenantID, datasourceID, reportKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get report definition: %w", err)
	}
	if def == nil {
		return nil, fmt.Errorf("report not found: %s", reportKey)
	}

	if len(def.SemanticQuery) == 0 {
		return nil, fmt.Errorf("report has no semantic_query")
	}

	var sq semanticQuery
	if err := json.Unmarshal(def.SemanticQuery, &sq); err != nil {
		return nil, fmt.Errorf("failed to parse semantic_query: %w", err)
	}
	if len(sq.DataBindings) == 0 {
		return nil, fmt.Errorf("semantic_query has no data_bindings")
	}

	var paramsMap map[string]interface{}
	if len(parameters) > 0 {
		var rawMap map[string]interface{}
		if err := json.Unmarshal(parameters, &rawMap); err == nil {
			if nested, ok := rawMap["parameters"].(map[string]interface{}); ok {
				paramsMap = nested
			} else {
				paramsMap = rawMap
			}
		}
	}

	for _, sb := range sq.DataBindings {
		result, err := s.executeSemanticBinding(ctx, tenantID, sb, paramsMap)
		if err != nil {
			continue
		}
		return result, nil
	}

	return &QueryResult{Error: "no bindings could be executed"}, nil
}

func (s *ReportRenderService) executeSemanticBinding(
	ctx context.Context,
	tenantID uuid.UUID,
	sb semanticBinding,
	paramsMap map[string]interface{},
) (*QueryResult, error) {
	boPath := sb.BOPath

	childBOID, childErr := s.resolveBOIDByKey(ctx, tenantID, boPath)
	boID := childBOID
	if childErr != nil {
		parentKey := normalizeBOPath(boPath)
		boID, childErr = s.resolveBOIDByKey(ctx, tenantID, parentKey)
		if childErr != nil {
			return nil, fmt.Errorf("failed to resolve BO %s (or %s): %w", boPath, parentKey, childErr)
		}
	}

	boDef, err := s.boRepo.GetBODefinition(boID)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO definition: %w", err)
	}

	fieldNameMap := make(map[string]bool)
	for _, f := range boDef.Fields {
		fieldNameMap[f.Name] = true
	}

	var dims []string
	for _, d := range sb.Dimensions {
		if fieldNameMap[d] {
			dims = append(dims, d)
		}
	}

	type measureDef struct {
		Field       string `json:"field"`
		Aggregation string `json:"aggregation"`
	}
	var filteredMeasures []measureDef
	for _, m := range sb.Measures {
		if fieldNameMap[m.Field] {
			filteredMeasures = append(filteredMeasures, measureDef{Field: m.Field, Aggregation: m.Aggregation})
		}
	}

	if len(dims) == 0 && len(filteredMeasures) == 0 || (len(sb.Measures) > 0 && len(filteredMeasures) == 0) {
		return nil, fmt.Errorf("no valid fields found in binding")
	}

	var selectedCols []string
	var groupByCols []string
	argIdx := 2
	var args []interface{}
	appliedParams := make(map[string]interface{})

	for _, dim := range dims {
		col := sanitizeColName(dim)
		selectedCols = append(selectedCols, fmt.Sprintf("t0.%s AS %q", col, dim))
		groupByCols = append(groupByCols, fmt.Sprintf("t0.%s", col))
	}

	for _, m := range filteredMeasures {
		agg := strings.ToUpper(m.Aggregation)
		col := sanitizeColName(m.Field)
		switch agg {
		case "SUM", "AVG", "COUNT", "MIN", "MAX":
			selectedCols = append(selectedCols, fmt.Sprintf("%s(t0.%s) AS %q", agg, col, m.Field))
		default:
			selectedCols = append(selectedCols, fmt.Sprintf("t0.%s AS %q", col, m.Field))
		}
	}

	tableName := sanitizeTableNameForSQL(boDef.DrivingTable)

	var whereClause string
	if subtype := extractSubtype(boPath); subtype != "" {
		whereClause = fmt.Sprintf("WHERE t0.tenant_id = $1 AND t0.subtype_code = $%d AND t0.valid_to IS NULL", argIdx)
		args = append(args, tenantID.String(), subtype)
		argIdx += 2
	} else {
		whereClause = "WHERE t0.tenant_id = $1 AND t0.valid_to IS NULL"
		args = append(args, tenantID.String())
	}

	// Filter fields that are explicitly configured in the binding
	filteredFields := make(map[string]bool)

	for _, f := range sb.Filters {
		if !fieldNameMap[f.Field] {
			continue
		}
		filteredFields[f.Field] = true
		col := sanitizeColName(f.Field)
		op := normalizeOperator(f.Operator)
		if op == "" {
			continue
		}

		// Resolve parameter dynamic override if present
		filterVal := f.Value
		if f.Parameter != "" {
			if pVal, exists := lookupParam(paramsMap, f.Parameter); exists {
				filterVal = pVal
				appliedParams[f.Parameter] = pVal
			}
		} else if strVal, isStr := f.Value.(string); isStr {
			if strings.HasPrefix(strVal, "@") {
				paramName := strings.TrimPrefix(strVal, "@")
				if pVal, exists := lookupParam(paramsMap, paramName); exists {
					filterVal = pVal
					appliedParams[paramName] = pVal
				}
			} else if strings.HasPrefix(strVal, "Parameters!") && strings.HasSuffix(strVal, ".Value") {
				paramName := strings.TrimSuffix(strings.TrimPrefix(strVal, "Parameters!"), ".Value")
				if pVal, exists := lookupParam(paramsMap, paramName); exists {
					filterVal = pVal
					appliedParams[paramName] = pVal
				}
			}
		} else if filterVal == nil || filterVal == "" {
			if pVal, exists := lookupParam(paramsMap, f.Field); exists {
				filterVal = pVal
				appliedParams[f.Field] = pVal
			}
		}

		cond, val, ok := buildFilterClause(op, filterVal, argIdx)
		if !ok {
			continue
		}
		whereClause += fmt.Sprintf(" AND t0.%s %s", col, cond)
		if val != nil {
			args = append(args, val)
			argIdx++
		}
	}

	// Also dynamically apply any passed parameters that directly match table fields
	for pKey, pVal := range paramsMap {
		if pVal == nil || pVal == "" {
			continue
		}
		// Match field name (e.g. client_id, status, year)
		for fieldName := range fieldNameMap {
			if !filteredFields[fieldName] && (strings.EqualFold(fieldName, pKey) || strings.EqualFold(fieldName, strings.ReplaceAll(pKey, "_", ""))) {
				col := sanitizeColName(fieldName)
				cond, val, ok := buildFilterClause("=", pVal, argIdx)
				if ok && val != nil {
					whereClause += fmt.Sprintf(" AND t0.%s %s", col, cond)
					args = append(args, val)
					argIdx++
					appliedParams[pKey] = pVal
					filteredFields[fieldName] = true
				}
			}
		}
	}

	limit := 1000
	var sqlStr string
	if len(groupByCols) > 0 {
		sqlStr = fmt.Sprintf("SELECT %s FROM %s AS t0 %s GROUP BY %s LIMIT %d",
			strings.Join(selectedCols, ", "),
			tableName,
			whereClause,
			strings.Join(groupByCols, ", "),
			limit)
	} else {
		sqlStr = fmt.Sprintf("SELECT %s FROM %s AS t0 %s LIMIT %d",
			strings.Join(selectedCols, ", "),
			tableName,
			whereClause,
			limit)
	}

	rows, err := s.db.QueryxContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var resultRows []map[string]interface{}
	for rows.Next() {
		vals, err := rows.SliceScan()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				if num, err := strconv.ParseFloat(string(v), 64); err == nil {
					row[col] = num
				} else {
					row[col] = string(v)
				}
			default:
				row[col] = v
			}
		}
		resultRows = append(resultRows, row)
	}

	return &QueryResult{
		Columns:           cols,
		Rows:              resultRows,
		RowCount:          len(resultRows),
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		BOKey:             boPath,
		SQLQuery:          sqlStr,
		ParametersApplied: appliedParams,
	}, nil
}

func lookupParam(params map[string]interface{}, key string) (interface{}, bool) {
	if params == nil || key == "" {
		return nil, false
	}
	if val, ok := params[key]; ok && val != nil && val != "" {
		return val, true
	}
	lowerKey := strings.ToLower(key)
	for k, v := range params {
		if strings.ToLower(k) == lowerKey && v != nil && v != "" {
			return v, true
		}
		cleanK := strings.ReplaceAll(strings.ToLower(k), "_", "")
		cleanTarget := strings.ReplaceAll(lowerKey, "_", "")
		if cleanK == cleanTarget && v != nil && v != "" {
			return v, true
		}
	}
	return nil, false
}

func sanitizeTableNameForSQL(tableName string) string {
	parts := strings.Split(tableName, ".")
	if len(parts) == 2 {
		return fmt.Sprintf("%q.%q", parts[0], parts[1])
	}
	return fmt.Sprintf("%q", tableName)
}

func sanitizeColName(name string) string {
	return fmt.Sprintf("%q", name)
}

func normalizeBOPath(boPath string) string {
	if strings.Contains(boPath, "/") {
		parts := strings.Split(boPath, "/")
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return boPath
}

func extractSubtype(boPath string) string {
	if strings.Contains(boPath, "/") {
		parts := strings.Split(boPath, "/")
		return parts[len(parts)-1]
	}
	return ""
}

func normalizeOperator(op string) string {
	switch strings.ToLower(op) {
	case "equals", "eq", "=":
		return "="
	case "not_equals", "neq", "!=":
		return "!="
	case "greater_than", "gt", ">":
		return ">"
	case "less_than", "lt", "<":
		return "<"
	case "greater_equal", "gte", ">=":
		return ">="
	case "less_equal", "lte", "<=":
		return "<="
	case "in":
		return "IN"
	case "not_in":
		return "NOT IN"
	case "is_null":
		return "IS NULL"
	case "is_not_null":
		return "IS NOT NULL"
	case "contains":
		return "ILIKE"
	case "starts_with":
		return "ILIKE"
	case "ends_with":
		return "ILIKE"
	default:
		return ""
	}
}

func buildFilterClause(op string, value interface{}, argIdx int) (string, interface{}, bool) {
	// If the value is a slice/array, upgrade '=' to 'IN' / '= ANY($%d)'
	if sliceVal, ok := value.([]interface{}); ok {
		if len(sliceVal) == 0 {
			return "", nil, false
		}
		strSlice := make([]string, 0, len(sliceVal))
		for _, item := range sliceVal {
			strSlice = append(strSlice, fmt.Sprintf("%v", item))
		}
		if op == "=" || op == "IN" {
			return fmt.Sprintf("= ANY($%d)", argIdx), strSlice, true
		} else if op == "!=" || op == "NOT IN" {
			return fmt.Sprintf("!= ALL($%d)", argIdx), strSlice, true
		}
	} else if strSlice, ok := value.([]string); ok {
		if len(strSlice) == 0 {
			return "", nil, false
		}
		if op == "=" || op == "IN" {
			return fmt.Sprintf("= ANY($%d)", argIdx), strSlice, true
		} else if op == "!=" || op == "NOT IN" {
			return fmt.Sprintf("!= ALL($%d)", argIdx), strSlice, true
		}
	}

	// Resolve relative date strings if present
	if strVal, ok := value.(string); ok {
		value = resolveRelativeDateBackend(strVal)
	}

	switch op {
	case "=", "!=", ">", "<", ">=", "<=":
		return fmt.Sprintf("%s $%d", op, argIdx), value, true
	case "IS NULL":
		return "IS NULL", nil, false
	case "IS NOT NULL":
		return "IS NOT NULL", nil, false
	case "IN", "NOT IN":
		return compileInFilter(op, value, argIdx)
	case "ILIKE":
		return compileLikeFilter(op, value, argIdx)
	default:
		return "", nil, false
	}
}

func resolveRelativeDateBackend(keyword string) string {
	now := time.Now().UTC()
	year := now.Year()
	month := now.Month()

	switch strings.ToUpper(strings.TrimSpace(keyword)) {
	case "TODAY", "NOW", "CURRENT_DATE":
		return now.Format("2006-01-02")
	case "YESTERDAY":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "START_OF_YEAR", "YTD", "YEAR_TO_DATE":
		return fmt.Sprintf("%d-01-01", year)
	case "END_OF_YEAR":
		return fmt.Sprintf("%d-12-31", year)
	case "START_OF_MONTH", "MTD", "MONTH_TO_DATE":
		return fmt.Sprintf("%d-%02d-01", year, month)
	case "PREV_MONTH", "START_OF_PREV_MONTH":
		prevMonthDate := now.AddDate(0, -1, 0)
		return fmt.Sprintf("%d-%02d-01", prevMonthDate.Year(), prevMonthDate.Month())
	default:
		return keyword
	}
}

func compileInFilter(op string, value interface{}, argIdx int) (string, interface{}, bool) {
	switch v := value.(type) {
	case []interface{}:
		return fmt.Sprintf("%s $%d::text[]", op, argIdx), fmt.Sprintf("{%s}", strings.Join(toStringArray(v), ",")), true
	case string:
		return fmt.Sprintf("%s $%d::text[]", op, argIdx), fmt.Sprintf("{%s}", v), true
	}
	return "", nil, false
}

func compileLikeFilter(op string, value interface{}, argIdx int) (string, interface{}, bool) {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%s $%d", op, argIdx), "%" + v + "%", true
	}
	return "", nil, false
}

func toStringArray(vals []interface{}) []string {
	result := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(string); ok {
			result = append(result, fmt.Sprintf("%q", s))
		}
	}
	return result
}

func (s *ReportRenderService) resolveBOIDByKey(ctx context.Context, tenantID uuid.UUID, boKey string) (string, error) {
	var boID string
	err := s.db.GetContext(ctx, &boID,
		`SELECT id FROM business_objects WHERE tenant_id = $1 AND key = $2 LIMIT 1`,
		tenantID, boKey)
	if err == nil {
		return boID, nil
	}

	err = s.db.GetContext(ctx, &boID,
		`SELECT id FROM business_objects WHERE tenant_id = $1 AND key = $2 LIMIT 1`,
		uuid.Nil, boKey)
	if err == nil {
		return boID, nil
	}

	return "", fmt.Errorf("business object not found: %s", boKey)
}
