package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/cubeengine"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// Local types for diffing and patching, to decouple from the reporting-focused models.Change
type patchChangeKind string

const (
	patchAdd       patchChangeKind = "add"
	patchModify    patchChangeKind = "modify"
	patchDeprecate patchChangeKind = "deprecate"
)

// SeverityThresholds maps severity names to integer levels for comparison.
var SeverityThresholds = map[string]int{
	"none":     4,
	"low":      3,
	"medium":   2,
	"breaking": 1,
}

type patchChange struct {
	Kind   patchChangeKind
	Path   []string
	Before any
	After  any
	Reason string
}

type patchDiff struct {
	Changes []patchChange
}

// RemovalPolicy defines how to handle removed items.
type RemovalPolicy int

const (
	DeprecateOnly RemovalPolicy = iota
	HardDelete
)

// UpgradeService provides methods for upgrading semantic models.
type UpgradeService struct {
	DB *sqlx.DB
}

// NewUpgradeService creates a new UpgradeService.
func NewUpgradeService(db *sqlx.DB) *UpgradeService {
	return &UpgradeService{DB: db}
}

// UpgradeCoreModel orchestrates the discovery, diffing, and patching of a core model.
func (s *UpgradeService) UpgradeCoreModel(ctx context.Context, dsn, driver string, schemas []string, currentDef models.FabricDefn, policy RemovalPolicy, dryRun bool, actorID uuid.UUID) (*patchDiff, *models.FabricDefn, error) {
	sourceDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open source DB connection: %w", err)
	}
	defer sourceDB.Close()

	if err := sourceDB.PingContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to ping source DB: %w", err)
	}

	// 1. Discover the current state of the database.
	catalog, err := s.DiscoverCatalog(ctx, sourceDB, schemas)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover catalog: %w", err)
	}

	// 2. Generate the desired "cubes" from the discovered catalog.
	desiredCubes := CatalogToCubes(catalog, []string{"core"})
	desiredConfig := models.ResolvedModelConfig{
		ModelKey: currentDef.ModelKey,
		Cubes:    desiredCubes,
	}

	// 3. Diff the current model against the desired state.
	var currentConfig models.ResolvedModelConfig
	if err := json.Unmarshal(currentDef.ResolvedConfig, &currentConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal current resolved config: %w", err)
	}
	diff := DiffModels(currentConfig, desiredConfig)

	if dryRun {
		return &diff, nil, nil
	}

	// 4. Apply the patch to create the new configuration.
	patchedConfig := ApplyPatch(currentConfig, diff, policy)

	// 5. Create and persist the new draft version.
	nextDef := currentDef
	nextDef.ID = uuid.New() // New version gets a new ID
	nextDef.Version++
	nextDef.Status = models.StatusDraft
	nextDef.IsCurrent = false
	nextDef.CreatedBy = actorID

	resolvedJSON, err := json.Marshal(patchedConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal patched config: %w", err)
	}
	nextDef.ResolvedConfig = models.JSONB(resolvedJSON)

	// Insert the new version into the database (transaction recommended).
	// You would also write to your audit table here.
	query := `
		INSERT INTO public.fabric_defn (id, tenant_id, tenant_datasource_id, model_key, version, status, is_current, title, description, source_config, resolved_config, created_by)
		VALUES (:id, :tenant_id, :tenant_datasource_id, :model_key, :version, :status, :is_current, :title, :description, :source_config, :resolved_config, :created_by)
	`
	_, err = s.DB.NamedExecContext(ctx, query, &nextDef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert new model version: %w", err)
	}

	return &diff, &nextDef, nil
}

