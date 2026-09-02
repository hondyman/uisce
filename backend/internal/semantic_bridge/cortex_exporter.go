package semantic_bridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type CortexModelSpec struct {
	Name            string                 `yaml:"name"`
	Description     string                 `yaml:"description"`
	Tables          []CortexTableSpec      `yaml:"tables"`
	Relationships   []CortexRelationship   `yaml:"relationships,omitempty"`
	VerifiedQueries []CortexVerifiedQuery  `yaml:"verified_queries,omitempty"`
}

type CortexVerifiedQuery struct {
	Name                    string   `yaml:"name"`
	Question                string   `yaml:"question"`
	SQL                     string   `yaml:"sql"`
	VerifiedBy              string   `yaml:"verified_by,omitempty"`
	VerifiedAt              string   `yaml:"verified_at,omitempty"`
	UseAsOnboardingExample  bool     `yaml:"use_as_onboarding_example,omitempty"`
}

type CortexTableSpec struct {
	Name           string            `yaml:"name"`
	BaseTable      CortexBaseTable   `yaml:"base_table"`
	Dimensions     []CortexDimension `yaml:"dimensions,omitempty"`
	TimeDimensions []CortexTimeDim   `yaml:"time_dimensions,omitempty"`
	Measures       []CortexMeasure   `yaml:"measures,omitempty"`
}

type CortexBaseTable struct {
	Database string `yaml:"database"`
	Schema   string `yaml:"schema"`
	Table    string `yaml:"table"`
}

type CortexDimension struct {
	Name         string   `yaml:"name"`
	Expr         string   `yaml:"expr"`
	DataType     string   `yaml:"data_type"`
	Description  string   `yaml:"description,omitempty"`
	Synonyms     []string `yaml:"synonyms,omitempty"`
	SampleValues []string `yaml:"sample_values,omitempty"`
}

type CortexTimeDim struct {
	Name     string   `yaml:"name"`
	Expr     string   `yaml:"expr"`
	DataType string   `yaml:"data_type"`
	Synonyms []string `yaml:"synonyms,omitempty"`
}

type CortexMeasure struct {
	Name               string   `yaml:"name"`
	Expr               string   `yaml:"expr"`
	DataType           string   `yaml:"data_type"`
	DefaultAggregation string   `yaml:"default_aggregation"`
	Description        string   `yaml:"description,omitempty"`
	Synonyms           []string `yaml:"synonyms,omitempty"`
}

type CortexRelationship struct {
	Name             string `yaml:"name"`
	LeftTable        string `yaml:"left_table"`
	RightTable       string `yaml:"right_table"`
	RelationshipType string `yaml:"relationship_type"`
	JoinCondition    string `yaml:"join_condition"`
}

type GovernanceTagStatement struct {
	TableName string `json:"table_name"`
	ColumnName string `json:"column_name"`
	TagKey    string `json:"tag_key"`
	TagValue  string `json:"tag_value"`
	SQL       string `json:"sql"`
}

type CortexExporter struct {
	db *sqlx.DB
}

func NewCortexExporter(db *sqlx.DB) *CortexExporter {
	return &CortexExporter{db: db}
}

