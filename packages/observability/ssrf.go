// Package observability — SSRF protection for HTTP crawlers.
package observability

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRFAllowlist defines which hosts the platform's crawlers are allowed to
// fetch from. This prevents Server-Side Request Forgery attacks where a
// malicious URL could cause the crawler to access internal services.
//
// The allowlist is deliberately restrictive — only official Kenyan government
// sources and their CDN domains are permitted.
var SSRFAllowlist = []string{
	// Parliament of Kenya
	"parliament.go.ke",
	"www.parliament.go.ke",
	"nationalassembly.go.ke",
	"senate.go.ke",

	// Kenya Law
	"kenyalaw.org",
	"new.kenyalaw.org",

	// President
	"president.go.ke",
	"www.president.go.ke",

	// National Treasury
	"treasury.go.ke",
	"www.treasury.go.ke",

	// Kenya Gazette
	"gazettes.africa",
	"www.gazettes.africa",

	// International financial institutions (for loans/grants tracking)
	"imf.org",
	"www.imf.org",
	"worldbank.org",
	"www.worldbank.org",
	"afdb.org",
	"www.afdb.org",
}

// blockedIPRanges are IP ranges that must NEVER be accessed, even if a host
// resolves to them. This prevents DNS rebinding attacks.
var blockedCIDRs = []string{
	"127.0.0.0/8",     // localhost
	"10.0.0.0/8",      // private
	"172.16.0.0/12",   // private
	"192.168.0.0/16",  // private
	"169.254.0.0/16",  // link-local
	"0.0.0.0/8",       // current network
	"::1/128",         // localhost IPv6
	"fc00::/7",        // IPv6 private
	"fe80::/10",       // IPv6 link-local
}

// IsURLAllowed checks whether a URL's host is in the SSRF allowlist.
// Returns an error if the URL is not allowed or if the host resolves to a
// blocked IP range.
func IsURLAllowed(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme %s not allowed (only http/https)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty host in URL %s", rawURL)
	}

	// Check allowlist.
	allowed := false
	for _, h := range SSRFAllowlist {
		if host == h || strings.HasSuffix(host, "."+h) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("host %s not in SSRF allowlist", host)
	}

	// Check if host resolves to a blocked IP.
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip.String()) {
			return fmt.Errorf("host %s resolves to blocked IP %s", host, ip)
		}
	}

	return nil
}

// isBlockedIP checks whether an IP is in a blocked range.
func isBlockedIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true // can't parse → block
	}
	for _, cidr := range blockedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
