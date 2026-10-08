package tenantplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	discoveryPath       = "/.well-known/openid-configuration"
	maxDiscoveryBytes   = 64 << 10
	defaultProbeTimeout = 10 * time.Second
)

// HTTPIdentityProbe fetches the realm's OIDC discovery document and checks
// its issuer.
type HTTPIdentityProbe struct {
	Client *http.Client
}

// NewHTTPIdentityProbe returns a probe with a bounded timeout. Redirects are
// refused, so the check cannot be sent to a host other than the recorded issuer.
func NewHTTPIdentityProbe() *HTTPIdentityProbe {
	return &HTTPIdentityProbe{Client: &http.Client{
		Timeout: defaultProbeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

// CheckIssuer returns nil only when the discovery document at the issuer's
// well-known path returns 200 and names exactly this issuer.
func (p *HTTPIdentityProbe) CheckIssuer(ctx context.Context, issuer string) error {
	if p.Client == nil {
		return errors.New("identity probe has no HTTP client")
	}
	if _, err := url.ParseRequestURI(issuer); err != nil {
		return errors.New("issuer is not a valid URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(issuer, "/")+discoveryPath, nil)
	if err != nil {
		return errors.New("discovery request could not be built")
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return errors.New("discovery document could not be fetched")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery document returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryBytes+1))
	if err != nil {
		return errors.New("discovery document could not be read")
	}
	if len(body) > maxDiscoveryBytes {
		return errors.New("discovery document is too large")
	}
	var doc struct {
		Issuer string `json:"issuer"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return errors.New("discovery document is not valid JSON")
	}
	if doc.Issuer != issuer {
		return errors.New("discovery issuer does not match the recorded issuer")
	}
	return nil
}
