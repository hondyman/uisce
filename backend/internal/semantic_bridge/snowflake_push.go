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

// SnowflakePusher executes generated governance DDL (COMMENT ON TABLE/COLUMN
// statements that Cortex Analyst reads directly from the warehouse) against
// a real Snowflake account via the SQL API v2 (POST /api/v2/statements),
// authenticated with a Programmatic Access Token.
//
// This does NOT upload the Cortex Analyst semantic YAML to an internal
// stage — that upload path needs the Snowflake driver's file-staging
// protocol (PUT), which isn't a dependency of this project. Pushing the
// governance DDL is the real, working subset: it's plain SQL over REST, and
// Cortex Analyst can already read table/column comments as semantic
// context. Stage upload is a follow-up if the full YAML artifact needs to
// live in Snowflake too.
type SnowflakePusher struct {
	httpClient *http.Client
}

func NewSnowflakePusher() *SnowflakePusher {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{DialContext: dialGuard(dialer.DialContext)}
	return &SnowflakePusher{httpClient: &http.Client{Timeout: 60 * time.Second, Transport: transport}}
}

// Push executes each statement in ddlStatements against the configured
// Snowflake account. config must contain "account" (e.g. "xy12345.us-east-1")
// and "warehouse"; creds must contain "token" (a Snowflake Programmatic
// Access Token) — both decrypted by the caller via CredentialVault.
func (p *SnowflakePusher) Push(ctx context.Context, config map[string]interface{}, creds map[string]string, ddlStatements []string) (*PushResult, error) {
	account, _ := config["account"].(string)
	warehouse, _ := config["warehouse"].(string)
	database, _ := config["database"].(string)
	schema, _ := config["schema"].(string)
	token := creds["token"]

	if account == "" || warehouse == "" {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: "missing required config: account and warehouse"}, nil
	}
	if token == "" {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: "missing required credential: token"}, nil
	}
	if err := validateHostable(account); err != nil {
		return &PushResult{Success: false, HTTPStatus: 0, ResponseBody: err.Error()}, nil
	}

	var lastStatus int
	var lastBody strings.Builder

	for _, stmt := range ddlStatements {
		stmt = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(stmt), ";"))
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		payload := map[string]interface{}{
			"statement": stmt,
			"warehouse": warehouse,
			"timeout":   30,
		}
		if database != "" {
			payload["database"] = database
		}
		if schema != "" {
			payload["schema"] = schema
		}
		body, _ := json.Marshal(payload)

		url := fmt.Sprintf("https://%s.snowflakecomputing.com/api/v2/statements", account)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Snowflake-Authorization-Token-Type", "PROGRAMMATIC_ACCESS_TOKEN")

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
