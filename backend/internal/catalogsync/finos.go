package catalogsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	cdmSchemaListURL = "https://api.github.com/repos/finos/common-domain-model/contents/cdm-json-schema"
	cdmRepoURL       = "https://github.com/finos/common-domain-model.git"
	cdmSchemaRelPath = "cdm-json-schema"
)

// CDMClass models a FINOS CDM class in a neutral form.
type CDMClass struct {
	Name         string
	Namespace    string
	Kind         string
	SuperTypes   []string
	Fields       []CDMField
	Enums        []CDMEnum
	Associations []CDMAssociation
}

// CDMField represents a primitive field on a CDM class.
type CDMField struct {
	Name        string
	Type        string
	Cardinality string
	IsEnum      bool
}

// CDMEnum captures an inline enum and its values.
type CDMEnum struct {
	Name   string
	Values []string
}

// CDMAssociation represents a reference to another class.
type CDMAssociation struct {
	Name            string
	TargetType      string
	Cardinality     string
	AssociationRole string
}

// CDMTypeIDs holds node type IDs needed for graph construction.
type CDMTypeIDs struct {
	Class     uuid.UUID
	Field     uuid.UUID
	Enum      uuid.UUID
	EnumValue uuid.UUID
}

// CDMEdgeSpec is a name-based edge description before ID resolution.
type CDMEdgeSpec struct {
	Type       string
	SourceType uuid.UUID
	SourcePath string
	TargetType uuid.UUID
	TargetPath string
	Props      map[string]any
}

type ghContentEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

type cdmSchema struct {
	Title      string                       `json:"title"`
	Properties map[string]json.RawMessage   `json:"properties"`
	AllOf      []map[string]json.RawMessage `json:"allOf"`
	Required   []string                     `json:"required"`
}

type propertySchema struct {
	Type   string          `json:"type"`
	Format string          `json:"format"`
	Ref    string          `json:"$ref"`
	Items  *propertySchema `json:"items"`
	Enum   []string        `json:"enum"`
}

// FetchAndParseCDM clones/pulls the CDM repo and parses all *.schema.json files locally.
func FetchAndParseCDM(localRepoDir string) ([]CDMClass, error) {
	schemaRoot, err := ensureCDMRepo(localRepoDir)
	if err != nil {
		return nil, err
	}

	classes := []CDMClass{}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if filepath.Ext(path) != ".json" {
			return nil
		}
		// Match .schema.json files OR files with -schema- in the name
		if !strings.HasSuffix(name, ".schema.json") && !strings.Contains(name, "-schema-") {
			return nil
		}
		cls, parseErr := ParseCDMClassFromFile(path)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		classes = append(classes, *cls)
		return nil
	}

	if err := filepath.WalkDir(schemaRoot, walkFn); err != nil {
		return nil, err
	}

	return classes, nil
}

