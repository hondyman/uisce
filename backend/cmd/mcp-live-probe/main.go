// Command mcp-live-probe is the flip-checklist initialize-capable probe
// for the streamable MCP surface. It mints via MintDevToken (same path
// as hardened devjwt) and runs initialize → tools/list → tools/call →
// refuse against UISCE_API_URL/api/mcp.
//
// Named receipt item: "devjwt-equivalent live mint+call".
//
// Env: JWT_SECRET (required), UISCE_API_URL (default http://127.0.0.1:8080),
// ENVIRONMENT (unset ok / treated as local; production refused).
// Token never printed; summary lines go to stdout, errors to stderr.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/hondyman/uisce/backend/internal/services"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	if err := requireDevEnvironment(); err != nil {
		return err
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is not set")
	}
	base := strings.TrimSpace(os.Getenv("UISCE_API_URL"))
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	endpoint := strings.TrimRight(base, "/") + "/api/mcp"
	tenant := strings.TrimSpace(os.Getenv("UISCE_PROBE_TENANT"))
	if tenant == "" {
		tenant = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	}

	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    "mcp-live-probe",
		TenantIDs: []string{tenant},
		Roles:     []string{"portfolio_manager"},
	})
	if err != nil {
		return fmt.Errorf("mint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	res, err := mcp.ProbeStreamable(ctx, endpoint, token)
	if err != nil {
		return err
	}
	if res.ToolCount < 7 {
		return fmt.Errorf("tools/list: expected >=7 tools, got %d", res.ToolCount)
	}
	if !strings.Contains(res.CallText, tenant) {
		return fmt.Errorf("tools/call list_pages: missing tenant_id field in result")
	}
	refused := res.RefuseIsErr || strings.Contains(res.RefuseText, "refused") || strings.Contains(res.RefuseText, "unknown")
	if !refused {
		return fmt.Errorf("expected run_sql refusal, got %q", res.RefuseText)
	}

	fmt.Printf("mcp-live-probe ok session=%s tools=%d tenant_in_call=true refuse=true endpoint=%s\n",
		mcp.SessionMode, res.ToolCount, endpoint)
	return nil
}

func requireDevEnvironment() error {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	switch env {
	case "development", "local", "test", "":
		return nil
	default:
		return fmt.Errorf("mcp-live-probe refused: ENVIRONMENT=%q", env)
	}
}
