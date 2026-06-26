package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// EnumInfo holds information about an enum
type EnumInfo struct {
	Name   string
	Values []string
}

// StructInfo holds information about a Go struct
type StructInfo struct {
	Name   string
	Doc    string
	Fields []FieldInfo
}

// FieldInfo holds information about a struct field
type FieldInfo struct {
	Name     string
	Type     string
	Doc      string
	JSONTag  string
	Optional bool
}

// ASLTypeGenerator generates JSON Schema from Go structs
type ASLTypeGenerator struct {
	packages  []string
	outputDir string
}

// NewASLTypeGenerator creates a new generator
func NewASLTypeGenerator(packages []string, outputDir string) *ASLTypeGenerator {
	return &ASLTypeGenerator{
		packages:  packages,
		outputDir: outputDir,
	}
}

// Generate runs the full generation process
func (g *ASLTypeGenerator) Generate() error {
	structs, enums := g.parsePackages()

	if err := g.generateJSONSchema(structs, enums); err != nil {
		return fmt.Errorf("failed to generate JSON Schema: %w", err)
	}

	return nil
}

// parsePackages loads Go packages and extracts structs and enums
func (g *ASLTypeGenerator) parsePackages() (map[string]*StructInfo, map[string]*EnumInfo) {
	fset := token.NewFileSet()
	structs := make(map[string]*StructInfo)
	enums := make(map[string]*EnumInfo)

	// Sort packages for deterministic processing
	sortedPackages := make([]string, len(g.packages))
	copy(sortedPackages, g.packages)
	sort.Strings(sortedPackages)

	for _, pkgPath := range sortedPackages {
		pkgs, err := parser.ParseDir(fset, pkgPath, nil, parser.ParseComments)
		if err != nil {
			log.Printf("Warning: failed to parse package %s: %v", pkgPath, err)
			continue
		}

		// Sort package names for deterministic processing
		pkgNames := make([]string, 0, len(pkgs))
		for name := range pkgs {
			pkgNames = append(pkgNames, name)
		}
		sort.Strings(pkgNames)

		for _, pkgName := range pkgNames {
			pkg := pkgs[pkgName]

			// Sort files for deterministic processing
			fileNames := make([]string, 0, len(pkg.Files))
			for name := range pkg.Files {
				fileNames = append(fileNames, name)
			}
			sort.Strings(fileNames)

			for _, fileName := range fileNames {
				file := pkg.Files[fileName]
				g.extractFromFile(file, structs, enums)
			}
		}
	}

	return structs, enums
}

// extractFromFile parses a Go file for structs and enums
func (g *ASLTypeGenerator) extractFromFile(file *ast.File, structs map[string]*StructInfo, enums map[string]*EnumInfo) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				g.extractTypes(d, structs, enums)
			} else if d.Tok == token.CONST {
				g.extractConsts(d, enums)
			}
		}
	}
}

// extractTypes extracts struct and type alias information
func (g *ASLTypeGenerator) extractTypes(decl *ast.GenDecl, structs map[string]*StructInfo, enums map[string]*EnumInfo) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		switch t := typeSpec.Type.(type) {
		case *ast.StructType:
			structInfo := &StructInfo{
				Name:   typeSpec.Name.Name,
				Fields: []FieldInfo{},
			}

			if decl.Doc != nil {
				structInfo.Doc = strings.TrimSpace(decl.Doc.Text())
			}

			for _, field := range t.Fields.List {
				fieldInfo := g.extractFieldInfo(field)
				structInfo.Fields = append(structInfo.Fields, fieldInfo)
			}

			structs[typeSpec.Name.Name] = structInfo

		case *ast.Ident:
			// Type alias - check if it's a string-based enum
			if t.Name == "string" {
				enums[typeSpec.Name.Name] = &EnumInfo{
					Name:   typeSpec.Name.Name,
					Values: []string{},
				}
			}
		}
	}
}

// extractFieldInfo extracts field information from an AST field
func (g *ASLTypeGenerator) extractFieldInfo(field *ast.Field) FieldInfo {
	fieldInfo := FieldInfo{}

	if len(field.Names) > 0 {
		fieldInfo.Name = field.Names[0].Name
	}

	fieldInfo.Type = g.goTypeToString(field.Type)

	if field.Tag != nil {
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		fieldInfo.JSONTag = tag.Get("json")
		fieldInfo.Doc = tag.Get("doc")
		if tag.Get("optional") == "true" {
			fieldInfo.Optional = true
		}
	}

	if field.Doc != nil {
		fieldInfo.Doc = strings.TrimSpace(field.Doc.Text())
	}

	return fieldInfo
}

