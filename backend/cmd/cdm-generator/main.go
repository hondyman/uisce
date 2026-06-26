package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/catalog"
)

func main() {
	cdmPath := flag.String("path", "./internal/cdm", "Path to CDM Go files")
	outputFile := flag.String("output", "catalog.json", "Output file path")
	flag.Parse()

	// List of CDM types we want to expose in our catalog
	// In a real scenario, this might come from a config file
	targetTypes := map[string]struct{}{
		"MockSwap":            {},
		"InterestRatePayout":  {},
		"CreditDefaultPayout": {},
		"CommodityPayout":     {},
		"OptionPayout":        {},
		"ForeignExchange":     {},
		"AssetPayout":         {},
	}

	cat := generateCatalog(*cdmPath, targetTypes)

	file, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cat); err != nil {
		log.Fatalf("Failed to encode catalog: %v", err)
	}

	fmt.Printf("Successfully generated catalog with %d products to %s\n", len(cat.Products), *outputFile)
}

func generateCatalog(root string, targets map[string]struct{}) catalog.Catalog {
	var products []catalog.ProductDef

	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			return nil
		}

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				name := typeSpec.Name.Name
				if _, desired := targets[name]; !desired {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				product := catalog.ProductDef{
					ID:          name,
					Label:       splitCamelCase(name),
					Family:      guessFamily(name),
					CdmType:     name,
					Description: fmt.Sprintf("Generated from CDM type %s", name),
					Attributes:  []catalog.AttributeDef{},
				}

				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue // Embedded field
					}
					fieldName := field.Names[0].Name

					// Naive type mapper
					attrType := mapAstTypeToAttributeType(field.Type)

					attr := catalog.AttributeDef{
						Name:     fieldName,
						Type:     attrType,
						Required: true, // Defaulting to true for now
					}
					product.Attributes = append(product.Attributes, attr)
				}

				products = append(products, product)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking CDM directory: %v", err)
	}

	return catalog.Catalog{Products: products}
}

func mapAstTypeToAttributeType(expr ast.Expr) catalog.AttributeType {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return catalog.AttrString
		case "int", "int32", "int64", "float32", "float64":
			return catalog.AttrNumber
		case "bool":
			return catalog.AttrBool
		default:
			// Custom structs/types map to object
			return catalog.AttrObject
		}
	case *ast.ArrayType:
		return catalog.AttrArray
	case *ast.StarExpr:
		return mapAstTypeToAttributeType(t.X)
	case *ast.SelectorExpr:
		// e.g. time.Time
		return catalog.AttrObject
	default:
		return catalog.AttrObject
	}
}

func splitCamelCase(s string) string {
	// Simple helper to make labels nicer (e.g. InterestRateSwap -> Interest Rate Swap)
	// For production, use a regex or library
	return s // Placeholder
}

func guessFamily(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "swap") {
		return "Derivatives"
	}
	if strings.Contains(lower, "nav") || strings.Contains(lower, "fund") {
		return "Funds"
	}
	return "Uncategorized"
}
