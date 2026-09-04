package metadata

import (
	"fmt"
	"strings"
	"time"
)

// PolyglotDialect identifies the physical storage engine
type PolyglotDialect string

const (
	DialectIceberg    PolyglotDialect = "ICEBERG"
	DialectStarRocks  PolyglotDialect = "STARROCKS"
	DialectPostgres   PolyglotDialect = "POSTGRES"
	DialectTrino      PolyglotDialect = "TRINO"
	DialectDataFusion PolyglotDialect = "DATAFUSION"
)

// BiTemporalConfig defines the column names for bi-temporal table time boundaries
type BiTemporalConfig struct {
	ValidStartCol       string `json:"valid_time_start_col"`
	ValidEndCol         string `json:"valid_time_end_col"`
	TransactionStartCol string `json:"transaction_time_start_col"`
	TransactionEndCol   string `json:"transaction_time_end_col"`
}

// PolyglotQueryRequest describes a query against a potentially bi-temporal BO binding
type PolyglotQueryRequest struct {
	TenantID            string
	BOID                string
	TableName           string
	Dialect             PolyglotDialect
	BiTemporalConfig    *BiTemporalConfig
	AsOfValidTime       string // ISO 8601 timestamp
	AsOfTransactionTime string // ISO 8601 timestamp
	SelectColumns       []string
	WhereClause         string
	OrderBy             string
	Limit               int
}

// PolyglotQueryResult carries the compiled SQL and its dialect metadata
type PolyglotQueryResult struct {
	SQL            string
	Dialect        PolyglotDialect
	IsTimeTravel   bool
	CompiledAt     time.Time
	SnapshotClause string // Iceberg/StarRocks FOR SYSTEM_TIME clause if used
}

// BuildBiTemporalWhereClause appends time-travel bounds to OLAP analytics queries
// (Valid Time and Transaction Time predicates)
func BuildBiTemporalWhereClause(config BiTemporalConfig, asOfValidTime, asOfTransactionTime string) string {
	var predicates []string

	if asOfValidTime != "" && config.ValidStartCol != "" && config.ValidEndCol != "" {
		p := fmt.Sprintf("('%s' >= %s AND '%s' < COALESCE(%s, '9999-12-31 23:59:59'))",
			asOfValidTime, config.ValidStartCol, asOfValidTime, config.ValidEndCol)
		predicates = append(predicates, p)
	}

	if asOfTransactionTime != "" && config.TransactionStartCol != "" && config.TransactionEndCol != "" {
		p := fmt.Sprintf("('%s' >= %s AND '%s' < COALESCE(%s, '9999-12-31 23:59:59'))",
			asOfTransactionTime, config.TransactionStartCol, asOfTransactionTime, config.TransactionEndCol)
		predicates = append(predicates, p)
	}

	if len(predicates) == 0 {
		return ""
	}
	return "AND " + strings.Join(predicates, " AND ")
}

// CompilePolyglotQuery generates dialect-aware, time-travel-enabled SQL for any storage backend
func CompilePolyglotQuery(req PolyglotQueryRequest) (*PolyglotQueryResult, error) {
	result := &PolyglotQueryResult{
		Dialect:     req.Dialect,
		CompiledAt:  time.Now(),
		IsTimeTravel: req.AsOfValidTime != "" || req.AsOfTransactionTime != "",
	}

	cols := "*"
	if len(req.SelectColumns) > 0 {
		cols = strings.Join(req.SelectColumns, ", ")
	}

	switch req.Dialect {
	case DialectIceberg, DialectDataFusion:
		result.SQL = compileIcebergQuery(req, cols, result)
	case DialectStarRocks, DialectTrino:
		result.SQL = compileStarRocksQuery(req, cols, result)
	case DialectPostgres:
		result.SQL = compilePostgresQuery(req, cols, result)
	default:
		result.SQL = compilePostgresQuery(req, cols, result)
	}

	return result, nil
}