// CompareModels generates a detailed, annotated diff between two model configurations,
// suitable for the live Upgrade Compare UI.
func (s *UpgradeService) CompareModels(before, after models.ResolvedModelConfig) ([]models.ModelDiff, error) {
	diff := DiffModels(before, after)
	changesByModel := make(map[string][]models.FieldChange)
	modelChangeTypes := make(map[string]string)

	for _, change := range diff.Changes {
		if len(change.Path) < 2 {
			continue // Not a model-level change
		}

		modelName := change.Path[1]
		if modelName == "" {
			continue
		}

		if change.Path[0] == "cube" && len(change.Path) == 2 {
			switch change.Kind {
			case patchAdd:
				modelChangeTypes[modelName] = "added"
			case patchDeprecate:
				modelChangeTypes[modelName] = "removed"
			}
			continue
		}

		if _, exists := modelChangeTypes[modelName]; !exists {
			modelChangeTypes[modelName] = "modified"
		}

		fc := models.FieldChange{
			ID:             "chg-" + uuid.New().String(),
			Path:           strings.Join(change.Path[2:], "."),
			ChangeType:     string(change.Kind),
			SelectionState: "untouched",
		}

		if change.Before != nil {
			beforeBytes, _ := json.Marshal(change.Before)
			fc.Before = string(beforeBytes)
		}
		if change.After != nil {
			afterBytes, _ := json.Marshal(change.After)
			fc.After = string(afterBytes)

			if item, ok := change.After.(map[string]any); ok {
				if meta, metaOk := item["meta"].(map[string]any); metaOk {
					fc.RuleID, _ = meta["rule_id"].(string)
					fc.Provenance, _ = meta["provenance"].(string)
					fc.Reason = fmt.Sprintf("Generated by rule '%s' based on evidence from '%s'.", fc.RuleID, fc.Provenance)
					if fc.RuleID != "" {
						fc.RuleLink = fmt.Sprintf("/models/upgrade/analytics/rule/%s", fc.RuleID)
					}
				}
			}
		}

		changesByModel[modelName] = append(changesByModel[modelName], fc)
	}

	var modelDiffs []models.ModelDiff
	for modelName, fieldChanges := range changesByModel {
		modelDiffs = append(modelDiffs, models.ModelDiff{
			Model:        modelName,
			ChangeType:   modelChangeTypes[modelName],
			FieldChanges: fieldChanges,
		})
	}

	sort.Slice(modelDiffs, func(i, j int) bool {
		return modelDiffs[i].Model < modelDiffs[j].Model
	})

	return modelDiffs, nil
}

// GenerateAnnotatedDiff compares two model configurations and returns a list of semantic annotations.
// This is used by the tuning simulation to explain proposed changes.
func GenerateAnnotatedDiff(before, after models.ResolvedModelConfig) []models.ChangeAnnotation {
	diff := DiffModels(before, after)

	var annotations []models.ChangeAnnotation

	for _, change := range diff.Changes {
		// We only care about additions and modifications for extracting metadata from the 'after' state.
		if change.Kind != patchAdd && change.Kind != patchModify {
			continue
		}

		// The 'After' field holds the new map[string]any for the dimension, measure, or join.
		item, ok := change.After.(map[string]any)
		if !ok {
			continue
		}

		meta, metaOk := item["meta"].(map[string]any)
		if !metaOk {
			continue
		}

		ruleID, _ := meta["rule_id"].(string)
		provenance, _ := meta["provenance"].(string)

		// Construct a reason from the metadata
		reason := fmt.Sprintf("Generated by rule '%s' based on evidence from '%s'.", ruleID, provenance)

		annotations = append(annotations, models.ChangeAnnotation{
			Path:       strings.Join(change.Path, "."),
			ChangeType: string(change.Kind),
			RuleID:     ruleID,
			Provenance: provenance,
			Reason:     reason,
		})
	}

	return annotations
}

// GetChangesForSimulation prepares a list of changes for a "what-if" scenario.
func (s *UpgradeService) GetChangesForSimulation(ctx context.Context, fromDS, toDS, migrationFile string) ([]db.Change, error) {
	if fromDS != "" && toDS != "" {
		logging.GetLogger().Sugar().Infof("Simulating changes between snapshots %s and %s", fromDS, toDS)
		fromUUID, err := uuid.Parse(fromDS)
		if err != nil {
			return nil, fmt.Errorf("invalid from_ds UUID: %w", err)
		}
		toUUID, err := uuid.Parse(toDS)
		if err != nil {
			return nil, fmt.Errorf("invalid to_ds UUID: %w", err)
		}

		oldSnap, err := db.BuildSnapshot(ctx, s.DB, fromUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to build 'from' snapshot: %w", err)
		}
		newSnap, err := db.BuildSnapshot(ctx, s.DB, toUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to build 'to' snapshot: %w", err)
		}
		return db.DiffSnapshots(oldSnap, newSnap), nil
	}

	if migrationFile != "" {
		logging.GetLogger().Sugar().Infof("Simulating changes from migration file: %s", migrationFile)
		// This is where a real SQL parser would be used.
		// For now, we'll return a mock change based on the content.
		if strings.Contains(strings.ToLower(migrationFile), "drop column") {
			return []db.Change{
				{ChangeType: "drop_column", NodeType: "column", QualifiedPath: "public.users.email", Details: "Mock change: dropped column 'email' from 'users'", Severity: db.SeverityBreaking},
			}, nil
		}
		return []db.Change{
			{
				ChangeType:    "add_column",
				NodeType:      "column",
				QualifiedPath: "public.products.new_feature_flag",
				Details:       "Mock change: added column 'new_feature_flag' to 'products'",
				Severity:      db.SeverityLow,
			},
		}, nil
	}

	return nil, fmt.Errorf("simulation requires either from/to snapshots or a migration file")
}

