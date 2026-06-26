package models

import (
	"bytes"
	"compress/gzip"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenerateModelsRequest defines the request body for the model generation endpoint.
type GenerateModelsRequest struct {
	DatasourceID        string                 `json:"datasource_id"`
	Scope               map[string]interface{} `json:"scope"`
	Overwrite           bool                   `json:"overwrite"`
	AcceptRelationships bool                   `json:"accept_relationships"`
}

// SemanticMember represents a dimension or measure in the generated model.
type SemanticMember struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Type        string `json:"type"` // e.g., "string", "number", "time"
	Description string `json:"description,omitempty"`
	SQL         string `json:"sql"`
}

// CubeJoin represents a join in a Cube.dev model, for API responses.
type CubeJoin struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	SQL          string `json:"sql"`
}

// SemanticModel represents a single generated semantic model.
type SemanticModel struct {
	TableName   string           `json:"table_name"`
	SqlTable    string           `json:"sql_table"`
	ModelName   string           `json:"model_name,omitempty"`
	Description string           `json:"description"`
	Dimensions  []SemanticMember `json:"dimensions"`
	Measures    []SemanticMember `json:"measures"`
	Joins       []CubeJoin       `json:"joins"`
}

// JoinSuggestion represents a potential relationship between two tables.
type JoinSuggestion struct {
	FromTable     string `json:"from_table"`
	FromColumn    string `json:"from_column"`
	FromCol       string `json:"from_col"`
	ToTable       string `json:"to_table"`
	ToColumn      string `json:"to_column"`
	ToCol         string `json:"to_col"`
	JoinType      string `json:"join_type"`
	Relationship  string `json:"relationship"`
	Cardinality   string `json:"cardinality"`
	Referential   string `json:"referential_integrity"`
	Security      string `json:"security"`
	JoinCondition string `json:"join_condition"`
	Source        string `json:"source"`
}

// ModelMetadata holds metadata about a single table.
type ModelMetadata struct {
	Exists    bool       `json:"exists"`
	TableName string     `json:"table_name" db:"table_name"`
	Title     string     `json:"title" db:"title"`
	ModelKey  string     `json:"model_key"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// FabricDefn represents a semantic model definition.
type FabricDefn struct {
	ID                 uuid.UUID  `db:"id" json:"id"`
	TenantID           uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	TenantDatasourceID uuid.UUID  `db:"tenant_datasource_id" json:"tenant_datasource_id"`
	ModelKey           string     `db:"model_key" json:"model_key"`
	Version            int        `db:"version" json:"version"`
	Status             string     `db:"status" json:"status"`
	Title              string     `db:"title" json:"title"`
	Description        string     `db:"description" json:"description"`
	SourceConfig       JSONB      `db:"source_config" json:"source_config"`
	ResolvedConfig     JSONB      `db:"resolved_config" json:"resolved_config"`
	CreatedBy          uuid.UUID  `db:"created_by" json:"created_by"`
	IsCurrent          bool       `db:"is_current" json:"is_current"`
	PublishedAt        *time.Time `db:"published_at" json:"published_at,omitempty"`
	CreatedAt          *time.Time `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt          *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	ChecksumSha256     []byte     `db:"checksum_sha256" json:"checksum_sha256,omitempty"`
}

// JSONB represents a PostgreSQL JSONB column and marshals to raw JSON instead of base64.
type JSONB []byte

