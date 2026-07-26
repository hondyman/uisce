package migration

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ASTNode represents a simplified AST node for LLM consumption
type ASTNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "function", "if", "loop", "assignment", "call"
	Name        string                 `json:"name,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
	Body        string                 `json:"body,omitempty"`
	Children    []ASTNode              `json:"children,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"` // Filled by LLM
	LineStart   int                    `json:"lineStart"`
	LineEnd     int                    `json:"lineEnd"`
}

// ParsedCode represents the full parsed result
type ParsedCode struct {
	Language string   `json:"language"`
	FileName string   `json:"fileName"`
	RootNode *ASTNode `json:"rootNode"`
	Errors   []string `json:"errors,omitempty"`
}

// ASTParserService handles parsing legacy code into structured AST
type ASTParserService struct {
	// Extensible: Add external parsers for COBOL, C#, etc.
}

func NewASTParserService() *ASTParserService {
	return &ASTParserService{}
}

// Parse parses source code and returns a simplified AST
func (s *ASTParserService) Parse(sourceCode string, language string) (*ParsedCode, error) {
	switch strings.ToLower(language) {
	case "go":
		return s.parseGo(sourceCode)
	case "java":
		return s.parseJavaStub(sourceCode)
	case "csharp", "c#":
		return s.parseCSharpStub(sourceCode)
	case "cobol":
		return s.parseCOBOLStub(sourceCode)
	case "sql":
		return s.parseSQLStub(sourceCode)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// parseGo uses Go's native parser for Go code
func (s *ASTParserService) parseGo(sourceCode string) (*ParsedCode, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "source.go", sourceCode, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go code: %w", err)
	}

	rootNode := &ASTNode{
		ID:       "root",
		Type:     "file",
		Name:     node.Name.Name,
		Children: []ASTNode{},
	}

	// Walk declarations
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			fnNode := s.parseFuncDecl(fn, fset, sourceCode)
			rootNode.Children = append(rootNode.Children, fnNode)
		}
	}

	return &ParsedCode{
		Language: "go",
		FileName: "source.go",
		RootNode: rootNode,
	}, nil
}

func (s *ASTParserService) parseFuncDecl(fn *ast.FuncDecl, fset *token.FileSet, source string) ASTNode {
	startPos := fset.Position(fn.Pos())
	endPos := fset.Position(fn.End())

	fnNode := ASTNode{
		ID:        fmt.Sprintf("func_%s", fn.Name.Name),
		Type:      "function",
		Name:      fn.Name.Name,
		LineStart: startPos.Line,
		LineEnd:   endPos.Line,
		Children:  []ASTNode{},
	}

	// Parse function body statements
	if fn.Body != nil {
		for _, stmt := range fn.Body.List {
			stmtNode := s.parseStmt(stmt, fset, source)
			if stmtNode != nil {
				fnNode.Children = append(fnNode.Children, *stmtNode)
			}
		}
	}

	return fnNode
}

func (s *ASTParserService) parseStmt(stmt ast.Stmt, fset *token.FileSet, source string) *ASTNode {
	startPos := fset.Position(stmt.Pos())
	endPos := fset.Position(stmt.End())

	switch st := stmt.(type) {
	case *ast.IfStmt:
		// Extract condition as string
		condStart := fset.Position(st.Cond.Pos()).Offset
		condEnd := fset.Position(st.Cond.End()).Offset
		condition := ""
		if condEnd <= len(source) && condStart < condEnd {
			condition = source[condStart:condEnd]
		}

		node := &ASTNode{
			ID:        fmt.Sprintf("if_%d", startPos.Line),
			Type:      "if",
			Condition: condition,
			LineStart: startPos.Line,
			LineEnd:   endPos.Line,
			Children:  []ASTNode{},
		}

		// Parse then block
		for _, bodyStmt := range st.Body.List {
			child := s.parseStmt(bodyStmt, fset, source)
			if child != nil {
				node.Children = append(node.Children, *child)
			}
		}

		return node

	case *ast.ForStmt:
		return &ASTNode{
			ID:        fmt.Sprintf("for_%d", startPos.Line),
			Type:      "loop",
			LineStart: startPos.Line,
			LineEnd:   endPos.Line,
		}

	case *ast.ExprStmt:
		if call, ok := st.X.(*ast.CallExpr); ok {
			funcName := ""
			if ident, ok := call.Fun.(*ast.Ident); ok {
				funcName = ident.Name
			} else if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				funcName = sel.Sel.Name
			}
			return &ASTNode{
				ID:        fmt.Sprintf("call_%d", startPos.Line),
				Type:      "call",
				Name:      funcName,
				LineStart: startPos.Line,
				LineEnd:   endPos.Line,
			}
		}

	case *ast.AssignStmt:
		return &ASTNode{
			ID:        fmt.Sprintf("assign_%d", startPos.Line),
			Type:      "assignment",
			LineStart: startPos.Line,
			LineEnd:   endPos.Line,
		}

	case *ast.ReturnStmt:
		return &ASTNode{
			ID:        fmt.Sprintf("return_%d", startPos.Line),
			Type:      "return",
			LineStart: startPos.Line,
			LineEnd:   endPos.Line,
		}
	}

	return nil
}

// Stub parsers for other languages - to be implemented with external tools

func (s *ASTParserService) parseJavaStub(sourceCode string) (*ParsedCode, error) {
	// TODO: Integrate with tree-sitter-java or Eclipse JDT
	return &ParsedCode{
		Language: "java",
		RootNode: &ASTNode{
			ID:   "root",
			Type: "file",
			Annotations: map[string]interface{}{
				"_stub": "Java parsing requires external parser integration (tree-sitter or Eclipse JDT)",
			},
			Body: sourceCode,
		},
	}, nil
}

func (s *ASTParserService) parseCSharpStub(sourceCode string) (*ParsedCode, error) {
	// TODO: Integrate with Roslyn or tree-sitter-csharp
	return &ParsedCode{
		Language: "csharp",
		RootNode: &ASTNode{
			ID:   "root",
			Type: "file",
			Annotations: map[string]interface{}{
				"_stub": "C# parsing requires Roslyn integration",
			},
			Body: sourceCode,
		},
	}, nil
}

func (s *ASTParserService) parseCOBOLStub(sourceCode string) (*ParsedCode, error) {
	// TODO: Integrate with GnuCOBOL or specialized COBOL parser
	return &ParsedCode{
		Language: "cobol",
		RootNode: &ASTNode{
			ID:   "root",
			Type: "file",
			Annotations: map[string]interface{}{
				"_stub": "COBOL parsing requires specialized parser (GnuCOBOL or proprietary)",
			},
			Body: sourceCode,
		},
	}, nil
}

func (s *ASTParserService) parseSQLStub(sourceCode string) (*ParsedCode, error) {
	// TODO: Integrate with sqlparser-go or similar
	return &ParsedCode{
		Language: "sql",
		RootNode: &ASTNode{
			ID:   "root",
			Type: "file",
			Annotations: map[string]interface{}{
				"_stub": "SQL parsing requires sqlparser integration",
			},
			Body: sourceCode,
		},
	}, nil
}

// ToJSON serializes the parsed code to JSON
func (p *ParsedCode) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
