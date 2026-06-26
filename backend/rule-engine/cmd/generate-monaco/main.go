package main

import (
	"encoding/json"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

const (
	goPkgPath    = "github.com/hondyman/semlayer/backend/internal/services"
	outputDir    = "generated"
	outputMonaco = "asl.monaco.json"
)

type MonacoSnippet struct {
	Label  string `json:"label"`
	Insert string `json:"insert"`
	Detail string `json:"detail"`
}

type MonacoMetadata struct {
	Keywords  []string            `json:"keywords"`
	Operators []string            `json:"operators"`
	NodeKinds []string            `json:"nodeKinds"`
	Snippets  []MonacoSnippet     `json:"snippets"`
	Enums     map[string][]string `json:"enums"`
}

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}

	pkgs, err := packages.Load(cfg, goPkgPath)
	if err != nil {
		log.Printf("Warning: failed to load packages: %v", err)
		// Continue with empty metadata
		pkgs = []*packages.Package{}
	}
	if packages.PrintErrors(pkgs) > 0 {
		log.Printf("Warning: errors while loading packages")
		// Continue with empty metadata
		pkgs = []*packages.Package{}
	}

	meta := MonacoMetadata{
		Keywords:  []string{"rule", "when", "then", "and", "or", "not"},
		Operators: []string{},
		NodeKinds: []string{},
		Snippets:  []MonacoSnippet{},
		Enums:     map[string][]string{},
	}

	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		names := scope.Names()
		sort.Strings(names)

		for _, name := range names {
			obj := scope.Lookup(name)
			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}

			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}

			// Enums
			if isEnum(named) {
				values := collectEnumValues(named)
				sort.Strings(values)
				meta.Enums[name] = values

				if isOperatorEnum(name) {
					meta.Operators = append(meta.Operators, values...)
				}
				continue
			}

			// Structs
			structType, ok := named.Underlying().(*types.Struct)
			if !ok {
				continue
			}

			if hasDiscriminator(structType) {
				meta.NodeKinds = append(meta.NodeKinds, name)
				meta.Snippets = append(meta.Snippets, MonacoSnippet{
					Label:  name,
					Insert: name + " {\n  \n}",
					Detail: "Insert " + name + " node",
				})
			}
		}
	}

	sort.Strings(meta.Operators)
	sort.Strings(meta.NodeKinds)
	sort.Slice(meta.Snippets, func(i, j int) bool {
		return meta.Snippets[i].Label < meta.Snippets[j].Label
	})

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	outPath := filepath.Join(outputDir, outputMonaco)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal monaco metadata: %v", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		log.Fatalf("failed to write %s: %v", outPath, err)
	}
}

func isEnum(named *types.Named) bool {
	_, ok := named.Underlying().(*types.Basic)
	return ok
}

func collectEnumValues(named *types.Named) []string {
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return nil
	}

	scope := pkg.Scope()
	names := scope.Names()
	sort.Strings(names)

	var values []string
	for _, name := range names {
		obj := scope.Lookup(name)
		c, ok := obj.(*types.Const)
		if ok && c.Type() == named {
			values = append(values, c.Val().ExactString())
		}
	}
	return values
}

func isOperatorEnum(name string) bool {
	return name == "Operator" || name == "RuleOperator"
}

func hasDiscriminator(s *types.Struct) bool {
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if f.Exported() && (f.Name() == "Type" || f.Name() == "Kind") {
			return true
		}
	}
	return false
}
