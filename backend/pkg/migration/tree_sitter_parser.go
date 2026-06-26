package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// TreeSitterParser provides AST parsing for multiple languages using tree-sitter
type TreeSitterParser struct {
	// Path to tree-sitter CLI or wrapper script
	CLIPath string
	// Supported languages with their grammar modules
	SupportedLanguages map[string]string
}

func NewTreeSitterParser() *TreeSitterParser {
	return &TreeSitterParser{
		CLIPath: "tree-sitter", // Assumes tree-sitter CLI is in PATH
		SupportedLanguages: map[string]string{
			"java":   "tree-sitter-java",
			"csharp": "tree-sitter-c-sharp",
			"python": "tree-sitter-python",
			"cobol":  "tree-sitter-cobol",
			"sql":    "tree-sitter-sql",
			"go":     "tree-sitter-go",
		},
	}
}

// TreeSitterNode represents a node in the tree-sitter AST
type TreeSitterNode struct {
	Type       string           `json:"type"`
	Name       string           `json:"name,omitempty"`
	Text       string           `json:"text,omitempty"`
	StartPoint Point            `json:"startPoint"`
	EndPoint   Point            `json:"endPoint"`
	Children   []TreeSitterNode `json:"children,omitempty"`
}

type Point struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}

// Parse parses source code using tree-sitter and returns a structured AST
func (p *TreeSitterParser) Parse(ctx context.Context, sourceCode string, language string) (*ParsedCode, error) {
	lang := strings.ToLower(language)

	// Check if language is supported
	grammar, ok := p.SupportedLanguages[lang]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Check if tree-sitter is available
	if !p.isTreeSitterAvailable() {
		// Fallback to stub parsing
		return p.fallbackParse(sourceCode, language)
	}

	// Use tree-sitter CLI to parse
	ast, err := p.parseWithCLI(ctx, sourceCode, grammar)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parsing failed: %w", err)
	}

	// Convert tree-sitter AST to our ASTNode format
	rootNode := p.convertToASTNode(ast)

	return &ParsedCode{
		Language: language,
		FileName: fmt.Sprintf("source.%s", getExtension(language)),
		RootNode: rootNode,
	}, nil
}

// isTreeSitterAvailable checks if tree-sitter CLI is installed
func (p *TreeSitterParser) isTreeSitterAvailable() bool {
	cmd := exec.Command(p.CLIPath, "--version")
	return cmd.Run() == nil
}

// parseWithCLI uses tree-sitter CLI to parse code
func (p *TreeSitterParser) parseWithCLI(ctx context.Context, sourceCode string, grammar string) (*TreeSitterNode, error) {
	// Create a temp file with the source code
	// Run tree-sitter parse <file> --format json
	// Parse the JSON output

	// For now, we'll use a simplified approach via stdin
	// In production, you'd use tree-sitter bindings directly

	cmd := exec.CommandContext(ctx, p.CLIPath, "parse", "--format", "json", "-")
	cmd.Stdin = strings.NewReader(sourceCode)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tree-sitter CLI failed: %w", err)
	}

	var ast TreeSitterNode
	if err := json.Unmarshal(output, &ast); err != nil {
		return nil, fmt.Errorf("failed to parse tree-sitter output: %w", err)
	}

	return &ast, nil
}

// convertToASTNode converts TreeSitterNode to our ASTNode format
func (p *TreeSitterParser) convertToASTNode(ts *TreeSitterNode) *ASTNode {
	if ts == nil {
		return nil
	}

	node := &ASTNode{
		ID:        fmt.Sprintf("%s_%d_%d", ts.Type, ts.StartPoint.Row, ts.StartPoint.Column),
		Type:      mapTreeSitterType(ts.Type),
		Name:      ts.Name,
		LineStart: ts.StartPoint.Row + 1, // tree-sitter is 0-indexed
		LineEnd:   ts.EndPoint.Row + 1,
	}

	// Extract condition for conditional nodes
	if ts.Type == "if_statement" || ts.Type == "while_statement" {
		for _, child := range ts.Children {
			if child.Type == "condition" || child.Type == "parenthesized_expression" {
				node.Condition = child.Text
				break
			}
		}
	}

	// Convert children
	for _, child := range ts.Children {
		if childNode := p.convertToASTNode(&child); childNode != nil {
			node.Children = append(node.Children, *childNode)
		}
	}

	return node
}

// fallbackParse provides basic parsing when tree-sitter is unavailable
func (p *TreeSitterParser) fallbackParse(sourceCode string, language string) (*ParsedCode, error) {
	// Use regex-based heuristics for basic structure detection
	lines := strings.Split(sourceCode, "\n")

	rootNode := &ASTNode{
		ID:       "root",
		Type:     "file",
		Children: []ASTNode{},
	}

	currentFunction := ""
	functionStart := 0
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect function/method definitions (simplified)
		if containsAny(trimmed, []string{"public ", "private ", "protected ", "func ", "def "}) &&
			(strings.Contains(trimmed, "(") && !strings.HasPrefix(trimmed, "if") && !strings.HasPrefix(trimmed, "while")) {

			// Extract function name (simplified)
			name := extractFunctionName(trimmed)
			currentFunction = name
			functionStart = i + 1
		}

		// Track brace depth
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		// Function end
		if currentFunction != "" && braceCount == 0 && strings.Contains(line, "}") {
			fnNode := ASTNode{
				ID:        fmt.Sprintf("func_%s", currentFunction),
				Type:      "function",
				Name:      currentFunction,
				LineStart: functionStart,
				LineEnd:   i + 1,
			}
			rootNode.Children = append(rootNode.Children, fnNode)
			currentFunction = ""
		}

		// Detect if statements
		if strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "if(") {
			condition := extractCondition(trimmed)
			ifNode := ASTNode{
				ID:        fmt.Sprintf("if_%d", i+1),
				Type:      "if",
				Condition: condition,
				LineStart: i + 1,
				LineEnd:   i + 1, // Will be updated when we find matching brace
			}
			rootNode.Children = append(rootNode.Children, ifNode)
		}
	}

	return &ParsedCode{
		Language: language,
		FileName: fmt.Sprintf("source.%s", getExtension(language)),
		RootNode: rootNode,
		Errors:   []string{"Parsed with fallback heuristics (tree-sitter unavailable)"},
	}, nil
}

// Helper functions

func mapTreeSitterType(tsType string) string {
	mapping := map[string]string{
		"method_declaration":  "function",
		"function_definition": "function",
		"if_statement":        "if",
		"for_statement":       "loop",
		"while_statement":     "loop",
		"method_invocation":   "call",
		"assignment":          "assignment",
		"return_statement":    "return",
		"class_declaration":   "class",
	}
	if mapped, ok := mapping[tsType]; ok {
		return mapped
	}
	return tsType
}

func getExtension(language string) string {
	extensions := map[string]string{
		"java":   "java",
		"csharp": "cs",
		"python": "py",
		"cobol":  "cob",
		"sql":    "sql",
		"go":     "go",
	}
	if ext, ok := extensions[language]; ok {
		return ext
	}
	return "txt"
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func extractFunctionName(line string) string {
	// Very simplified extraction
	parts := strings.Fields(line)
	for i, part := range parts {
		if strings.Contains(part, "(") {
			name := strings.Split(part, "(")[0]
			return name
		}
		if i > 0 && strings.Contains(parts[i-1], ")") && part == "{" {
			// Previous might be return type, go back further
		}
	}
	return "unknown"
}

func extractCondition(line string) string {
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start != -1 && end > start {
		return line[start+1 : end]
	}
	return ""
}
