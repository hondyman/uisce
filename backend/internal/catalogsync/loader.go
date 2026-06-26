package catalogsync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/catalog"
)

// LoadCatalogJSON reads a catalog.json file from the given path.
func LoadCatalogJSON(path string) (*catalog.Catalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open catalog json: %w", err)
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read catalog json: %w", err)
	}

	var cat catalog.Catalog
	if err := json.Unmarshal(body, &cat); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json: %w", err)
	}

	return &cat, nil
}

// BuildGraphFromCatalogJSON converts a catalog.Catalog into node and edge inputs.
func BuildGraphFromCatalogJSON(cat *catalog.Catalog, typeIDs CDMTypeIDs, tenantID uuid.UUID, tenantDatasourceID *uuid.UUID) ([]NodeInput, []CDMEdgeSpec) {
	nodes := make([]NodeInput, 0)
	edges := make([]CDMEdgeSpec, 0)

	// We use the same namespace pattern as finos.go for consistency
	const namespace = "cdm"

	for _, prod := range cat.Products {
		classPath := fmt.Sprintf("cdm/%s/%s", namespace, prod.ID)

		// Create Class Node
		nodes = append(nodes, NodeInput{
			TypeID:        typeIDs.Class,
			Name:          prod.ID, // e.g. "InterestRatePayout"
			QualifiedPath: classPath,
			Properties: map[string]any{
				"namespace":   namespace,
				"kind":        "payout", // or "product", preserving semantics
				"description": prod.Description,
				"label":       prod.Label,
				"family":      prod.Family,
			},
			Config:             map[string]any{},
			TenantID:           tenantID,
			TenantDatasourceID: tenantDatasourceID,
		})

		// Create Field Nodes and Edges
		for _, attr := range prod.Attributes {
			fieldPath := fmt.Sprintf("%s/field/%s", classPath, attr.Name)

			// Append Field Node
			nodes = append(nodes, NodeInput{
				TypeID:        typeIDs.Field,
				Name:          attr.Name,
				QualifiedPath: fieldPath,
				Properties: map[string]any{
					"data_type":   string(attr.Type),
					"cardinality": cardinality(attr.Required),
					"is_enum":     attr.Type == catalog.AttrEnum,
				},
				Config:             map[string]any{},
				TenantID:           tenantID,
				TenantDatasourceID: tenantDatasourceID,
			})

			// Append Edge (Class -> Field)
			edges = append(edges, CDMEdgeSpec{
				Type:       "HAS_FIELD",
				SourceType: typeIDs.Class,
				SourcePath: classPath,
				TargetType: typeIDs.Field,
				TargetPath: fieldPath,
				Props: map[string]any{
					"cardinality": cardinality(attr.Required),
					"data_type":   string(attr.Type),
				},
			})
		}
	}

	return nodes, edges
}

func cardinality(required bool) string {
	if required {
		return "1"
	}
	return "0..1"
}