// BuildCDMGraph converts parsed CDM classes into node and edge specs tied to known node type IDs.
func BuildCDMGraph(classes []CDMClass, typeIDs CDMTypeIDs, tenantID uuid.UUID, tenantDatasourceID *uuid.UUID) ([]NodeInput, []CDMEdgeSpec) {
	nodeSeen := map[string]bool{}
	nodes := make([]NodeInput, 0, len(classes))
	edges := make([]CDMEdgeSpec, 0)

	classPathByName := map[string]string{}
	classDefined := map[string]bool{}
	for _, cls := range classes {
		classPathByName[cls.Name] = classQualifiedPath(cls.Namespace, cls.Name)
		classDefined[cls.Name] = true
	}

	ensureClassNode := func(name string) string {
		path := classPathByName[name]
		if path == "" {
			path = classQualifiedPath(cdmDefaultNamespace, name)
		}
		if nodeSeen[path] {
			return path
		}
		if classDefined[name] {
			return path
		}

		nodes = append(nodes, NodeInput{
			TypeID:        typeIDs.Class,
			Name:          name,
			QualifiedPath: path,
			Properties: map[string]any{
				"namespace":   cdmDefaultNamespace,
				"kind":        "class",
				"placeholder": true,
			},
			Config:             map[string]any{},
			TenantID:           tenantID,
			TenantDatasourceID: tenantDatasourceID,
		})
		nodeSeen[path] = true
		classPathByName[name] = path
		return path
	}

	for _, cls := range classes {
		classPath := classPathByName[cls.Name]
		nodes = append(nodes, NodeInput{
			TypeID:        typeIDs.Class,
			Name:          cls.Name,
			QualifiedPath: classPath,
			Properties: map[string]any{
				"namespace": cls.Namespace,
				"kind":      cls.Kind,
			},
			Config:             map[string]any{},
			TenantID:           tenantID,
			TenantDatasourceID: tenantDatasourceID,
		})
		nodeSeen[classPath] = true

		requiredEdges := func(fromPath string, fromType uuid.UUID, toPath string, toType uuid.UUID, edgeType string, props map[string]any) {
			edges = append(edges, CDMEdgeSpec{
				Type:       edgeType,
				SourceType: fromType,
				SourcePath: fromPath,
				TargetType: toType,
				TargetPath: toPath,
				Props:      props,
			})
		}

		for _, f := range cls.Fields {
			fieldPath := fmt.Sprintf("%s/field/%s", classPath, f.Name)
			if !nodeSeen[fieldPath] {
				nodes = append(nodes, NodeInput{
					TypeID:        typeIDs.Field,
					Name:          f.Name,
					QualifiedPath: fieldPath,
					Properties: map[string]any{
						"data_type":   f.Type,
						"cardinality": f.Cardinality,
						"is_enum":     f.IsEnum,
					},
					Config:             map[string]any{},
					TenantID:           tenantID,
					TenantDatasourceID: tenantDatasourceID,
				})
				nodeSeen[fieldPath] = true
			}

			requiredEdges(classPath, typeIDs.Class, fieldPath, typeIDs.Field, "HAS_FIELD", map[string]any{
				"cardinality": f.Cardinality,
				"data_type":   f.Type,
				"is_enum":     f.IsEnum,
			})
		}

		for _, e := range cls.Enums {
			enumPath := fmt.Sprintf("%s/enum/%s", classPath, e.Name)
			if !nodeSeen[enumPath] {
				nodes = append(nodes, NodeInput{
					TypeID:        typeIDs.Enum,
					Name:          e.Name,
					QualifiedPath: enumPath,
					Properties: map[string]any{
						"owner": cls.Name,
					},
					Config:             map[string]any{},
					TenantID:           tenantID,
					TenantDatasourceID: tenantDatasourceID,
				})
				nodeSeen[enumPath] = true
			}

			requiredEdges(classPath, typeIDs.Class, enumPath, typeIDs.Enum, "HAS_FIELD", map[string]any{
				"cardinality": "1",
				"field_kind":  "enum",
			})

			for _, val := range e.Values {
				valPath := fmt.Sprintf("%s/value/%s", enumPath, val)
				if !nodeSeen[valPath] {
					nodes = append(nodes, NodeInput{
						TypeID:        typeIDs.EnumValue,
						Name:          val,
						QualifiedPath: valPath,
						Properties: map[string]any{
							"enum": e.Name,
						},
						Config:             map[string]any{},
						TenantID:           tenantID,
						TenantDatasourceID: tenantDatasourceID,
					})
					nodeSeen[valPath] = true
				}

				requiredEdges(enumPath, typeIDs.Enum, valPath, typeIDs.EnumValue, "HAS_ENUM_VALUE", map[string]any{})
			}
		}

		for _, assoc := range cls.Associations {
			targetPath := ensureClassNode(assoc.TargetType)
			requiredEdges(classPath, typeIDs.Class, targetPath, typeIDs.Class, "ASSOCIATES_TO", map[string]any{
				"cardinality": assoc.Cardinality,
				"role":        assoc.AssociationRole,
			})
		}

		for _, super := range cls.SuperTypes {
			superPath := ensureClassNode(super)
			requiredEdges(classPath, typeIDs.Class, superPath, typeIDs.Class, "EXTENDS", map[string]any{})
		}
	}

	return nodes, edges
}

func listCDMSchemaEntries(ctx context.Context) ([]ghContentEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cdmSchemaListURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list schemas status %d: %s", resp.StatusCode, string(body))
	}

	var entries []ghContentEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	filtered := make([]ghContentEntry, 0, len(entries))
	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		if !strings.HasSuffix(e.Name, ".schema.json") {
			continue
		}
		if strings.TrimSpace(e.DownloadURL) == "" {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered, nil
}