// MarshalJSON ensures the underlying JSON is emitted verbatim.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	trim := bytes.TrimSpace(j)
	if len(trim) == 0 {
		return []byte("null"), nil
	}
	// If looks like a JSON object/array, return directly
	if trim[0] == '{' || trim[0] == '[' {
		return trim, nil
	}
	// If looks like a JSON string that might hold base64 or gzip+base64
	if trim[0] == '"' && trim[len(trim)-1] == '"' {
		// Remove surrounding quotes
		inner := trim[1 : len(trim)-1]
		// Attempt base64 decode
		if dec, err := base64.StdEncoding.DecodeString(string(inner)); err == nil && len(dec) > 1 {
			// If decoded looks like gzip, decompress
			if dec[0] == 0x1f && dec[1] == 0x8b {
				if gr, err2 := gzip.NewReader(bytes.NewReader(dec)); err2 == nil {
					decomp, derr := io.ReadAll(gr)
					gr.Close()
					if derr == nil && len(decomp) > 0 && (decomp[0] == '{' || decomp[0] == '[') {
						return decomp, nil
					}
					if derr != nil {
						log.Printf("JSONB: gzip decompression failed inside MarshalJSON: %v", derr)
					}
				} else {
					log.Printf("JSONB: gzip reader init failed inside MarshalJSON: %v", err2)
				}
			}
			// Not gzip, maybe raw JSON
			if len(dec) > 0 && (dec[0] == '{' || dec[0] == '[') {
				return dec, nil
			}
		}
		// Fall through: return original string (still quoted) so encoding/json doesn't add extra quotes
		return trim, nil
	}
	// Last resort, return bytes as-is (may already be valid JSON literal like "null" or a number)
	return trim, nil
}

// UnmarshalJSON stores raw JSON.
func (j *JSONB) UnmarshalJSON(b []byte) error {
	if j == nil {
		return fmt.Errorf("jsonb: cannot unmarshal into nil pointer")
	}
	*j = append((*j)[0:0], b...)
	return nil
}

// Value implements driver.Valuer.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// Scan implements sql.Scanner.
func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = JSONB([]byte("null"))
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("jsonb: unsupported Scan type %T", src)
	}
	// Detect gzip magic number and decompress if needed
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		gr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			log.Printf("JSONB: gzip reader init failed inside Scan: %v", err)
		} else {
			decompressed, derr := io.ReadAll(gr)
			gr.Close()
			if derr != nil {
				log.Printf("JSONB: gzip decompression failed inside Scan: %v", derr)
			} else {
				// Replace raw with decompressed JSON bytes
				raw = decompressed
			}
		}
	}
	*j = append((*j)[0:0], raw...)
	return nil
}

// MustJSONB marshals a value to JSONB, panicking on error (usage in initialization only).
func MustJSONB(v interface{}) JSONB {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("MustJSONB marshal failed: %v", err))
	}
	return JSONB(b)
}

// Request is an alias for the unified request struct from the models package.
type Request = GenerateModelsRequest

// Response defines the structured output of a generation job.
type Response struct {
	Generated   []SemanticModel `json:"generated"`
	Skipped     []string        `json:"skipped"`
	Overwritten []string        `json:"overwritten"`
}

// Services defines the dependencies required by the generation logic.
type Services struct {
	SemanticModelService interface {
		GetModelMetadata(uuid.UUID, []string) (map[string]ModelMetadata, error)
		DeleteModels(uuid.UUID, []string) error
		GenerateModels(uuid.UUID, map[string]interface{}) ([]*FabricDefn, error)
		SuggestJoinsFromChart(uuid.UUID, []string) ([]JoinSuggestion, error)
		GetModelDefinition(uuid.UUID, string) (*FabricDefn, error)
	}
}