// CompileFullCortexModel generates the unified multi-table YAML model for the tenant
func (e *CortexExporter) CompileFullCortexModel(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	// 1. Discover all published Business Objects for this tenant or Gold Copy
	boQuery := `
		SELECT bo.id, bo.bo_key, bo.bo_name, COALESCE(bo.description, '') AS description,
		       bob.driving_node_id, tbl.node_name AS table_name,
		       COALESCE(tbl.properties->>'database_name', 'ANALYTICS_DB') AS database_name,
		       COALESCE(tbl.properties->>'schema_name', 'PUBLIC') AS schema_name
		FROM public.business_object bo
		JOIN public.business_object_binding bob ON bo.id = bob.bo_id
		JOIN public.catalog_node tbl ON bob.driving_node_id = tbl.node_id
		WHERE (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bo.is_active = TRUE AND bob.is_default = TRUE;`

	type BORow struct {
		ID            uuid.UUID `db:"id"`
		BOKey         string    `db:"bo_key"`
		BOName        string    `db:"bo_name"`
		Description   string    `db:"description"`
		DrivingNodeID uuid.UUID `db:"driving_node_id"`
		TableName     string    `db:"table_name"`
		DatabaseName  string    `db:"database_name"`
		SchemaName    string    `db:"schema_name"`
	}

	var boRows []BORow
	if e.db != nil {
		if err := e.db.SelectContext(ctx, &boRows, boQuery, tenantID); err != nil {
			return nil, fmt.Errorf("failed fetching business objects: %w", err)
		}
	}

	modelSpec := CortexModelSpec{
		Name:            fmt.Sprintf("uisce_semantic_catalog_%s", strings.ReplaceAll(tenantID.String(), "-", "")[:8]),
		Description:     "Multi-tenant governed semantic catalog compiled directly from Uisce Semantic OS",
		Tables:          make([]CortexTableSpec, 0),
		Relationships:   make([]CortexRelationship, 0),
		VerifiedQueries: make([]CortexVerifiedQuery, 0),
	}

	for _, bo := range boRows {
		tableSpec := CortexTableSpec{
			Name: bo.BOKey,
			BaseTable: CortexBaseTable{
				Database: bo.DatabaseName,
				Schema:   bo.SchemaName,
				Table:    bo.TableName,
			},
			Dimensions:     make([]CortexDimension, 0),
			TimeDimensions: make([]CortexTimeDim, 0),
			Measures:       make([]CortexMeasure, 0),
		}

		// 2. Fetch fields, semantic terms, transformations, and synonyms
		fieldQuery := `
			SELECT bof.field_name, bof.field_role,
			       COALESCE(st.node_name, bof.field_name) AS term_name,
			       COALESCE(st.properties->>'data_type', 'VARCHAR') AS data_type,
			       COALESCE(st.properties->>'aggregation_type', 'SUM') AS agg_type,
			       COALESCE(fb.transformation_sql, col.node_name, bof.field_name) AS expr,
			       COALESCE(st.description, '') AS description
			FROM public.business_object_field bof
			LEFT JOIN public.catalog_node st ON bof.term_node_id = st.node_id
			LEFT JOIN public.field_binding fb ON fb.field_id = bof.id AND (fb.tenant_id = $1 OR fb.tenant_id = '00000000-0000-0000-0000-000000000000')
			LEFT JOIN public.catalog_node col ON fb.source_node_id = col.node_id
			WHERE bof.bo_id = $2 AND (bof.tenant_id = $1 OR bof.tenant_id = '00000000-0000-0000-0000-000000000000')
			  AND bof.is_active = TRUE;`

		type FieldRow struct {
			FieldName   string `db:"field_name"`
			FieldRole   string `db:"field_role"`
			TermName    string `db:"term_name"`
			DataType    string `db:"data_type"`
			AggType     string `db:"agg_type"`
			Expr        string `db:"expr"`
			Description string `db:"description"`
		}

		var fRows []FieldRow
		if e.db != nil {
			if err := e.db.SelectContext(ctx, &fRows, fieldQuery, tenantID, bo.ID); err == nil {
				for _, f := range fRows {
					synonyms := []string{f.TermName}
					if !strings.EqualFold(f.FieldName, f.TermName) {
						synonyms = append(synonyms, strings.ReplaceAll(f.FieldName, "_", " "))
					}

					dtUpper := strings.ToUpper(f.DataType)
					roleUpper := strings.ToUpper(f.FieldRole)

					if roleUpper == "MEASURE" {
						tableSpec.Measures = append(tableSpec.Measures, CortexMeasure{
							Name:               f.FieldName,
							Expr:               f.Expr,
							DataType:           f.DataType,
							DefaultAggregation: f.AggType,
							Description:        f.Description,
							Synonyms:           synonyms,
						})
					} else if strings.Contains(dtUpper, "DATE") || strings.Contains(dtUpper, "TIME") {
						tableSpec.TimeDimensions = append(tableSpec.TimeDimensions, CortexTimeDim{
							Name:     f.FieldName,
							Expr:     f.Expr,
							DataType: f.DataType,
							Synonyms: synonyms,
						})
					} else {
						tableSpec.Dimensions = append(tableSpec.Dimensions, CortexDimension{
							Name:        f.FieldName,
							Expr:        f.Expr,
							DataType:    f.DataType,
							Description: f.Description,
							Synonyms:    synonyms,
						})
					}
				}
			}
		}

		modelSpec.Tables = append(modelSpec.Tables, tableSpec)

		// 3. Fetch relationships
		relQuery := `
			SELECT bor.rel_key, target_bo.bo_key AS target_table, rb.join_condition_sql
			FROM public.business_object_relationship bor
			JOIN public.business_object target_bo ON bor.to_bo_id = target_bo.id
			JOIN public.relationship_binding rb ON rb.rel_id = bor.id AND (rb.tenant_id = $1 OR rb.tenant_id = '00000000-0000-0000-0000-000000000000')
			WHERE bor.from_bo_id = $2;`

		type RelRow struct {
			RelKey      string `db:"rel_key"`
			TargetTable string `db:"target_table"`
			JoinSQL     string `db:"join_condition_sql"`
		}

		var relRows []RelRow
		if e.db != nil {
			if err := e.db.SelectContext(ctx, &relRows, relQuery, tenantID, bo.ID); err == nil {
				for _, r := range relRows {
					modelSpec.Relationships = append(modelSpec.Relationships, CortexRelationship{
						Name:             r.RelKey,
						LeftTable:        bo.BOKey,
						RightTable:       r.TargetTable,
						RelationshipType: "many_to_one",
						JoinCondition:    r.JoinSQL,
					})
				}
			}
		}
	}

	// 4. Inject Verified Historical Queries from conversational query sessions & semantic query templates
	if e.db != nil {
		vqQuery := `
			SELECT query_name, user_question, generated_sql, verified_by
			FROM (
				SELECT COALESCE(title, 'Verified Analytical Query') AS query_name,
				       COALESCE(prompt_text, user_intent, '') AS user_question,
				       generated_sql,
				       COALESCE(verified_by, 'GOVERNANCE_CHAMBER') AS verified_by
				FROM catalog_ai.conversational_query_sessions
				WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
				  AND is_verified = TRUE AND generated_sql IS NOT NULL AND generated_sql != ''
				UNION ALL
				SELECT template_name AS query_name,
				       description AS user_question,
				       sql_query AS generated_sql,
				       'SEMANTIC_REGISTRY' AS verified_by
				FROM public.semantic_query_templates
				WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
				  AND is_active = TRUE AND sql_query IS NOT NULL AND sql_query != ''
			) sub
			LIMIT 25;`

		type VQRow struct {
			QueryName    string `db:"query_name"`
			UserQuestion string `db:"user_question"`
			GeneratedSQL string `db:"generated_sql"`
			VerifiedBy   string `db:"verified_by"`
		}

		var vqRows []VQRow
		if err := e.db.SelectContext(ctx, &vqRows, vqQuery, tenantID); err == nil {
			for _, vq := range vqRows {
				if vq.UserQuestion != "" && vq.GeneratedSQL != "" {
					modelSpec.VerifiedQueries = append(modelSpec.VerifiedQueries, CortexVerifiedQuery{
						Name:                   vq.QueryName,
						Question:               vq.UserQuestion,
						SQL:                    strings.TrimSpace(vq.GeneratedSQL),
						VerifiedBy:             vq.VerifiedBy,
						UseAsOnboardingExample: true,
					})
				}
			}
		}
	}

	return yaml.Marshal(modelSpec)
}

