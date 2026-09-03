package semantic_bridge

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// dialGuard returns a DialContext that rejects connections to
// loopback/private/link-local/multicast/unspecified addresses — including
// the cloud metadata endpoint (169.254.169.254) — no matter what hostname
// resolves to them.
//
// This matters because target host/account values (config["host"],
// config["account"]) come straight from user input in CreateOrUpdateTarget,
// and Push() makes a real outbound HTTP request to them. Without this, a
// tenant could register a "Databricks target" pointing at an internal
// address or the metadata service and use this server as an SSRF proxy,
// with the response coming back in the sync log. Checking at dial time
// (rather than just validating the hostname string up front) also closes
// the DNS-rebinding gap: the check runs against the address actually being
// connected to, after resolution.
func dialGuard(dial func(ctx context.Context, network, addr string) (net.Conn, error)) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("ssrf guard: could not resolve %q: %w", host, err)
		}
		for _, ip := range ips {
			if isBlockedTarget(ip.IP) {
				return nil, fmt.Errorf("ssrf guard: refusing to connect to %s (%s) — private/loopback/link-local/metadata address", host, ip.IP)
			}
		}
		return dial(ctx, network, addr)
	}
}

func isBlockedTarget(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	// Cloud metadata endpoints (AWS/GCP/Azure/Databricks-on-cloud all use
	// this well-known link-local address; IsLinkLocalUnicast() already
	// covers 169.254.0.0/16, but spell it out for clarity/defense-in-depth).
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return true
	}
	return false
}

// validateHostable does a cheap up-front sanity check before we even build
// the request, so obviously-bad config (empty, a URL instead of a bare
// host, etc.) fails fast with a clear message rather than a confusing dial
// error.
func validateHostable(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host/account must not be empty")
	}
	if strings.Contains(host, "://") {
		return fmt.Errorf("host/account must be a bare hostname, not a URL (%q)", host)
	}
	if strings.ContainsAny(host, "/ \t\n") {
		return fmt.Errorf("host/account must not contain a path or whitespace (%q)", host)
	}
	return nil
}
