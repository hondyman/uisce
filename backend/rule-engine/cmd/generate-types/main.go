package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
)

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

// ASLTypeGenerator generates TypeScript definitions from Go structs
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

	if err := g.generateTypeScript(structs, enums); err != nil {
		return fmt.Errorf("failed to generate TypeScript: %w", err)
	}

	if err := g.generateJSONSchema(structs, enums); err != nil {
		return fmt.Errorf("failed to generate JSON Schema: %w", err)
	}

	if err := g.generateMonacoMetadata(structs, enums); err != nil {
		return fmt.Errorf("failed to generate Monaco metadata: %w", err)
	}

	if err := g.generateVersionInfo(); err != nil {
		return fmt.Errorf("failed to generate version info: %w", err)
	}

	return nil
}

// parsePackages loads Go packages and extracts structs and enums
func (g *ASLTypeGenerator) parsePackages() (map[string]*StructInfo, map[string][]string) {
	fset := token.NewFileSet()
	structs := make(map[string]*StructInfo)
	enums := make(map[string][]string)

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
func (g *ASLTypeGenerator) extractFromFile(file *ast.File, structs map[string]*StructInfo, enums map[string][]string) {
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
func (g *ASLTypeGenerator) extractTypes(decl *ast.GenDecl, structs map[string]*StructInfo, enums map[string][]string) {
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
				enums[typeSpec.Name.Name] = []string{}
				// We'll populate enum values from const declarations
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
func (g *ASLTypeGenerator) extractConsts(decl *ast.GenDecl, enums map[string][]string) {
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
			// This is a simplified check - in practice you'd need more sophisticated
			// analysis to match consts to their types
			for typeName := range enums {
				if strings.Contains(valueSpec.Names[0].Name, strings.Title(typeName)) ||
					strings.HasPrefix(valueSpec.Names[0].Name, typeName) {
					enums[typeName] = append(enums[typeName], value)
					// Sort after adding for deterministic order
					sort.Strings(enums[typeName])
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

// generateTypeScript generates TypeScript definition files
func (g *ASLTypeGenerator) generateTypeScript(structs map[string]*StructInfo, enums map[string][]string) error {
	os.MkdirAll(g.outputDir, 0755)

	// Generate asl.d.ts
	tsContent := g.generateTSContent(structs, enums)
	return os.WriteFile(g.outputDir+"/asl.d.ts", []byte(tsContent), 0644)
}

// generateTSContent creates TypeScript interface definitions
func (g *ASLTypeGenerator) generateTSContent(structs map[string]*StructInfo, enums map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("// Code generated by ASL type generator. DO NOT EDIT.\n\n")

	// Generate enums first (sorted for deterministic output)
	enumNames := make([]string, 0, len(enums))
	for name := range enums {
		enumNames = append(enumNames, name)
	}
	sort.Strings(enumNames)

	for _, enumName := range enumNames {
		values := enums[enumName]
		if len(values) > 0 {
			// Sort enum values for deterministic output
			sortedValues := make([]string, len(values))
			copy(sortedValues, values)
			sort.Strings(sortedValues)

			sb.WriteString(fmt.Sprintf("export type %s = %s;\n\n",
				enumName, strings.Join(sortedValues, " | ")))
		}
	}

	// Generate interfaces
	structNames := make([]string, 0, len(structs))
	for name := range structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		structInfo := structs[name]

		if structInfo.Doc != "" {
			sb.WriteString(fmt.Sprintf("/** %s */\n", structInfo.Doc))
		}

		sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

		// Sort fields for deterministic output
		fieldNames := make([]string, 0, len(structInfo.Fields))
		fieldMap := make(map[string]FieldInfo)
		for _, field := range structInfo.Fields {
			fieldNames = append(fieldNames, field.Name)
			fieldMap[field.Name] = field
		}
		sort.Strings(fieldNames)

		for _, fieldName := range fieldNames {
			field := fieldMap[fieldName]
			if field.Doc != "" {
				sb.WriteString(fmt.Sprintf("  /** %s */\n", field.Doc))
			}

			optional := ""
			if field.Optional {
				optional = "?"
			}

			tsType := g.goTypeToTSType(field.Type, enums)
			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// goTypeToTSType converts Go types to TypeScript types
func (g *ASLTypeGenerator) goTypeToTSType(goType string, enums map[string][]string) string {
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

// generateJSONSchema generates JSON Schema for validation
func (g *ASLTypeGenerator) generateJSONSchema(structs map[string]*StructInfo, enums map[string][]string) error {
	schema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$id":         "https://semlayer.com/schemas/asl-rule.schema.json",
		"title":       "ASL Rule Schema",
		"type":        "object",
		"definitions": make(map[string]interface{}),
	}

	definitions := schema["definitions"].(map[string]interface{})

	// Add enum definitions
	for enumName, values := range enums {
		if len(values) > 0 {
			definitions[enumName] = map[string]interface{}{
				"type": "string",
				"enum": values,
			}
		}
	}

	// Add struct definitions
	for name, structInfo := range structs {
		properties := make(map[string]interface{})

		for _, field := range structInfo.Fields {
			fieldSchema := g.fieldToJSONSchema(field, enums)
			properties[field.Name] = fieldSchema
		}

		definitions[name] = map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(g.outputDir+"/asl.schema.json", data, 0644)
}

// fieldToJSONSchema converts a field to JSON Schema
func (g *ASLTypeGenerator) fieldToJSONSchema(field FieldInfo, enums map[string][]string) map[string]interface{} {
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

// generateMonacoMetadata generates Monaco editor completion metadata
func (g *ASLTypeGenerator) generateMonacoMetadata(structs map[string]*StructInfo, enums map[string][]string) error {
	metadata := map[string]interface{}{
		"keywords": []string{
			"rule", "condition", "group", "and", "or", "not",
			"equals", "gt", "lt", "gte", "lte", "contains",
		},
		"types":     make(map[string]interface{}),
		"operators": []string{"=", ">", "<", ">=", "<=", "!=", "in", "contains"},
	}

	types := metadata["types"].(map[string]interface{})

	// Add type information for completion
	for name, structInfo := range structs {
		properties := make(map[string]interface{})

		for _, field := range structInfo.Fields {
			properties[field.Name] = map[string]interface{}{
				"type":          g.goTypeToTSType(field.Type, enums),
				"documentation": field.Doc,
			}
		}

		typeInfo := map[string]interface{}{
			"kind":          "interface",
			"documentation": structInfo.Doc,
			"properties":    properties,
		}
		types[name] = typeInfo
	}

	// Add enum completions
	for enumName, values := range enums {
		if len(values) > 0 {
			types[enumName] = map[string]interface{}{
				"kind":   "enum",
				"values": values,
			}
		}
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(g.outputDir+"/asl.monaco.json", data, 0644)
}

// generateVersionInfo generates version metadata
func (g *ASLTypeGenerator) generateVersionInfo() error {
	version := map[string]interface{}{
		"generator": "asl-type-generator",
		"version":   "1.0.0",
		"timestamp": "2026-02-02T00:00:00Z", // Would be dynamic in real implementation
		"commit":    "unknown",              // Would be git hash in real implementation
		"packages":  g.packages,
	}

	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(g.outputDir+"/version.json", data, 0644)
}

func main() {
	packages := []string{
		"../../../internal/services",
		"../../../internal/rules",
		"../../../internal/models",
	}

	generator := NewASLTypeGenerator(packages, "../../generated")
	if err := generator.Generate(); err != nil {
		log.Fatalf("Failed to generate ASL types: %v", err)
	}

	fmt.Println("ASL type generation completed successfully!")
}