// GenerateSnowflakeGovernanceDDL compiles ABAC classifications and tag metadata into Snowflake Horizon DDL
func (e *CortexExporter) GenerateSnowflakeGovernanceDDL(ctx context.Context, tenantID uuid.UUID) ([]GovernanceTagStatement, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	if e.db == nil {
		return []GovernanceTagStatement{}, nil
	}

	query := `
		SELECT tbl.node_name AS table_name, col.node_name AS column_name,
		       COALESCE(col.properties->>'classification', 'CONFIDENTIAL') AS tag_value
		FROM public.catalog_node col
		JOIN public.catalog_edge e ON col.node_id = e.target_node_id
		JOIN public.catalog_node tbl ON e.source_node_id = tbl.node_id
		WHERE (col.tenant_id = $1 OR col.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND col.node_type = 'ATTRIBUTE' AND col.properties ? 'classification';`

	type TagRow struct {
		TableName  string `db:"table_name"`
		ColumnName string `db:"column_name"`
		TagValue   string `db:"tag_value"`
	}

	var rows []TagRow
	var statements []GovernanceTagStatement

	if err := e.db.SelectContext(ctx, &rows, query, tenantID); err == nil {
		for _, r := range rows {
			ddl := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET TAG GOVERNANCE.DATA_CLASSIFICATION = '%s';",
				r.TableName, r.ColumnName, r.TagValue)
			statements = append(statements, GovernanceTagStatement{
				TableName:  r.TableName,
				ColumnName: r.ColumnName,
				TagKey:     "GOVERNANCE.DATA_CLASSIFICATION",
				TagValue:   r.TagValue,
				SQL:        ddl,
			})
		}
	}

	return statements, nil
}
