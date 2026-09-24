package mcp

import "fmt"

// Mutation names agents commonly invent. CallTool returns a clear refusal
// instead of "unknown tool" so the contract is explicit.
var refusedTools = map[string]struct{}{
	"save_record":            {},
	"delete_record":          {},
	"run_sql":                {},
	"create_business_object": {},
	"create_page_draft":      {},
}

const refusedMessage = "refused: page/MCP tools cannot mutate records, invent Business Objects, or write core pages (gold-copy admin uses Page Studio)"

func IsRefusedTool(name string) bool {
	_, ok := refusedTools[name]
	return ok
}

func refusedError(name string) error {
	return fmt.Errorf("%s: %s", name, refusedMessage)
}
