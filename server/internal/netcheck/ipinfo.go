package netcheck

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// IPDetail describes one address behind a hostname.
type IPDetail struct {
	IP       string `json:"ip"`
	Version  int    `json:"version"`
	PTR      string `json:"ptr,omitempty"`
	ASN      string `json:"asn,omitempty"`
	ASName   string `json:"as_name,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Country  string `json:"country,omitempty"`
	Registry string `json:"registry,omitempty"`
}

// IPInfoResult is the raw payload for the address report.
type IPInfoResult struct {
	Target    string     `json:"target"`
	Addresses []IPDetail `json:"addresses"`
	HasIPv4   bool       `json:"has_ipv4"`
	HasIPv6   bool       `json:"has_ipv6"`
	Networks  []string   `json:"networks"`
	Providers []string   `json:"providers"`
}

// knownNetworks recognises the large hosting and CDN operators from their AS
// name, so the report can say "Cloudflare" rather than "AS13335".
var knownNetworks = map[string]string{
	"CLOUDFLARE":   "Cloudflare",
	"AMAZON":       "AWS",
	"AMAZON-02":    "AWS",
	"AMAZON-AES":   "AWS",
	"GOOGLE":       "Google Cloud",
	"MICROSOFT":    "Azure",
	"FASTLY":       "Fastly",
	"AKAMAI":       "Akamai",
	"DIGITALOCEAN": "DigitalOcean",
	"LINODE":       "Linode",
	"HETZNER":      "Hetzner",
	"OVH":          "OVH",
	"VULTR":        "Vultr",
	"GITHUB":       "GitHub",
	"SHOPIFY":      "Shopify",
	"AUTOMATTIC":   "Automattic",
	"NETLIFY":      "Netlify",
	"VERCEL":       "Vercel",
	"HEROKU":       "Heroku",
	"SUCURI":       "Sucuri",
	"INCAPSULA":    "Imperva",
	"STACKPATH":    "StackPath",
	"BUNNYWAY":     "Bunny CDN",
}

// IPInfo resolves a hostname and describes each address behind it: who
// operates the network, which prefix it sits in, and what its reverse DNS
// says.
func IPInfo(ctx context.Context, r *Resolver, target string) (*Report, error) {
	var label string
	var ips []string

	if ip := net.ParseIP(strings.TrimSpace(target)); ip != nil {
		label = ip.String()
		ips = []string{ip.String()}
	} else {
		host, err := NormalizeDomain(target)
		if err != nil {
			return nil, err
		}
		label = host
		ips = r.HostIPs(ctx, host)
	}

	b := NewReport("ip_info", label)
	result := &IPInfoResult{Target: label}

	if len(ips) == 0 {
		return b.BuildError("no_addresses", "No addresses found",
			fmt.Sprintf("%s has no A or AAAA records, so it does not resolve to any address.", label)), nil
	}

	details := make([]IPDetail, len(ips))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i, ipStr := range ips {
		wg.Add(1)
		go func(i int, ipStr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			details[i] = describeIP(ctx, r, ipStr)
		}(i, ipStr)
	}
	wg.Wait()

	networks := map[string]bool{}
	providers := map[string]bool{}
	for _, d := range details {
		result.Addresses = append(result.Addresses, d)
		if d.Version == 4 {
			result.HasIPv4 = true
		} else {
			result.HasIPv6 = true
		}
		if d.ASN != "" {
			networks[d.ASN] = true
		}
		if p := providerFor(d.ASName); p != "" {
			providers[p] = true
		}
	}
	for n := range networks {
		result.Networks = append(result.Networks, n)
	}
	for p := range providers {
		result.Providers = append(result.Providers, p)
	}
	b.Raw(result)

	if len(result.Providers) > 0 {
		b.Info("provider", "Hosting provider",
			fmt.Sprintf("%s is served from %s.", label, strings.Join(result.Providers, ", ")))
	} else if len(details) > 0 && details[0].ASName != "" {
		b.Info("network", "Network operator",
			fmt.Sprintf("%s is announced by %s (%s).", details[0].IP, details[0].ASName, details[0].ASN))
	}

	b.Pass("resolves", "Address resolution",
		fmt.Sprintf("%s resolves to %d address(es): %s.", label, len(ips), strings.Join(truncateList(ips, 6), ", ")))

	if result.HasIPv6 {
		b.Pass("ipv6", "IPv6 available",
			"The host publishes AAAA records, so IPv6-only clients and mobile networks can reach it directly.")
	} else {
		b.Warn("no_ipv6", "No IPv6 address", SeverityLow,
			fmt.Sprintf("%s publishes only IPv4 addresses.", label),
			"IPv6-only clients reach the site through carrier translation, which adds latency and hides the real client address. Most CDNs and hosts enable IPv6 with a single setting.")
	}

	var missingPTR []string
	for _, d := range details {
		if d.PTR == "" {
			missingPTR = append(missingPTR, d.IP)
		}
	}
	switch {
	case len(missingPTR) == 0:
		b.Pass("ptr", "Reverse DNS configured",
			fmt.Sprintf("Every address has a PTR record, for example %s resolves to %s.", details[0].IP, details[0].PTR))
	case len(missingPTR) == len(details):
		b.Info("no_ptr", "No reverse DNS",
			"None of the addresses has a PTR record. This is normal for web servers behind a CDN, but it matters if this host also sends email - receiving mail servers check it.")
	default:
		b.Info("partial_ptr", "Reverse DNS is incomplete",
			fmt.Sprintf("%d of %d addresses have no PTR record: %s.", len(missingPTR), len(details), strings.Join(missingPTR, ", ")))
	}

	if len(result.Networks) > 1 {
		b.Pass("multi_network", "Addresses span multiple networks",
			fmt.Sprintf("Traffic is spread across %d autonomous systems, so an outage at one operator does not take the host offline.", len(result.Networks)))
	}

	b.Summary(ipSummary(result, details))
	return b.Build(), nil
}

// describeIP gathers reverse DNS and routing origin for one address.
func describeIP(ctx context.Context, r *Resolver, ipStr string) IPDetail {
	d := IPDetail{IP: ipStr, Version: 4}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return d
	}
	if ip.To4() == nil {
		d.Version = 6
	}

	var (
		wg     sync.WaitGroup
		ptr    string
		origin IPDetail
	)
	wg.Add(2)

	go func() {
		defer wg.Done()
		arpa, err := dns.ReverseAddr(ipStr)
		if err != nil {
			return
		}
		if ans, err := r.Query(ctx, strings.TrimSuffix(arpa, "."), dns.TypePTR); err == nil && len(ans.Values) > 0 {
			ptr = ans.Values[0]
		}
	}()

	go func() {
		defer wg.Done()
		name := cymruName(ip)
		if name == "" {
			return
		}
		ans, err := r.Query(ctx, name, dns.TypeTXT)
		if err != nil || len(ans.Values) == 0 {
			return
		}

		fields := splitCymru(ans.Values[0])
		if len(fields) < 4 || fields[0] == "" {
			return
		}
		origin.ASN = "AS" + fields[0]
		origin.Prefix = fields[1]
		origin.Country = fields[2]
		origin.Registry = strings.ToUpper(fields[3])

		asAns, err := r.Query(ctx, origin.ASN+".asn.cymru.com", dns.TypeTXT)
		if err != nil || len(asAns.Values) == 0 {
			return
		}
		if asFields := splitCymru(asAns.Values[0]); len(asFields) >= 5 {
			origin.ASName = asFields[4]
		}
	}()

	wg.Wait()

	d.PTR = ptr
	d.ASN, d.Prefix, d.Country, d.Registry, d.ASName =
		origin.ASN, origin.Prefix, origin.Country, origin.Registry, origin.ASName
	return d
}

// cymruName builds the query name for an address: the octets or nibbles are
// reversed, exactly like a PTR lookup.
func cymruName(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", v4[3], v4[2], v4[1], v4[0])
	}
	arpa, err := dns.ReverseAddr(ip.String())
	if err != nil {
		return ""
	}
	nibbles := strings.TrimSuffix(arpa, ".ip6.arpa.")
	return nibbles + ".origin6.asn.cymru.com"
}

func splitCymru(s string) []string {
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func providerFor(asName string) string {
	upper := strings.ToUpper(asName)
	for key, name := range knownNetworks {
		if strings.Contains(upper, key) {
			return name
		}
	}
	return ""
}

func ipSummary(r *IPInfoResult, details []IPDetail) string {
	stack := "IPv4 only"
	if r.HasIPv4 && r.HasIPv6 {
		stack = "dual stack"
	} else if r.HasIPv6 && !r.HasIPv4 {
		stack = "IPv6 only"
	}
	if len(r.Providers) > 0 {
		return fmt.Sprintf("%d address(es) on %s, %s", len(details), strings.Join(r.Providers, ", "), stack)
	}
	if len(details) > 0 && details[0].ASName != "" {
		return fmt.Sprintf("%d address(es) on %s, %s", len(details), details[0].ASName, stack)
	}
	return fmt.Sprintf("%d address(es), %s", len(details), stack)
}
