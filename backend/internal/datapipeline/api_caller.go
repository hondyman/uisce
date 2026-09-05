package datapipeline

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/apistudio"
	"github.com/hondyman/uisce/backend/internal/secrets"
)

const (
	maxRespBytes = 1 << 20
	maxWorkers   = 10
)

var allowLoopback = os.Getenv("APICALLER_ALLOW_LOOPBACK") == "1"

func getCallerTargetURL() string {
	return os.Getenv("APICALLER_TARGET_BASE_URL")
}

type authConfig struct {
	HeaderName string `json:"header_name,omitempty"`
	Key        string `json:"key,omitempty"`
	Token      string `json:"token,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	TokenURL   string `json:"token_url,omitempty"`
	Scopes     string `json:"scopes,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type oauth2Token struct {
	accessToken string
	expiresAt   time.Time
}

type oauth2Cache struct {
	mu    sync.Mutex
	tokens map[string]*oauth2Token
}

var globalOAuth2Cache = &oauth2Cache{tokens: make(map[string]*oauth2Token)}

func (c *oauth2Cache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if tok, ok := c.tokens[key]; ok && time.Now().Before(tok.expiresAt.Add(-30*time.Second)) {
		return tok.accessToken, true
	}
	return "", false
}

func (c *oauth2Cache) set(key string, token string, expiresIn int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens[key] = &oauth2Token{
		accessToken: token,
		expiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
}

type APICallerTransformer struct {
	APIEndpointID   uuid.UUID              `json:"endpoint_id"`
	RequestTemplate map[string]interface{} `json:"request_template,omitempty"`
	TargetField    string                 `json:"target_field,omitempty"`
	MergeOutput    bool                   `json:"merge_output,omitempty"`
	BaseURL        string                 `json:"-"`

	registry   endpointRegistry
	httpClient *http.Client
	secrets    secrets.Provider
}

type endpointRegistry interface {
	GetEndpoint(ctx context.Context, id uuid.UUID) (*apistudio.APIEndpoint, error)
	LogTelemetry(ctx context.Context, t *apistudio.APITelemetry) error
}

func NewAPICallerTransformer(repo *apistudio.Repository, httpClient *http.Client, sp secrets.Provider) *APICallerTransformer {
	if httpClient == nil {
		transport := &http.Transport{
			DialContext: makeSSRFGuardDialer(),
		}
		httpClient = &http.Client{Transport: transport, Timeout: 30 * time.Second}
	}
	return &APICallerTransformer{
		registry:   repo,
		httpClient: httpClient,
		secrets:    sp,
	}
}

func makeSSRFGuardDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		if !allowLoopback {
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("ssrf guard: could not resolve %q: %w", host, err)
			}
			for _, ip := range ips {
				if isBlockedIP(ip.IP) {
					return nil, fmt.Errorf("ssrf guard: refusing to connect to %s (%s) — private/loopback/link-local/metadata address", host, ip.IP)
				}
			}
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return true
	}
	return false
}

func (t *APICallerTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	if t.APIEndpointID == uuid.Nil {
		return nil, nil, fmt.Errorf("api_caller transformer requires endpoint_id in config; endpoint_url is no longer accepted — register the endpoint in API Studio and reference endpoint_id")
	}
	baseURL := t.BaseURL
	if baseURL == "" {
		baseURL = getCallerTargetURL()
	}
	if baseURL == "" {
		return nil, nil, fmt.Errorf("APICALLER_TARGET_BASE_URL environment variable must be set to enable api_caller transformer")
	}

	endpoint, err := t.registry.GetEndpoint(ctx, t.APIEndpointID)
	if err != nil {
		return nil, nil, fmt.Errorf("api_caller: endpoint %s not found: %w", t.APIEndpointID, err)
	}

	output := make([]PipelineRecord, 0, len(input))
	var mu sync.Mutex
	var errs []string
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	errorsCollected := int32(0)

	for i, record := range input {
		wg.Add(1)
		go func(idx int, rec PipelineRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			recCopy := make(PipelineRecord)
			for k, v := range rec {
				recCopy[k] = v
			}

			httpErr := t.callForRecord(ctx, endpoint, recCopy)
			if httpErr != nil {
				atomic.AddInt32(&errorsCollected, 1)
				mu.Lock()
				errs = append(errs, fmt.Sprintf("row %d: %v", idx+1, httpErr))
				mu.Unlock()
				return
			}

			mu.Lock()
			output = append(output, recCopy)
			mu.Unlock()
		}(i, record)
	}

	wg.Wait()

	if errorsCollected > 0 {
		return output, errs, fmt.Errorf("api_caller: %d record(s) failed: %s", errorsCollected, strings.Join(errs, "; "))
	}
	return output, nil, nil
}