// compileIcebergQuery generates Apache Iceberg time-travel SQL
// Uses native FOR SYSTEM_TIME AS OF timestamp syntax
func compileIcebergQuery(req PolyglotQueryRequest, cols string, result *PolyglotQueryResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SELECT %s FROM %s", cols, req.TableName))

	// Iceberg native snapshot-based time travel for transaction time
	if req.AsOfTransactionTime != "" {
		result.SnapshotClause = fmt.Sprintf("FOR SYSTEM_TIME AS OF TIMESTAMP '%s'", req.AsOfTransactionTime)
		sb.WriteString(" " + result.SnapshotClause)
	}

	var conditions []string
	if req.WhereClause != "" {
		conditions = append(conditions, req.WhereClause)
	}
	// Valid time predicate applied as WHERE clause
	if req.BiTemporalConfig != nil && req.AsOfValidTime != "" {
		vtClause := BuildBiTemporalWhereClause(BiTemporalConfig{
			ValidStartCol: req.BiTemporalConfig.ValidStartCol,
			ValidEndCol:   req.BiTemporalConfig.ValidEndCol,
		}, req.AsOfValidTime, "")
		if vtClause != "" {
			conditions = append(conditions, strings.TrimPrefix(vtClause, "AND "))
		}
	}
	if len(conditions) > 0 {
		sb.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	if req.OrderBy != "" {
		sb.WriteString(" ORDER BY " + req.OrderBy)
	}
	if req.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", req.Limit))
	}
	return sb.String()
}

// compileStarRocksQuery generates StarRocks time-travel SQL
func compileStarRocksQuery(req PolyglotQueryRequest, cols string, result *PolyglotQueryResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SELECT %s FROM %s", cols, req.TableName))

	if req.AsOfTransactionTime != "" {
		result.SnapshotClause = fmt.Sprintf("FOR SYSTEM_TIME AS OF '%s'", req.AsOfTransactionTime)
		sb.WriteString(" " + result.SnapshotClause)
	}

	var conditions []string
	if req.WhereClause != "" {
		conditions = append(conditions, req.WhereClause)
	}
	if req.BiTemporalConfig != nil && req.AsOfValidTime != "" {
		vtClause := BuildBiTemporalWhereClause(BiTemporalConfig{
			ValidStartCol: req.BiTemporalConfig.ValidStartCol,
			ValidEndCol:   req.BiTemporalConfig.ValidEndCol,
		}, req.AsOfValidTime, "")
		if vtClause != "" {
			conditions = append(conditions, strings.TrimPrefix(vtClause, "AND "))
		}
	}
	if len(conditions) > 0 {
		sb.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	if req.OrderBy != "" {
		sb.WriteString(" ORDER BY " + req.OrderBy)
	}
	if req.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", req.Limit))
	}
	return sb.String()
}

// compilePostgresQuery generates Postgres temporal SQL using table predicates
// Postgres requires application-managed temporal columns (no native FOR SYSTEM_TIME)
func compilePostgresQuery(req PolyglotQueryRequest, cols string, result *PolyglotQueryResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SELECT %s FROM %s", cols, req.TableName))

	var conditions []string
	if req.WhereClause != "" {
		conditions = append(conditions, req.WhereClause)
	}
	if req.BiTemporalConfig != nil {
		biClause := BuildBiTemporalWhereClause(*req.BiTemporalConfig, req.AsOfValidTime, req.AsOfTransactionTime)
		if biClause != "" {
			// Remove leading "AND " since we're building conditions separately
			conditions = append(conditions, strings.TrimPrefix(biClause, "AND "))
		}
	}
	if len(conditions) > 0 {
		sb.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	if req.OrderBy != "" {
		sb.WriteString(" ORDER BY " + req.OrderBy)
	}
	if req.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", req.Limit))
	}
	return sb.String()
}

// ResolveBindingForQuery selected a binding by BindingMode (OLTP_CRUD /
// BI_TEMPORAL_OLAP / OLAP_READONLY) for time-travel vs. live queries. Removed:
// the live business_object_bindings table has no binding_mode column (see
// BusinessObjectBinding's docstring in binding_service.go) and never has —
// this function was already unreachable dead code (no caller outside its own
// package) built against a schema that was never applied. If per-mode
// binding selection is needed again, it needs a real column to read, not a
// revived version of this.
func ResolveBindingForQuery(bindings []BusinessObjectBinding, asOfTime string) *BusinessObjectBinding {
	if len(bindings) > 0 {
		return &bindings[0]
	}
	return nil
}