// DiscoverCatalog queries a database's information_schema to build a catalog.
func (s *UpgradeService) DiscoverCatalog(ctx context.Context, db *sql.DB, schemas []string) (*cubeengine.Catalog, error) {
	// This is a simplified discovery. A production implementation would be more robust.
	logging.GetLogger().Sugar().Infof("Discovering catalog for schemas: %v", schemas)
	// In a real implementation, you would query information_schema.tables,
	// information_schema.columns, and information_schema.referential_constraints.
	// For this example, we'll return a placeholder.
	return &cubeengine.Catalog{Tables: []cubeengine.Table{}}, nil
}

// --- Diffing Logic ---

// DiffModels compares two model configurations and returns a set of changes.
func DiffModels(current, desired models.ResolvedModelConfig) patchDiff {
	curIdx := indexCubes(current.Cubes)
	desIdx := indexCubes(desired.Cubes)
	var changes []patchChange

	// Check for added and modified cubes
	for name, dc := range desIdx {
		if cc, ok := curIdx[name]; !ok {
			changes = append(changes, patchChange{Kind: patchAdd, Path: []string{"cube", name}, After: dc})
		} else {
			changes = append(changes, diffCube(name, cc, dc)...)
		}
	}
	// Check for removed cubes
	for name, cc := range curIdx {
		if _, ok := desIdx[name]; !ok {
			changes = append(changes, patchChange{Kind: patchDeprecate, Path: []string{"cube", name}, Before: cc, Reason: "Table not found in catalog"})
		}
	}
	return patchDiff{Changes: changes}
}

func diffCube(name string, cur, des cube.Cube) []patchChange {
	var out []patchChange
	// Diff dimensions
	out = append(out, diffMap(cur.Dimensions, des.Dimensions, []string{"cube", name, "dimensions"})...)
	// Diff measures
	out = append(out, diffMap(cur.Measures, des.Measures, []string{"cube", name, "measures"})...)
	// Diff joins
	out = append(out, diffMap(cur.Joins, des.Joins, []string{"cube", name, "joins"})...)
	return out
}

func diffMap(cur, des map[string]map[string]any, pathPrefix []string) []patchChange {
	var changes []patchChange
	for k, v := range des {
		if cv, ok := cur[k]; !ok {
			changes = append(changes, patchChange{Kind: patchAdd, Path: append(pathPrefix, k), After: v})
		} else if !reflect.DeepEqual(cv, v) {
			changes = append(changes, patchChange{Kind: patchModify, Path: append(pathPrefix, k), Before: cv, After: v})
		}
	}
	for k, cv := range cur {
		if _, ok := des[k]; !ok {
			changes = append(changes, patchChange{Kind: patchDeprecate, Path: append(pathPrefix, k), Before: cv, Reason: "Item not found in source"})
		}
	}
	return changes
}

// --- Patching Logic ---

// ApplyPatch applies a diff to a base configuration to produce a new one.
func ApplyPatch(base models.ResolvedModelConfig, diff patchDiff, policy RemovalPolicy) models.ResolvedModelConfig {
	next := base
	cubes := indexCubes(next.Cubes)

	for _, ch := range diff.Changes {
		switch ch.Kind {
		case patchAdd:
			if len(ch.Path) == 2 && ch.Path[0] == "cube" {
				cubes[ch.Path[1]] = ch.After.(cube.Cube)
			} else {
				patchInMap(cubes, ch.Path, ch.After)
			}
		case patchModify:
			patchInMap(cubes, ch.Path, ch.After)
		case patchDeprecate:
			if policy == DeprecateOnly {
				addTombstone(cubes, ch.Path)
			} else { // HardDelete
				deleteAtPath(cubes, ch.Path)
			}
		}
	}

	// Rebuild the slice of cubes from the map to maintain order.
	var updatedCubes []cube.Cube
	for _, c := range base.Cubes {
		if updatedCube, exists := cubes[c.Name]; exists {
			updatedCubes = append(updatedCubes, updatedCube)
			delete(cubes, c.Name) // Remove from map to handle newly added cubes
		}
	}
	// Add any new cubes that were not in the original order
	for _, newCube := range cubes {
		updatedCubes = append(updatedCubes, newCube)
	}
	sort.Slice(updatedCubes, func(i, j int) bool { return updatedCubes[i].Name < updatedCubes[j].Name })

	next.Cubes = updatedCubes
	return next
}

func patchInMap(cubes map[string]cube.Cube, path []string, value any) {
	if len(path) < 3 {
		return
	}
	cubeName, prop, key := path[1], path[2], path[3]
	if cube, ok := cubes[cubeName]; ok {
		switch prop {
		case "dimensions":
			if cube.Dimensions == nil {
				cube.Dimensions = make(map[string]map[string]any)
			}
			cube.Dimensions[key] = value.(map[string]any)
		case "measures":
			if cube.Measures == nil {
				cube.Measures = make(map[string]map[string]any)
			}
			cube.Measures[key] = value.(map[string]any)
		case "joins":
			if cube.Joins == nil {
				cube.Joins = make(map[string]map[string]any)
			}
			cube.Joins[key] = value.(map[string]any)
		}
		cubes[cubeName] = cube
	}
}

