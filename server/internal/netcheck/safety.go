package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

var (
	ErrInvalidTarget = errors.New("netcheck: invalid target")

	ErrBlockedTarget = errors.New("netcheck: target resolves to a non-public address")
)

// blockedNets are the ranges the HTTP-facing tools must never connect to.
// Without this guard a user could submit a URL pointing at internal
// infrastructure and use the API as an SSRF proxy; 169.254.169.254 in
// particular is the cloud instance metadata endpoint.
var blockedNets = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
		"::/128",
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

// IsPublicIP reports whether an address is safe for the fetching tools to
// connect to.
func IsPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

// NormalizeDomain accepts the many shapes users paste into a domain box -
// "example.com", "https://example.com/path?q=1", "user@example.com",
// "EXAMPLE.COM.", "example.com:8443" - and reduces them to a lowercase ASCII
// hostname. Unicode domains are converted to punycode so they can be queried.
func NormalizeDomain(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidTarget)
	}

	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("%w: cannot parse URL", ErrInvalidTarget)
		}
		s = u.Host
	} else if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}

	if i := strings.LastIndex(s, "@"); i >= 0 {
		s = s[i+1:]
	}

	if strings.HasPrefix(s, "[") {
		if i := strings.Index(s, "]"); i >= 0 {
			s = s[1:i]
		}
	} else if i := strings.LastIndex(s, ":"); i >= 0 && strings.Count(s, ":") == 1 {
		s = s[:i]
	}

	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return "", fmt.Errorf("%w: empty after normalization", ErrInvalidTarget)
	}

	if net.ParseIP(s) != nil {
		return "", fmt.Errorf("%w: expected a domain name, got an IP address", ErrInvalidTarget)
	}

	ascii, err := idna.Lookup.ToASCII(s)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidTarget, err)
	}
	if !strings.Contains(ascii, ".") {
		return "", fmt.Errorf("%w: %q is not a fully qualified domain", ErrInvalidTarget, input)
	}
	if len(ascii) > 253 {
		return "", fmt.Errorf("%w: name exceeds 253 characters", ErrInvalidTarget)
	}
	for _, label := range strings.Split(ascii, ".") {
		if label == "" || len(label) > 63 {
			return "", fmt.Errorf("%w: invalid label length", ErrInvalidTarget)
		}
	}
	return ascii, nil
}

func NormalizeURL(input string) (*url.URL, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidTarget)
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTarget, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: only http and https are supported", ErrInvalidTarget)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidTarget)
	}
	return u, nil
}

// safeDialContext resolves the target itself and refuses any non-public
// address before connecting. Validating in the dialer rather than up front is
// deliberate: it closes the DNS-rebinding window where a name passes an early
// check and then resolves to an internal address at connect time, and it
// re-runs on every hop of a redirect chain.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			ips = append(ips, a.IP)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses for %s", host)
	}

	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 15 * time.Second}
	var lastErr error
	for _, ip := range ips {
		if !IsPublicIP(ip) {
			lastErr = fmt.Errorf("%w: %s resolves to %s", ErrBlockedTarget, host, ip)
			continue
		}

		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// IsBlocked reports whether an error came from the SSRF guard rather than
// from an ordinary network failure. Callers use it to return "forbidden"
// instead of reporting a generic fetch failure, which would otherwise let a
// caller distinguish internal hosts by their error timing and wording.
func IsBlocked(err error) bool {
	return errors.Is(err, ErrBlockedTarget)
}

// SafeHTTPClient returns a client that refuses to follow redirects (callers
// walk the chain themselves) and cannot be pointed at internal addresses.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext:           safeDialContext,
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
			MaxIdleConnsPerHost:   2,
		},
	}
}
