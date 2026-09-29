package iceberg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type TokenManager struct {
	tokenURL      string
	clientID      string
	clientSecret  string
	httpClient    *http.Client

	mu     sync.RWMutex
	token  string
	expiry time.Time
}

func NewTokenManager(tokenURL, clientID, clientSecret string) *TokenManager {
	return &TokenManager{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *TokenManager) Token(ctx context.Context) (string, error) {
	t.mu.RLock()
	if t.token != "" && time.Until(t.expiry) > 30*time.Second {
		tok := t.token
		t.mu.RUnlock()
		return tok, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" && time.Until(t.expiry) > 30*time.Second {
		return t.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", t.clientID)
	form.Set("client_secret", t.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("token request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token fetch: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("token response decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}

	t.token = tr.AccessToken
	t.expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return t.token, nil
}