func deleteAtPath(cubes map[string]cube.Cube, path []string) {
	if len(path) < 2 {
		return
	}
	if len(path) == 2 && path[0] == "cube" {
		delete(cubes, path[1])
		return
	}
	if len(path) == 4 {
		cubeName, prop, key := path[1], path[2], path[3]
		if cube, ok := cubes[cubeName]; ok {
			switch prop {
			case "dimensions":
				delete(cube.Dimensions, key)
			case "measures":
				delete(cube.Measures, key)
			case "joins":
				delete(cube.Joins, key)
			}
			cubes[cubeName] = cube
		}
	}
}

func addTombstone(cubes map[string]cube.Cube, path []string) {
	if len(path) < 2 {
		return
	}
	cubeName := path[1]
	if cube, ok := cubes[cubeName]; ok {
		if cube.Metadata == nil {
			cube.Metadata = make(map[string]any)
		}
		if len(path) == 2 { // Deprecating a whole cube
			cube.Metadata["deprecated"] = true
			cube.Metadata["deprecated_reason"] = "Table not found in source"
		} else if len(path) == 4 { // Deprecating a sub-item
			prop, key := path[2], path[3]
			if cube.Metadata[prop] == nil {
				cube.Metadata[prop] = make(map[string]any)
			}
			if metaProp, ok := cube.Metadata[prop].(map[string]any); ok {
				metaProp[key] = map[string]any{"deprecated": true, "reason": "Item not found in source"}
			}
		}
		cubes[cubeName] = cube
	}
}

// --- Helper Functions ---

func indexCubes(cubes []cube.Cube) map[string]cube.Cube {
	idx := make(map[string]cube.Cube)
	for _, c := range cubes {
		idx[c.Name] = c
	}
	return idx
}

func inferDimType(dt string) string {
	s := strings.ToLower(dt)
	switch {
	case strings.Contains(s, "int"), strings.Contains(s, "decimal"), strings.Contains(s, "numeric"), strings.Contains(s, "float"):
		return "number"
	case strings.Contains(s, "date"), strings.Contains(s, "time"):
		return "time"
	default:
		return "string"
	}
}

func CatalogToCubes(cat *cubeengine.Catalog, tags []string) []cube.Cube {
	var cubes []cube.Cube
	for _, tbl := range cat.Tables {
		c := cube.Cube{
			Name:       fmt.Sprintf("%s_%s", tbl.Schema, tbl.Name),
			Tags:       append([]string{}, tags...),
			SQL:        fmt.Sprintf("SELECT * FROM %s.%s", tbl.Schema, tbl.Name),
			SQLTable:   fmt.Sprintf("%s.%s", tbl.Schema, tbl.Name),
			Measures:   map[string]map[string]any{"count": {"type": "count"}},
			Dimensions: map[string]map[string]any{},
			Joins:      map[string]map[string]any{},
			Metadata:   map[string]any{"read_only": true},
		}
		// Provide a human-readable title/description where sensible
		tc := cases.Title(language.Und)
		c.Title = fmt.Sprintf("%s %s", tc.String(strings.ReplaceAll(tbl.Schema, "_", " ")), tc.String(strings.ReplaceAll(tbl.Name, "_", " ")))
		c.Description = fmt.Sprintf("Auto-generated cube for table %s.%s", tbl.Schema, tbl.Name)
		public := true
		c.Public = &public
		for _, col := range tbl.Columns {
			c.Dimensions[col.Name] = map[string]any{
				"sql":  col.Name,
				"type": inferDimType(col.DataType),
				"meta": models.ExplainMeta("auto_dimension", tbl.Name, col.Name),
			}
		}
		for _, fk := range tbl.FKs {
			if len(fk.FromCols) == 1 && len(fk.ToCols) == 1 {
				target := fmt.Sprintf("%s_%s", fk.ToSchema, fk.ToTable)
				c.Joins[target] = map[string]any{
					"relationship": "many_to_one",
					"sql":          fmt.Sprintf("${CUBE}.%s = ${%s}.%s", fk.FromCols[0], target, fk.ToCols[0]),
					"meta":         models.ExplainMeta("auto_join_fk", tbl.Name, strings.Join(fk.FromCols, ",")),
				}
			}
		}
		cubes = append(cubes, c)
	}
	sort.Slice(cubes, func(i, j int) bool { return cubes[i].Name < cubes[j].Name })
	return cubes
}
