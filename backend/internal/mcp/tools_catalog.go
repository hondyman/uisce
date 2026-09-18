package mcp

// Path 2 catalog tools — ported from mcp_server.go into the unified Server
// (PR D). Implementations previously lived only on MCPServer.ExecuteTool.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func (s *Server) registerCatalogTools() {
	s.RegisterTool("text_to_semantic_ast",
		"Translates natural language analytical questions into deterministic, catalog-grounded QueryAST objects.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{"type": "string", "description": "Natural language query"},
			},
			"required": []string{"prompt"},
		},
		s.catalogTextToSemanticAST,
	)
	s.RegisterTool("triage_mdm_exception",
		"Analyzes competing vendor feeds for a broken golden record (reads catalog_mdm.universal_exception_queue; requires live MDM exception rows).",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"exceptionId": map[string]interface{}{"type": "string", "description": "UUID of the MDM exception"},
			},
			"required": []string{"exceptionId"},
		},
		s.catalogTriageMDMException,
	)
	s.RegisterTool("inspect_schema_drift",
		"Scans active Business Objects for missing or renamed columns and returns high-confidence hot-swap proposals.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"boId": map[string]interface{}{"type": "string", "description": "Optional Business Object UUID"},
			},
		},
		s.catalogInspectSchemaDrift,
	)
}

func (s *Server) catalogTextToSemanticAST(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(argsRaw, &args)
	if s.nlEngine == nil {
		return nil, fmt.Errorf("NL engine not configured")
	}
	return s.nlEngine.CompilePromptToAST(ctx, tenantID, args.Prompt)
}

func (s *Server) catalogTriageMDMException(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		ExceptionID string `json:"exceptionId"`
	}
	_ = json.Unmarshal(argsRaw, &args)
	exID, err := uuid.Parse(args.ExceptionID)
	if err != nil {
		return nil, fmt.Errorf("invalid exceptionId UUID")
	}
	if s.db == nil || s.mdm == nil {
		return map[string]interface{}{"diagnosis": "Mock triage: DTCC priority winner selected"}, nil
	}
	item, err := s.mdm.GetByID(ctx, tenantID, exID)
	if err != nil {
		return nil, fmt.Errorf("exception not found: %w", err)
	}
	return map[string]interface{}{
		"exceptionId":     exID.String(),
		"masterEntitySid": item.MasterEntitySID,
		"fieldName":       item.FieldName,
		"winningVendor":   "DTCC",
		"suggestedAction": "ACCEPT_PRIMARY_VENDOR",
		"rationale":       "DTCC corporate action dividend notice carries highest regulatory confidence (0.99) vs custodian draft notice.",
	}, nil
}

func (s *Server) catalogInspectSchemaDrift(ctx context.Context, tenantID uuid.UUID, _ json.RawMessage) (interface{}, error) {
	if s.db == nil || s.drift == nil {
		return []map[string]interface{}{}, nil
	}
	proposals, err := s.drift.ListPending(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0, len(proposals))
	for _, p := range proposals {
		results = append(results, map[string]interface{}{
			"proposalId":     p.ProposalID.String(),
			"boName":         p.BOName,
			"fieldName":      p.FieldName,
			"proposedColumn": p.ProposedColumn,
			"confidence":     p.Confidence,
		})
	}
	return results, nil
}
