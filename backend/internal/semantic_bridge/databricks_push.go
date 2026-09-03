package semantic_bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DatabricksPusher executes generated Unity Catalog DDL against a real
// Databricks SQL warehouse via the Statement Execution REST API
// (POST /api/2.0/sql/statements), so "sync" actually reaches Databricks
// instead of only compiling metadata locally.
type DatabricksPusher struct {
	httpClient *http.Client
}

func NewDatabricksPusher() *DatabricksPusher {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{DialContext: dialGuard(dialer.DialContext)}
	return &DatabricksPusher{httpClient: &http.Client{Timeout: 60 * time.Second, Transport: transport}}
}

// PushResult carries what actually happened on the wire, so callers can
// write a real HTTP status/response into the audit ledger instead of a
// hardcoded 200.
type PushResult struct {
	Success      bool
	HTTPStatus   int
	ResponseBody string
}

// Push executes each statement in ddlScript against the configured
// Databricks SQL warehouse. config must contain "host" (e.g.
// "dbc-xxxx.cloud.databricks.com") and "warehouse_id"; creds must contain
// "token" (a Databricks personal access token) — both decrypted by the
// caller via CredentialVault before this is called.
func (p *DatabricksPusher) Push(ctx context.Context, config map[string]interface{}, creds map[string]string, ddlScript string) (*PushResult, error) {
	host, _ := config["host"].(string)
	warehouseID, _ := config["warehouse_id"].(string)
	token := creds["token"]

	if host == "" || warehouseID == "" {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: "missing required config: host and warehouse_id"}, nil
	}
	if token == "" {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: "missing required credential: token"}, nil
	}
	if err := validateHostable(host); err != nil {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: err.Error()}, nil
	}

	statements := splitStatements(ddlScript)
	var lastStatus int
	var lastBody strings.Builder

	for _, stmt := range statements {
		body, _ := json.Marshal(map[string]interface{}{
			"warehouse_id": warehouseID,
			"statement":    stmt,
			"wait_timeout": "30s",
		})

		url := fmt.Sprintf("https://%s/api/2.0/sql/statements", strings.TrimSuffix(host, "/"))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return &PushResult{Success: false, ResponseBody: err.Error()}, nil
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		lastStatus = resp.StatusCode
		lastBody.WriteString(fmt.Sprintf("[%d] %s\n", resp.StatusCode, truncate(string(respBody), 500)))

		if resp.StatusCode >= 300 {
			return &PushResult{Success: false, HTTPStatus: lastStatus, ResponseBody: lastBody.String()}, nil
		}
	}

	return &PushResult{Success: true, HTTPStatus: lastStatus, ResponseBody: lastBody.String()}, nil
}

func splitStatements(script string) []string {
	raw := strings.Split(script, ";\n")
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		out = append(out, s)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
