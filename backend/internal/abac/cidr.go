package abac

import "net"

// parseAndMatch parses the IP and CIDR and reports membership.
// Returns true if ip falls within cidr, false otherwise.
// Returns false if either is malformed.
func parseAndMatch(ip, cidr string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.Contains(parsedIP)
}