package analytics

import (
	"strings"

	"github.com/hondyman/semlayer/backend/internal/boresolver"
	"github.com/jmoiron/sqlx"
)

// CatalogTypeEnv implements boresolver.TypeEnv using the database catalog
type CatalogTypeEnv struct {
	types map[string]boresolver.ExprType
}

// NewCatalogTypeEnv creates a type environment for a specific BO
func NewCatalogTypeEnv(db *sqlx.DB, boID string) (*CatalogTypeEnv, error) {
	// Fetch all terms for this BO and determine their types
	query := `
		SELECT 
			st.node_name, 
			st.properties->>'data_type' as data_type
		FROM catalog_edge ce
		JOIN catalog_node st ON ce.target_node_id = st.id
		WHERE ce.source_node_id = $1 
		  AND ce.edge_type = 'HAS_ATTRIBUTE'
	`
	rows, err := db.Query(query, boID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeMap := make(map[string]boresolver.ExprType)
	for rows.Next() {
		var name string
		var dataType *string
		if err := rows.Scan(&name, &dataType); err != nil {
			continue
		}

		t := boresolver.TypeUnknown
		if dataType != nil {
			t = mapDataType(*dataType)
		}
		typeMap[name] = t
	}

	return &CatalogTypeEnv{types: typeMap}, nil
}

func (e *CatalogTypeEnv) TermType(name string) boresolver.ExprType {
	if t, ok := e.types[name]; ok {
		return t
	}
	return boresolver.TypeUnknown
}

func mapDataType(dt string) boresolver.ExprType {
	switch strings.ToLower(dt) {
	case "integer", "bigint", "smallint", "numeric", "decimal", "double", "float", "real":
		return boresolver.TypeNumber
	case "boolean":
		return boresolver.TypeBool
	case "date", "timestamp", "timestamptz":
		return boresolver.TypeDate
	case "string", "text", "varchar":
		return boresolver.TypeString
	default:
		return boresolver.TypeString
	}
}