func parseCDMClassFromURL(ctx context.Context, url string) (*CDMClass, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch status %d", resp.StatusCode)
	}

	return parseCDMClassFromBytes(body, url)
}

const cdmDefaultNamespace = "cdm"

func classQualifiedPath(namespace, name string) string {
	return fmt.Sprintf("cdm/%s/%s", namespace, name)
}

func deriveNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".schema.json")
	return name
}

func trimRef(ref string) string {
	ref = strings.TrimPrefix(ref, "#/definitions/")
	ref = strings.TrimSuffix(ref, ".schema.json")
	return ref
}

func fieldCardinality(name string, required map[string]struct{}) string {
	if _, ok := required[name]; ok {
		return "1"
	}
	return "0..1"
}

func ensureCDMRepo(localDir string) (string, error) {
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--depth", "1", cdmRepoURL, localDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git clone failed: %w", err)
		}
	} else if err == nil {
		cmd := exec.Command("git", "-C", localDir, "pull", "--ff-only")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git pull failed: %w", err)
		}
	} else {
		return "", fmt.Errorf("stat %s: %w", localDir, err)
	}

	schemaRoot := filepath.Join(localDir, cdmSchemaRelPath)
	if _, err := os.Stat(schemaRoot); err == nil {
		return schemaRoot, nil
	}

	// Fallback: discover first directory containing *-schema-*.json or *.schema.json
	var discovered string
	_ = filepath.WalkDir(localDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || discovered != "" {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".json") && (strings.Contains(name, "-schema-") || strings.HasSuffix(name, ".schema.json")) {
			discovered = filepath.Dir(p)
			return fs.SkipDir
		}
		return nil
	})
	if discovered == "" {
		return "", fmt.Errorf("schema root not found under %s", localDir)
	}
	return discovered, nil
}

func ParseCDMClassFromFile(path string) (*CDMClass, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return parseCDMClassFromBytes(body, path)
}

func parseCDMClassFromBytes(body []byte, nameHint string) (*CDMClass, error) {
	var schema cdmSchema
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, err
	}

	className := strings.TrimSpace(schema.Title)
	if className == "" {
		className = deriveNameFromURL(nameHint)
	}

	cls := &CDMClass{
		Name:         className,
		Namespace:    cdmDefaultNamespace,
		Kind:         "class",
		SuperTypes:   []string{},
		Fields:       []CDMField{},
		Enums:        []CDMEnum{},
		Associations: []CDMAssociation{},
	}

	requiredSet := make(map[string]struct{}, len(schema.Required))
	for _, r := range schema.Required {
		requiredSet[r] = struct{}{}
	}

	for _, all := range schema.AllOf {
		if refRaw, ok := all["$ref"]; ok {
			var refStr string
			_ = json.Unmarshal(refRaw, &refStr)
			if refStr != "" {
				cls.SuperTypes = append(cls.SuperTypes, trimRef(refStr))
			}
		}
	}

	for propName, raw := range schema.Properties {
		var prop propertySchema
		_ = json.Unmarshal(raw, &prop)

		if len(prop.Enum) > 0 {
			cls.Enums = append(cls.Enums, CDMEnum{Name: propName, Values: prop.Enum})
			continue
		}

		if prop.Ref != "" {
			cls.Associations = append(cls.Associations, CDMAssociation{
				Name:        propName,
				TargetType:  trimRef(prop.Ref),
				Cardinality: "1",
			})
			continue
		}

		if prop.Type == "array" && prop.Items != nil && prop.Items.Ref != "" {
			cls.Associations = append(cls.Associations, CDMAssociation{
				Name:        propName,
				TargetType:  trimRef(prop.Items.Ref),
				Cardinality: "0..*",
			})
			continue
		}

		if prop.Type == "array" && prop.Items != nil && prop.Items.Type != "" {
			cls.Fields = append(cls.Fields, CDMField{
				Name:        propName,
				Type:        prop.Items.Type,
				Cardinality: "0..*",
				IsEnum:      false,
			})
			continue
		}

		if prop.Type != "" {
			cls.Fields = append(cls.Fields, CDMField{
				Name:        propName,
				Type:        prop.Type,
				Cardinality: fieldCardinality(propName, requiredSet),
				IsEnum:      false,
			})
		}
	}

	return cls, nil
}
