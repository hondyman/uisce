package mcp

// Owned OMS tool copy (PR C rescue). Source was uncommitted oms_tools.go /
// tool_handler.go from a concurrent session — copied here so the catalog
// cannot vanish with the dirty tree. Do not edit those dirty files.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	fixpkg "github.com/hondyman/uisce/backend/internal/fix"
	"github.com/hondyman/uisce/backend/internal/trading"
)

const (
	omsOwnedListSlug   = "order-list-tl48"
	omsOwnedDetailSlug = "order-detail-tj2e"
	omsOwnedOrderBOID  = "b611af7b-8689-407d-807a-eeb315065e7d"
	omsOwnedFIXQueue   = "bp_queue"
)

// ErrTemporalNotConfigured is the named degradation for start_fix_order_entry
// when Temporal is unset (not a panic).
const ErrTemporalNotConfigured = "Temporal not configured; FIX order entry unavailable"

func (s *Server) registerOMSTools() {
	s.RegisterTool("generate_page_spec",
		"Returns a deterministic Page Studio spec (list or detail) for a Business Object. Does not save. Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"bo_id":     map[string]string{"type": "string"},
				"bo_key":    map[string]string{"type": "string"},
				"bo_name":   map[string]string{"type": "string"},
				"page_kind": map[string]string{"type": "string", "description": "list or detail"},
			},
		},
		s.omsGeneratePageSpec,
	)
	s.RegisterTool("describe_oms_journey",
		"Returns the Northwind Order blotter + ticket slugs, primary BO, and FIX command contract. Presentation only; fills are not written here.",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		func(ctx context.Context, tenantID uuid.UUID, _ json.RawMessage) (interface{}, error) {
			return s.omsDescribeOMSJourney(ctx, tenantID)
		},
	)
	s.RegisterTool("get_bo_schema",
		"Returns form schema for a BO: display labels, enum lists, reference BOs, date defaults. Same shape the Order ticket form uses.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"bo_id":  map[string]string{"type": "string"},
				"bo_key": map[string]string{"type": "string"},
			},
		},
		s.omsGetBOSchema,
	)
	s.RegisterTool("start_fix_order_entry",
		"Starts FIXOrderEntryWorkflow for an existing order (NewOrderSingle, CancelReplace, Cancel). Does not save the page or write fills. Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"order_id": map[string]string{"type": "string"},
				"command":  map[string]string{"type": "string"},
			},
			"required": []string{"order_id"},
		},
		s.omsStartFIXOrderEntry,
	)
}

func (s *Server) omsGeneratePageSpec(_ context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		BOID     string `json:"bo_id"`
		BOKey    string `json:"bo_key"`
		BOName   string `json:"bo_name"`
		PageKind string `json:"page_kind"`
	}
	_ = json.Unmarshal(argsRaw, &args)
	kind := strings.ToLower(strings.TrimSpace(args.PageKind))
	if kind != "detail" {
		kind = "list"
	}
	name := args.BOName
	if name == "" {
		name = args.BOKey
	}
	if name == "" {
		name = "Business Object"
	}
	title := name + " List"
	if kind == "detail" {
		title = name + " Detail"
	}
	spec := map[string]interface{}{
		"tenant_id":    tenantID,
		"pageKind":     kind,
		"title":        title,
		"primaryBoId":  args.BOID,
		"primaryBoKey": args.BOKey,
		"saved":        false,
		"commands":     []string{"NewOrderSingle", "CancelReplace", "Cancel"},
	}
	if kind == "list" {
		spec["layoutTemplate"] = "list-with-filter-bar"
		spec["widgets"] = []string{"Slicer", "Table", "FixCommand"}
		spec["rowClickAction"] = "select"
		spec["recordPageSlug"] = strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-detail"
		spec["note"] = "View/Edit/Add on the Table open the detail ticket. FIX commands start FIXOrderEntryWorkflow."
	} else {
		spec["layoutTemplate"] = "ticket-tabs"
		spec["widgets"] = []string{"FixCommand", "Form", "Table"}
		spec["tabs"] = []string{"Order", "Allocations", "Placements", "Executions"}
		spec["form"] = map[string]interface{}{
			"sections":     []string{"Identity", "Economics", "Dates"},
			"enumFields":   []string{"status", "side", "order_type", "time_in_force"},
			"defaultDates": []string{"trade_date"},
		}
		spec["note"] = "Create mode is /pages/{slug}/new. Related tabs and FIX hide until the order exists."
	}
	return spec, nil
}

