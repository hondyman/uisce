package featureflag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Provider abstracts a feature-flag backend (LaunchDarkly, Unleash, etc.)
type Provider interface {
	// IsEnabled returns true if the named flag is on for the given tenant.
	IsEnabled(ctx context.Context, flagKey, tenantID string) bool
}

// NewProvider returns the configured provider based on FEATURE_FLAG_PROVIDER env var.
// Supported: "launchdarkly", "unleash", "static" (always-on/always-off).
func NewProvider() Provider {
	switch os.Getenv("FEATURE_FLAG_PROVIDER") {
	case "launchdarkly":
		return newLaunchDarklyProvider()
	case "unleash":
		return newUnleashProvider()
	default:
		return &staticProvider{enabled: os.Getenv("FEATURE_FLAG_STATIC_DEFAULT") == "true"}
	}
}

// staticProvider is a no-op provider useful for local dev or CI.
type staticProvider struct{ enabled bool }

func (p *staticProvider) IsEnabled(_ context.Context, _, _ string) bool { return p.enabled }

// launchDarklyProvider queries LD REST API (or SDK relay).
type launchDarklyProvider struct {
	baseURL string
	sdkKey  string
	client  *http.Client
	cache   sync.Map // flagKey:tenantID -> bool
}

func newLaunchDarklyProvider() *launchDarklyProvider {
	return &launchDarklyProvider{
		baseURL: os.Getenv("LAUNCHDARKLY_BASE_URL"),
		sdkKey:  os.Getenv("LAUNCHDARKLY_SDK_KEY"),
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

func (p *launchDarklyProvider) IsEnabled(ctx context.Context, flagKey, tenantID string) bool {
	cacheKey := flagKey + ":" + tenantID
	if v, ok := p.cache.Load(cacheKey); ok {
		return v.(bool)
	}
	url := fmt.Sprintf("%s/sdk/evalx/users/%s/flags/%s", p.baseURL, tenantID, flagKey)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", p.sdkKey)
	resp, err := p.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Value bool `json:"value"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	p.cache.Store(cacheKey, body.Value)
	return body.Value
}

// unleashProvider queries Unleash HTTP API.
type unleashProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
	cache   sync.Map
}

func newUnleashProvider() *unleashProvider {
	return &unleashProvider{
		baseURL: os.Getenv("UNLEASH_BASE_URL"),
		apiKey:  os.Getenv("UNLEASH_API_KEY"),
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

func (p *unleashProvider) IsEnabled(ctx context.Context, flagKey, tenantID string) bool {
	cacheKey := flagKey + ":" + tenantID
	if v, ok := p.cache.Load(cacheKey); ok {
		return v.(bool)
	}
	url := fmt.Sprintf("%s/api/client/features/%s", p.baseURL, flagKey)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", p.apiKey)
	req.Header.Set("UNLEASH-APPNAME", "cube-semantic")
	req.Header.Set("UNLEASH-INSTANCEID", tenantID)
	resp, err := p.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	p.cache.Store(cacheKey, body.Enabled)
	return body.Enabled
}
