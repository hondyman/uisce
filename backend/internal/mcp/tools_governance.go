package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/agentic"
)

func (s *Server) registerGovernanceTools() {
	s.RegisterTool("search_catalog",
		"Search published Business Objects / catalog entries by name for the authenticated tenant.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search string"},
			},
			"required": []string{"query"},
		},
		s.searchCatalog,
	)
	s.RegisterTool("draft_business_object",
		"Submit a Business Object draft as a maker-checker governance proposal (does not write the BO directly).",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"bo_name":       map[string]string{"type": "string"},
				"bo_key":        map[string]string{"type": "string"},
				"justification": map[string]string{"type": "string"},
				"diff_payload":  map[string]interface{}{"type": "object"},
			},
			"required": []string{"bo_name", "justification"},
		},
		s.draftBusinessObject,
	)
}

func (s *Server) searchCatalog(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(args.Query)
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}
	if s.db == nil {
		return map[string]interface{}{
			"matches": []map[string]string{
				{"id": "offline", "key": q, "display_name": q, "note": "db unavailable"},
			},
		}, nil
	}
	rows, err := s.db.QueryxContext(ctx, `
		SELECT id::text, COALESCE(name,'') AS name, COALESCE(display_name, name, '') AS display_name
		FROM public.business_objects
		WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND (name ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY display_name
		LIMIT 50
	`, tenantID.String(), q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []map[string]interface{}
	for rows.Next() {
		var id, name, display string
		if err := rows.Scan(&id, &name, &display); err != nil {
			return nil, err
		}
		matches = append(matches, map[string]interface{}{"id": id, "key": name, "display_name": display})
	}
	if matches == nil {
		matches = []map[string]interface{}{}
	}
	return map[string]interface{}{"matches": matches, "query": q}, nil
}

func (s *Server) draftBusinessObject(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		BOName        string          `json:"bo_name"`
		BOKey         string          `json:"bo_key"`
		Justification string          `json:"justification"`
		DiffPayload   json.RawMessage `json:"diff_payload"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.BOName) == "" {
		return nil, fmt.Errorf("bo_name is required")
	}
	if strings.TrimSpace(args.Justification) == "" {
		return nil, fmt.Errorf("justification is required")
	}
	payload := args.DiffPayload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	target := args.BOKey
	if target == "" {
		target = args.BOName
	}
	mc := agentic.NewMakerCheckerService(s.db)
	ticketID, err := mc.SubmitAgentProposal(ctx, agentic.ProposalRequest{
		TenantID:   tenantID.String(),
		AgentID:    "MCP-Server",
		TargetBOID: target,
		ActionType: "draft_business_object",
		Payload:    payload,
	})
	if err != nil && ticketID == "" {
		return nil, err
	}
	return map[string]interface{}{
		"status":        "QUEUED_FOR_MAKER_CHECKER_APPROVAL",
		"ticket_id":     ticketID,
		"bo_name":       args.BOName,
		"justification": args.Justification,
		"message": fmt.Sprintf(
			"Drafted BO %q as proposal %s (PENDING_APPROVAL).",
			args.BOName, ticketID,
		),
	}, nil
}