func (t *APICallerTransformer) callForRecord(ctx context.Context, ep *apistudio.APIEndpoint, rec PipelineRecord) error {
	start := time.Now()

	baseURL := t.BaseURL
	if baseURL == "" {
		baseURL = getCallerTargetURL()
	}
	if baseURL == "" {
		return fmt.Errorf("APICALLER_TARGET_BASE_URL environment variable must be set to enable api_caller transformer")
	}
	fullURL := strings.TrimSuffix(baseURL, "/") + ep.Path
	if ep.Path == "" {
		return fmt.Errorf("endpoint path is empty")
	}

	var body []byte
	if t.RequestTemplate != nil {
		body, _ = json.Marshal(t.RequestTemplate)
	} else if ep.Method != "GET" && ep.Method != "HEAD" {
		body, _ = json.Marshal(rec)
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := t.injectAuth(ctx, ep, req); err != nil {
		return fmt.Errorf("inject auth: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logTelemetry(ctx, ep, 0, 0, err.Error())
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	latencyMs := int(time.Since(start).Milliseconds())
	statusCode := resp.StatusCode

	if resp.ContentLength > maxRespBytes {
		t.logTelemetry(ctx, ep, statusCode, latencyMs, fmt.Sprintf("response Content-Length %d exceeds %d bytes", resp.ContentLength, maxRespBytes))
		return fmt.Errorf("response exceeded 1 MiB limit")
	}

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil && readErr != io.EOF {
		errStr := readErr.Error()
		t.logTelemetry(ctx, ep, statusCode, latencyMs, errStr)
		return fmt.Errorf("read response: %w", readErr)
	}
	if int64(len(bodyBytes)) > maxRespBytes {
		t.logTelemetry(ctx, ep, statusCode, latencyMs, fmt.Sprintf("response body %d bytes exceeds %d limit", len(bodyBytes), maxRespBytes))
		return fmt.Errorf("response exceeded 1 MiB limit")
	}

	if statusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d: %s", statusCode, string(bodyBytes))
		t.logTelemetry(ctx, ep, statusCode, latencyMs, errMsg)
		return fmt.Errorf("HTTP %d", statusCode)
	}

	var responseData interface{}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		responseData = string(bodyBytes)
	}

	if t.MergeOutput {
		rec["_api_response"] = responseData
	} else if t.TargetField != "" {
		rec[t.TargetField] = responseData
	} else {
		rec["api_status"] = "success"
		rec["api_status_code"] = statusCode
		rec["api_latency_ms"] = latencyMs
		rec["api_response"] = responseData
	}

	t.logTelemetry(ctx, ep, statusCode, latencyMs, "")
	return nil
}

func (t *APICallerTransformer) injectAuth(ctx context.Context, ep *apistudio.APIEndpoint, req *http.Request) error {
	var cfg authConfig
	if ep.AuthConfig != nil {
		data, err := json.Marshal(ep.AuthConfig)
		if err == nil {
			json.Unmarshal(data, &cfg)
		}
	}

	switch ep.AuthType {
	case "", "none":
		return nil

	case "api_key":
		headerName := cfg.HeaderName
		if headerName == "" {
			headerName = "X-Api-Key"
		}
		key := cfg.Key
		if ep.AuthSecretID != "" && t.secrets != nil {
			key, _ = t.secrets.Get(ctx, ep.AuthSecretID)
		}
		req.Header.Set(headerName, key)
		return nil

	case "bearer":
		token := cfg.Token
		if ep.AuthSecretID != "" && t.secrets != nil {
			token, _ = t.secrets.Get(ctx, ep.AuthSecretID)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil

	case "basic_auth":
		username := cfg.Username
		password := cfg.Password
		if ep.AuthSecretID != "" && t.secrets != nil {
			creds, _ := t.secrets.GetMap(ctx, ep.AuthSecretID)
			if username == "" {
				username = creds["username"]
			}
			if password == "" {
				password = creds["password"]
			}
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+encoded)
		return nil

	case "oauth2_client_credentials":
		cacheKey := ep.AuthSecretID
		if cacheKey == "" {
			cacheKey = ep.ID.String()
		}
		if token, ok := globalOAuth2Cache.get(cacheKey); ok {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}
		tokenURL := cfg.TokenURL
		clientID := cfg.ClientID
		clientSecret := cfg.ClientSecret
		if ep.AuthSecretID != "" && t.secrets != nil {
			creds, _ := t.secrets.GetMap(ctx, ep.AuthSecretID)
			if tokenURL == "" {
				tokenURL = creds["token_url"]
			}
			if clientID == "" {
				clientID = creds["client_id"]
			}
			if clientSecret == "" {
				clientSecret = creds["client_secret"]
			}
		}
		if tokenURL == "" {
			return fmt.Errorf("oauth2_client_credentials requires token_url")
		}
		accessToken, expiresIn, err := t.fetchOAuth2Token(ctx, tokenURL, clientID, clientSecret, cfg.Scopes)
		if err != nil {
			return fmt.Errorf("oauth2 token fetch: %w", err)
		}
		if expiresIn > 0 {
			globalOAuth2Cache.set(cacheKey, accessToken, expiresIn)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return nil

	default:
		return fmt.Errorf("unknown auth_type: %q", ep.AuthType)
	}
}

func (t *APICallerTransformer) fetchOAuth2Token(ctx context.Context, tokenURL, clientID, clientSecret, scopes string) (accessToken string, expiresIn int, err error) {
	body := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
	if scopes != "" {
		body += "&scope=" + scopes
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn  int    `json:"expires_in"`
		TokenType  string `json:"token_type"`
		Error      string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	if result.Error != "" {
		return "", 0, fmt.Errorf("oauth2 error: %s", result.Error)
	}
	return result.AccessToken, result.ExpiresIn, nil
}

func (t *APICallerTransformer) logTelemetry(ctx context.Context, ep *apistudio.APIEndpoint, statusCode, latencyMs int, errMsg string) {
	if t.registry == nil {
		return
	}
	tenantUUID, _ := uuid.Parse(ep.TenantID)
	tm := &apistudio.APITelemetry{
		APIID:        ep.ID,
		Env:          ep.Env,
		TenantID:     &tenantUUID,
		ClientType:   "transformer",
		StatusCode:   statusCode,
		LatencyMs:    latencyMs,
		ErrorMessage: nil,
	}
	if errMsg != "" {
		tm.ErrorMessage = &errMsg
	}
	tm.RequestedAt = time.Now()
	_ = t.registry.LogTelemetry(ctx, tm)
}
