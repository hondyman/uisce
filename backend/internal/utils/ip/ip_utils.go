package ip

import (
	"net"
	"strconv"
	"strings"
)

// IsIpAllowed checks if a given requestIP matches any of the allowed patterns.
// Patterns can be exact IPv4, wildcards (e.g., 192.168.*.*), or CIDR ranges (e.g., 10.0.0.0/8).
func IsIpAllowed(whitelist []string, requestIP string) bool {
	parsedRequestIP := net.ParseIP(requestIP)
	if parsedRequestIP == nil {
		return false
	}

	for _, pattern := range whitelist {
		if IsIpMatch(pattern, requestIP, parsedRequestIP) {
			return true
		}
	}
	return false
}

// IsIpMatch checks if a single pattern matches the request IP.
// It accepts the string representation of the request IP and the parsed IP for efficiency.
func IsIpMatch(pattern, requestIP string, parsedRequestIP net.IP) bool {
	if strings.Contains(pattern, "/") {
		// CIDR range
		_, cidrNet, err := net.ParseCIDR(pattern)
		if err == nil && cidrNet.Contains(parsedRequestIP) {
			return true
		}
	} else if strings.Contains(pattern, "*") {
		// Wildcard
		if patternsOverlap(pattern, requestIP) {
			return true
		}
	} else {
		// Exact match
		if pattern == requestIP {
			return true
		}
	}
	return false
}

// IsValidIPv4WildcardOrCIDR validates IPv4 dotted (with optional * wildcards) or CIDR
func IsValidIPv4WildcardOrCIDR(s string) bool {
	if strings.Contains(s, "/") {
		parts := strings.Split(s, "/")
		if len(parts) != 2 {
			return false
		}
		base, maskStr := parts[0], parts[1]
		mask, err := strconv.Atoi(maskStr)
		if err != nil || mask < 0 || mask > 32 {
			return false
		}
		_, ok := ipv4ToInt(base)
		return ok
	}
	return isValidIpOrWildcard(s)
}

// PatternsOverlapOrCIDR returns true when two IPv4 patterns (wildcards) or CIDR ranges overlap
func PatternsOverlapOrCIDR(a, b string) bool {
	// If both are pure wildcard/plain IPv4 (no '/'), degrade to legacy check
	if !strings.Contains(a, "/") && !strings.Contains(b, "/") {
		return patternsOverlap(a, b)
	}

	// Convert to numeric ranges [min,max]
	toRange := func(p string) (uint32, uint32, bool) {
		if strings.Contains(p, "/") {
			parts := strings.Split(p, "/")
			base, maskStr := parts[0], parts[1]
			mask, err := strconv.Atoi(maskStr)
			if err != nil || mask < 0 || mask > 32 {
				return 0, 0, false
			}
			baseInt, ok := ipv4ToInt(base)
			if !ok {
				return 0, 0, false
			}
			if mask == 0 {
				return 0, ^uint32(0), true
			}
			hostBits := 32 - mask
			maskInt := ^uint32((1 << hostBits) - 1)
			network := baseInt & maskInt
			broadcast := network | ^maskInt
			return network, broadcast, true
		}
		parts := strings.Split(p, ".")
		if len(parts) != 4 {
			return 0, 0, false
		}
		var minParts, maxParts [4]uint32
		for i := 0; i < 4; i++ {
			if parts[i] == "*" {
				minParts[i], maxParts[i] = 0, 255
				continue
			}
			v, err := strconv.Atoi(parts[i])
			if err != nil || v < 0 || v > 255 {
				return 0, 0, false
			}
			minParts[i], maxParts[i] = uint32(v), uint32(v)
		}
		min := (minParts[0] << 24) | (minParts[1] << 16) | (minParts[2] << 8) | minParts[3]
		max := (maxParts[0] << 24) | (maxParts[1] << 16) | (maxParts[2] << 8) | maxParts[3]
		return min, max, true
	}

	a1, a2, okA := toRange(a)
	b1, b2, okB := toRange(b)
	if !okA || !okB {
		return false
	}
	if a2 < b1 || b2 < a1 {
		return false
	}
	return true
}

// isValidIpOrWildcard validates an IPv4 dotted string where each octet is either a number 0-255 or '*'
func isValidIpOrWildcard(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "*" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

// patternsOverlap returns true when two IPv4 patterns overlap. A pattern is an octet list where
// each octet is either '*' or a numeric string. They overlap when for every octet the tokens
// are compatible (equal or one is '*').
func patternsOverlap(a, b string) bool {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	if len(pa) != 4 || len(pb) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		if pa[i] == "*" || pb[i] == "*" {
			continue
		}
		if pa[i] != pb[i] {
			return false
		}
	}
	return true
}

// ipv4ToInt converts dotted IPv4 to uint32
func ipv4ToInt(s string) (uint32, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, false
	}
	var n uint32
	for _, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || v > 255 {
			return 0, false
		}
		n = (n << 8) | uint32(v)
	}
	return n, true
}
