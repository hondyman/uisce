package semantic_bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type DatabricksGenieModel struct {
	Version      string                `json:"version"`
	SpaceName    string                `json:"space_name"`
	Tables       []GenieTableSpec      `json:"tables"`
	Joins        []GenieJoinSpec       `json:"joins"`
	Calculations []GenieCalculationDef `json:"calculations"`
	Benchmarks   []GenieBenchmarkQuery `json:"benchmarks,omitempty"`
}

type GenieBenchmarkQuery struct {
	Name         string `json:"name"`
	UserQuestion string `json:"user_question"`
	ExpectedSQL  string `json:"expected_sql"`
	VerifiedBy   string `json:"verified_by"`
}

type GenieTableSpec struct {
	TableName   string        `json:"table_name"`
	Description string        `json:"description"`
	Columns     []GenieColumn `json:"columns"`
}

type GenieColumn struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DataType    string   `json:"data_type"`
	Synonyms    []string `json:"synonyms"`
	IsMeasure   bool     `json:"is_measure"`
}

type GenieJoinSpec struct {
	LeftTable  string `json:"left_table"`
	RightTable string `json:"right_table"`
	Condition  string `json:"condition"`
}

type GenieCalculationDef struct {
	Name        string `json:"name"`
	Formula     string `json:"formula"`
	Description string `json:"description"`
}

type DatabricksExporter struct {
	db *sqlx.DB
}

func NewDatabricksExporter(db *sqlx.DB) *DatabricksExporter {
	return &DatabricksExporter{db: db}
}

// CompileGenieModel compiles the Uisce BO catalog into Databricks Genie Space definitions
func (e *DatabricksExporter) CompileGenieModel(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	cortexExp := NewCortexExporter(e.db)
	yamlBytes, err := cortexExp.CompileFullCortexModel(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var cortex CortexModelSpec
	if err := yaml.Unmarshal(yamlBytes, &cortex); err != nil {
		return nil, err
	}

	genie := DatabricksGenieModel{
		Version:      "1.0",
		SpaceName:    cortex.Name,
		Tables:       make([]GenieTableSpec, 0),
		Joins:        make([]GenieJoinSpec, 0),
		Calculations: make([]GenieCalculationDef, 0),
		Benchmarks:   make([]GenieBenchmarkQuery, 0),
	}

	for _, tbl := range cortex.Tables {
		gt := GenieTableSpec{
			TableName:   tbl.Name,
			Description: fmt.Sprintf("Source table: %s.%s.%s", tbl.BaseTable.Database, tbl.BaseTable.Schema, tbl.BaseTable.Table),
			Columns:     make([]GenieColumn, 0),
		}

		for _, dim := range tbl.Dimensions {
			gt.Columns = append(gt.Columns, GenieColumn{
				Name:        dim.Name,
				Description: dim.Description,
				DataType:    dim.DataType,
				Synonyms:    dim.Synonyms,
				IsMeasure:   false,
			})
		}

		for _, m := range tbl.Measures {
			gt.Columns = append(gt.Columns, GenieColumn{
				Name:        m.Name,
				Description: m.Description,
				DataType:    m.DataType,
				Synonyms:    m.Synonyms,
				IsMeasure:   true,
			})
			genie.Calculations = append(genie.Calculations, GenieCalculationDef{
				Name:        m.Name,
				Formula:     fmt.Sprintf("%s(%s)", m.DefaultAggregation, m.Expr),
				Description: m.Description,
			})
		}

		genie.Tables = append(genie.Tables, gt)
	}

	for _, r := range cortex.Relationships {
		genie.Joins = append(genie.Joins, GenieJoinSpec{
			LeftTable:  r.LeftTable,
			RightTable: r.RightTable,
			Condition:  r.JoinCondition,
		})
	}

	for _, vq := range cortex.VerifiedQueries {
		genie.Benchmarks = append(genie.Benchmarks, GenieBenchmarkQuery{
			Name:         vq.Name,
			UserQuestion: vq.Question,
			ExpectedSQL:  vq.SQL,
			VerifiedBy:   vq.VerifiedBy,
		})
	}

	return json.MarshalIndent(genie, "", "  ")
}

// GenerateUnityCatalogSQL generates ANSI SQL comments & tags to enrich Unity Catalog directly
func (e *DatabricksExporter) GenerateUnityCatalogSQL(ctx context.Context, tenantID uuid.UUID) (string, error) {
	genieBytes, err := e.CompileGenieModel(ctx, tenantID)
	if err != nil {
		return "", err
	}

	var model DatabricksGenieModel
	_ = json.Unmarshal(genieBytes, &model)

	var sb strings.Builder
	sb.WriteString("-- Auto-generated Unity Catalog Metadata & Governance by Uisce Semantic OS\n\n")

	for _, tbl := range model.Tables {
		sb.WriteString(fmt.Sprintf("COMMENT ON TABLE %s IS '%s';\n", tbl.TableName, strings.ReplaceAll(tbl.Description, "'", "''")))
		for _, col := range tbl.Columns {
			comment := col.Description
			if len(col.Synonyms) > 0 {
				comment += fmt.Sprintf(" (Synonyms: %s)", strings.Join(col.Synonyms, ", "))
			}
			sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s COMMENT '%s';\n", tbl.TableName, col.Name, strings.ReplaceAll(comment, "'", "''")))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