// extractConsts extracts enum values from const declarations
func (g *ASLTypeGenerator) extractConsts(decl *ast.GenDecl, enums map[string]*EnumInfo) {
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		if len(valueSpec.Names) == 0 || len(valueSpec.Values) == 0 {
			continue
		}

		// Check if this is an enum value (string literal)
		if basicLit, ok := valueSpec.Values[0].(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
			value := strings.Trim(basicLit.Value, `"`)

			// Find the type this const belongs to
			for _, enumInfo := range enums {
				if strings.Contains(valueSpec.Names[0].Name, strings.Title(enumInfo.Name)) ||
					strings.HasPrefix(valueSpec.Names[0].Name, enumInfo.Name) {
					enumInfo.Values = append(enumInfo.Values, value)
					// Sort after adding for deterministic order
					sort.Strings(enumInfo.Values)
				}
			}
		}
	}
}

// goTypeToString converts Go AST type to string representation
func (g *ASLTypeGenerator) goTypeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return g.goTypeToString(t.X)
	case *ast.ArrayType:
		return g.goTypeToString(t.Elt) + "[]"
	case *ast.MapType:
		return fmt.Sprintf("Record<%s, %s>", g.goTypeToString(t.Key), g.goTypeToString(t.Value))
	case *ast.InterfaceType:
		return "any"
	default:
		return "any"
	}
}

// generateJSONSchema generates JSON Schema for validation
func (g *ASLTypeGenerator) generateJSONSchema(structs map[string]*StructInfo, enums map[string]*EnumInfo) error {
	schema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$id":         "https://semlayer.com/schemas/asl-rule.schema.json",
		"title":       "ASL Rule Schema",
		"type":        "object",
		"definitions": make(map[string]interface{}),
	}

	definitions := schema["definitions"].(map[string]interface{})

	// Add enum definitions
	for name, enumInfo := range enums {
		if len(enumInfo.Values) > 0 {
			definitions[name] = map[string]interface{}{
				"type": "string",
				"enum": enumInfo.Values,
			}
		}
	}

	// Add struct definitions
	for name, structInfo := range structs {
		properties := make(map[string]interface{})
		required := []string{}

		for _, field := range structInfo.Fields {
			fieldSchema := g.fieldToJSONSchema(field, enums)
			properties[field.Name] = fieldSchema

			if !field.Optional {
				required = append(required, field.Name)
			}
		}

		def := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}

		if len(required) > 0 {
			def["required"] = required
		}

		if structInfo.Doc != "" {
			def["description"] = structInfo.Doc
		}

		definitions[name] = def
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(g.outputDir, "asl.schema.json"), data, 0644)
}

// fieldToJSONSchema converts a field to JSON Schema
func (g *ASLTypeGenerator) fieldToJSONSchema(field FieldInfo, enums map[string]*EnumInfo) map[string]interface{} {
	schema := make(map[string]interface{})

	tsType := g.goTypeToTSType(field.Type, enums)

	switch tsType {
	case "string":
		schema["type"] = "string"
	case "number":
		schema["type"] = "number"
	case "boolean":
		schema["type"] = "boolean"
	case "any":
		// Allow any type
	default:
		// Reference to another type
		schema["$ref"] = "#/definitions/" + tsType
	}

	if field.Doc != "" {
		schema["description"] = field.Doc
	}

	return schema
}

// goTypeToTSType converts Go types to TypeScript types
func (g *ASLTypeGenerator) goTypeToTSType(goType string, enums map[string]*EnumInfo) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int64", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "interface{}":
		return "any"
	default:
		// Check if it's an enum
		if _, isEnum := enums[goType]; isEnum {
			return goType
		}
		// Check if it's a known struct
		return goType
	}
}

func main() {
	packages := []string{
		"../../../internal/services",
		"../../../internal/rules",
		"../../../internal/models",
	}

	generator := NewASLTypeGenerator(packages, "../../generated")
	if err := generator.Generate(); err != nil {
		log.Fatalf("Failed to generate ASL schema: %v", err)
	}

	fmt.Println("ASL schema generation completed successfully!")
}