// Generate orchestrates the entire model generation process.
func Generate(svcs Services, req Request) (*Response, error) {
	dsID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid datasource_id: %w", err)
	}

	scopeTables := extractScopeTables(req.Scope)

	metaMap, err := svcs.SemanticModelService.GetModelMetadata(dsID, scopeTables)
	if err != nil {
		return nil, fmt.Errorf("failed to get model metadata: %w", err)
	}
	toGenerate, skipped, overwritten := partitionByExistence(metaMap, scopeTables, req.Overwrite)

	if req.Overwrite && len(overwritten) > 0 {
		if err := svcs.SemanticModelService.DeleteModels(dsID, overwritten); err != nil {
			return nil, fmt.Errorf("failed to delete existing models: %w", err)
		}
		toGenerate = append(toGenerate, overwritten...)
	}

	var generatedDefns []*FabricDefn
	if len(toGenerate) > 0 {
		gen, err := svcs.SemanticModelService.GenerateModels(dsID, map[string]interface{}{"type": "tables", "names": toGenerate})
		if err != nil {
			return nil, fmt.Errorf("model generation failed: %w", err)
		}
		generatedDefns = gen
	}

	semanticModels := fabricDefnsToSemanticModels(generatedDefns)

	modelMap := make(map[string]SemanticModel)
	for _, m := range semanticModels {
		modelMap[m.TableName] = m
	}

	// Always attempt to suggest joins from the chart and attach them to the
	// generated semantic models when possible. This ensures relationships
	// appear even if the request omitted AcceptRelationships.
	joins, err := svcs.SemanticModelService.SuggestJoinsFromChart(dsID, scopeTables)
	if err != nil {
		log.Printf("⚠️ Could not suggest joins: %v", err)
	} else {
		// Debug: log a small summary/sample of returned joins to help diagnose
		// why relationships may not be attached. Limit samples to the first 5.
		var samples []string
		for i, sj := range joins {
			if i >= 5 {
				break
			}
			samples = append(samples, fmt.Sprintf("%s -> %s (%s.%s = %s.%s)", sj.FromTable, sj.ToTable, sj.FromTable, sj.FromCol, sj.ToTable, sj.ToCol))
		}
		log.Printf("DEBUG: SuggestJoinsFromChart returned %d joins; samples=[%s]", len(joins), strings.Join(samples, ", "))

		// Normalize join table paths to dotted form before attachment.
		for i, sm := range semanticModels {
			// Create a map of existing joins for quick lookup to avoid duplicates
			existingJoins := make(map[string]bool)
			for _, j := range sm.Joins {
				existingJoins[j.Name] = true
			}

			for _, j := range joins {
				fromTable := j.FromTable
				toTable := j.ToTable

				// If the current semantic model is the "from" side of the FK (the "many" side)
				if sm.TableName == fromTable {
					joinName := strings.Split(toTable, ".")[1]
					if _, exists := existingJoins[joinName]; !exists {
						sql := fmt.Sprintf("${CUBE}.%s = ${%s}.%s", j.FromCol, joinName, j.ToCol)
						cubeJoin := CubeJoin{
							Name:         joinName,
							Relationship: "many_to_one",
							SQL:          sql,
						}
						semanticModels[i].Joins = append(semanticModels[i].Joins, cubeJoin)
						existingJoins[joinName] = true
					}
				}

				// If the current semantic model is the "to" side of the FK (the "one" side)
				if sm.TableName == toTable {
					joinName := strings.Split(fromTable, ".")[1]
					if _, exists := existingJoins[joinName]; !exists {
						sql := fmt.Sprintf("${CUBE}.%s = ${%s}.%s", j.ToCol, joinName, j.FromCol)
						cubeJoin := CubeJoin{
							Name:         joinName,
							Relationship: "one_to_many",
							SQL:          sql,
						}
						semanticModels[i].Joins = append(semanticModels[i].Joins, cubeJoin)
						existingJoins[joinName] = true
					}
				}
			}
		}
	}

	resp := NewResponse()
	resp.Generated = semanticModels
	resp.Skipped = skipped
	resp.Overwritten = overwritten

	// Ensure slices are non-nil to prevent 'null' in JSON response.
	// A nil slice in Go marshals to 'null' in JSON, which can cause
	// frontend errors. An empty slice marshals to '[]'.
	if resp.Skipped == nil {
		resp.Skipped = make([]string, 0)
	}
	if resp.Overwritten == nil {
		resp.Overwritten = make([]string, 0)
	}

	return resp, nil
}

// CatalogNodeProperties represents properties of a catalog node.
type CatalogNodeProperties struct {
	DataType string `json:"data_type"`
}

// NewResponse creates a response with initialized slices to avoid nulls in JSON.
func NewResponse() *Response {
	return &Response{
		Generated:   make([]SemanticModel, 0),
		Skipped:     make([]string, 0),
		Overwritten: make([]string, 0),
	}
}

// ScanProgress represents a progress update for a scan operation.
type ScanProgress struct {
	Phase       string  `json:"phase"`
	Percent     float64 `json:"percent"`
	CurrentItem string  `json:"current_item"`
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	Message     string  `json:"message"`
}