func (s *Server) omsDescribeOMSJourney(ctx context.Context, tenantID uuid.UUID) (interface{}, error) {
	out := map[string]interface{}{
		"tenant_id": tenantID,
		"list": map[string]interface{}{
			"slug": omsOwnedListSlug,
			"path": "/pages/" + omsOwnedListSlug,
		},
		"detail": map[string]interface{}{
			"slug":       omsOwnedDetailSlug,
			"path":       "/pages/" + omsOwnedDetailSlug + "/{id}",
			"createPath": "/pages/" + omsOwnedDetailSlug + "/new",
			"viewPath":   "/pages/" + omsOwnedDetailSlug + "/{id}?mode=view",
		},
		"primaryBo": map[string]interface{}{
			"id":  omsOwnedOrderBOID,
			"key": "order",
		},
		"fix": map[string]interface{}{
			"http":      "POST /api/oms/commands/fix-order-entry",
			"mcp":       "start_fix_order_entry",
			"sessionId": fixpkg.DemoSessionID(),
			"commands":  []string{"NewOrderSingle", "CancelReplace", "Cancel"},
			"note":      "Command starts FIXOrderEntryWorkflow. Fills persist to crims.orm, never the page.",
		},
	}
	if s.pages != nil {
		pages, _ := s.pages.ListBySlugs(ctx, tenantID, omsOwnedListSlug, omsOwnedDetailSlug)
		for _, p := range pages {
			if p.Slug == omsOwnedListSlug {
				out["list"] = map[string]interface{}{"id": p.ID, "name": p.Name, "slug": p.Slug, "path": "/pages/" + p.Slug}
			}
			if p.Slug == omsOwnedDetailSlug {
				out["detail"] = map[string]interface{}{
					"id": p.ID, "name": p.Name, "slug": p.Slug,
					"path":       "/pages/" + p.Slug + "/{id}",
					"createPath": "/pages/" + p.Slug + "/new",
					"viewPath":   "/pages/" + p.Slug + "/{id}?mode=view",
				}
			}
		}
	}
	return out, nil
}

func (s *Server) omsGetBOSchema(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		BOID  string `json:"bo_id"`
		BOKey string `json:"bo_key"`
	}
	_ = json.Unmarshal(argsRaw, &args)
	if s.bos == nil {
		return map[string]interface{}{"fields": []interface{}{}}, nil
	}
	boID := args.BOID
	if boID == "" && args.BOKey != "" {
		if id, err := s.bos.ResolveBOIDByKey(ctx, tenantID, args.BOKey); err == nil {
			boID = id
		}
	}
	if boID == "" {
		return map[string]interface{}{"fields": []interface{}{}, "note": "bo_id or bo_key required"}, nil
	}
	fields, err := s.bos.ListFieldSchema(ctx, tenantID, boID)
	if err != nil {
		return map[string]interface{}{"fields": []interface{}{}, "note": err.Error()}, nil
	}
	out := make([]map[string]interface{}, 0, len(fields))
	for _, f := range fields {
		out = append(out, map[string]interface{}{"name": f.Name, "displayName": f.DisplayName, "type": f.DataType})
	}
	return map[string]interface{}{"bo_id": boID, "fields": out}, nil
}

func (s *Server) omsStartFIXOrderEntry(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	if s.temporal == nil {
		return nil, fmt.Errorf("%s", ErrTemporalNotConfigured)
	}
	temporal := s.temporal
	var args struct {
		OrderID string `json:"order_id"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}
	if args.OrderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	cmd := strings.TrimPrefix(args.Command, "fix.")
	if cmd == "" {
		cmd = "NewOrderSingle"
	}
	order, err := trading.LoadOrder(ctx, tenantID, args.OrderID)
	if err != nil {
		return nil, err
	}
	adminURL := os.Getenv("FIX_ADMIN_ADDR")
	if adminURL == "" {
		adminURL = "127.0.0.1:8981"
	}
	if !strings.HasPrefix(adminURL, "http") {
		adminURL = "http://" + adminURL
	}
	short := strings.ReplaceAll(order.OrderID, "-", "")
	if len(short) > 8 {
		short = short[:8]
	}
	clOrdID := fmt.Sprintf("NW-%s-%d", short, time.Now().UTC().UnixNano()%1_000_000)
	input := trading.FIXOrderInput{
		Order:       *order,
		TenantID:    tenantID,
		BrokerCode:  "GSCO",
		AdminURL:    adminURL,
		AdminToken:  os.Getenv("FIX_ADMIN_TOKEN"),
		SessionID:   fixpkg.DemoSessionID(),
		ClOrdID:     clOrdID,
		Command:     cmd,
		PlacementID: uuid.NewString(),
	}
	wfID := "fix-order-" + tenantID.String() + "-" + clOrdID
	run, err := temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        wfID,
		TaskQueue: omsOwnedFIXQueue,
	}, trading.FIXOrderEntryWorkflow, input)
	if err != nil {
		return nil, fmt.Errorf("start FIXOrderEntryWorkflow: %w", err)
	}
	return map[string]interface{}{
		"workflowId": run.GetID(),
		"runId":      run.GetRunID(),
		"clOrdId":    clOrdID,
		"command":    cmd,
		"status":     "started",
	}, nil
}
