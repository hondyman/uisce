package cube

import (
	"database/sql"
	"strings"
)

type DatabaseJoinExtractor struct {
	db *sql.DB
}

func NewDatabaseJoinExtractor(db *sql.DB) *DatabaseJoinExtractor {
	return &DatabaseJoinExtractor{db: db}
}

type JoinSuggestion struct {
	SourceTable  string `json:"source_table"`
	TargetTable  string `json:"target_table"`
	SourceColumn string `json:"source_column"`
	TargetColumn string `json:"target_column"`
	Relationship string `json:"relationship"`
	JoinSQL      string `json:"join_sql"`
	Description  string `json:"description"`
}

func (e *DatabaseJoinExtractor) ExtractJoins(datasourceID string) ([]JoinSuggestion, error) {
	return []JoinSuggestion{}, nil
}

func (e *DatabaseJoinExtractor) GenerateJoinDefinitions(tableName string, datasourceID string) (map[string]map[string]any, error) {
	return make(map[string]map[string]any), nil
}

type TableColumn struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

func (e *DatabaseJoinExtractor) GetTableColumns(tableName string) ([]TableColumn, error) {
	return []TableColumn{}, nil
}

func (e *DatabaseJoinExtractor) GenerateCubeFromTable(tableName string, datasourceID string) (*Cube, error) {
	return &Cube{Name: tableName}, nil
}

func mapDatabaseTypeToCubeType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.Contains(dbType, "int") || strings.Contains(dbType, "decimal") ||
		strings.Contains(dbType, "numeric") || strings.Contains(dbType, "float") ||
		strings.Contains(dbType, "double"):
		return "number"
	case strings.Contains(dbType, "timestamp") || strings.Contains(dbType, "datetime") ||
		strings.Contains(dbType, "date"):
		return "time"
	case strings.Contains(dbType, "bool"):
		return "boolean"
	default:
		return "string"
	}
}

func isNumericType(dbType string) bool {
	dbType = strings.ToLower(dbType)
	return strings.Contains(dbType, "int") || strings.Contains(dbType, "decimal") ||
		strings.Contains(dbType, "numeric") || strings.Contains(dbType, "float") ||
		strings.Contains(dbType, "double")
}

func formatTitle(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func boolPtr(b bool) *bool {
	return &b
}
