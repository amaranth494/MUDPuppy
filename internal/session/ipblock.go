package session

import (
	"errors"
	"net"
	"strings"
)

var (
	// ErrPrivateIP indicates the target IP is a private/reserved address
	ErrPrivateIP = errors.New("private addresses not allowed")
	// ErrLocalhost indicates localhost is not allowed
	ErrLocalhost = errors.New("localhost not allowed")
)

// isPrivateIP checks if an IP address is private/reserved
func isPrivateIP(ip net.IP) bool {
	// Check 127.x.x.x (loopback)
	if ip.IsLoopback() {
		return true
	}

	// Check 10.x.x.x (10.0.0.0/8)
	if ip[0] == 10 {
		return true
	}

	// Check 172.16-31.x.x (172.16.0.0/12)
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}

	// Check 192.168.x.x (192.168.0.0/16)
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}

	// Check 169.254.x.x (link-local)
	if ip[0] == 169 && ip[1] == 254 {
		return true
	}

	// Check 0.x.x.x (current network)
	if ip[0] == 0 {
		return true
	}

	return false
}

// isHostnamePrivate checks if a hostname resolves to or contains private patterns
func isHostnamePrivate(hostname string) bool {
	hostnameLower := strings.ToLower(hostname)

	// Block common localhost patterns
	privateHostnames := []string{
		"localhost",
		"local",
		"localhost.localdomain",
	}

	for _, h := range privateHostnames {
		if hostnameLower == h {
			return true
		}
	}

	// Block private-looking hostnames
	if strings.HasSuffix(hostnameLower, ".local") {
		return true
	}

	return false
}

// ValidateHost validates a hostname or IP address for connection
// Returns ErrPrivateIP or ErrLocalhost if the host is blocked
func ValidateHost(host string) error {
	// First check hostname patterns
	if isHostnamePrivate(host) {
		return ErrLocalhost
	}

	// Try to resolve as IP first
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return ErrPrivateIP
		}
		return nil
	}

	// Try to resolve hostname to check if it points to private IP
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS resolution failed - let it through, the actual connection will fail
		// with a more specific error if the host is invalid
		return nil
	}

	// Check all resolved IPs
	for _, resolvedIP := range ips {
		if isPrivateIP(resolvedIP) {
			return ErrPrivateIP
		}
	}

	return nil
}
