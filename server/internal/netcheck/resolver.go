package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var (
	ErrNXDomain = errors.New("netcheck: domain does not exist")

	ErrNoAnswer = errors.New("netcheck: no records of the requested type")

	ErrDNSFailure = errors.New("netcheck: dns query failed")
)

// ResolverNode is one public recursive resolver used by the propagation check.
type ResolverNode struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Location string `json:"location"`
	Country  string `json:"country"`
}

// PublicResolvers is the fan-out set for the propagation checker.
var PublicResolvers = []ResolverNode{
	{"Google", "8.8.8.8", "Global (anycast)", "US"},
	{"Google Secondary", "8.8.4.4", "Global (anycast)", "US"},
	{"Cloudflare", "1.1.1.1", "Global (anycast)", "US"},
	{"Cloudflare Secondary", "1.0.0.1", "Global (anycast)", "US"},
	{"Quad9", "9.9.9.9", "Global (anycast)", "CH"},
	{"Quad9 Unsecured", "9.9.9.10", "Global (anycast)", "CH"},
	{"OpenDNS", "208.67.222.222", "Global (anycast)", "US"},
	{"OpenDNS Secondary", "208.67.220.220", "Global (anycast)", "US"},
	{"Level3 / Lumen", "4.2.2.1", "North America", "US"},
	{"Verisign", "64.6.64.6", "North America", "US"},
	{"UltraDNS", "156.154.70.1", "North America", "US"},
	{"AdGuard", "94.140.14.14", "Europe", "CY"},
	{"Yandex", "77.88.8.8", "Russia", "RU"},
	{"DNS.WATCH", "84.200.69.80", "Germany", "DE"},
	{"Digitalcourage", "5.9.164.112", "Germany", "DE"},
	{"FDN", "80.67.169.12", "France", "FR"},
	{"SafeDNS", "195.46.39.39", "Europe", "RU"},
	{"CleanBrowsing", "185.228.168.9", "Global (anycast)", "US"},
	{"Comodo Secure", "8.26.56.26", "North America", "US"},
	{"Hurricane Electric", "74.82.42.42", "North America", "US"},
	{"Mullvad", "194.242.2.2", "Sweden", "SE"},
	{"CIRA Shield", "149.112.121.10", "Canada", "CA"},
	{"AliDNS", "223.5.5.5", "China", "CN"},
	{"DNSPod", "119.29.29.29", "China", "CN"},
	{"114DNS", "114.114.114.114", "China", "CN"},
}

// primaryResolvers are the trusted recursors used by every non-propagation
// tool. Queried in order, falling through on failure.
var primaryResolvers = []string{"1.1.1.1:53", "8.8.8.8:53", "9.9.9.9:53"}

// QueryOptions tunes a single lookup.
type QueryOptions struct {
	DNSSEC bool

	NoRecursion bool

	Timeout time.Duration
}

// Answer is a full DNS response, not just the record values. The TTL, flags
// and latency are what the propagation, DNSSEC and nameserver tools are
// actually built on, and are precisely what the net.Lookup* helpers discard.
type Answer struct {
	Server        string   `json:"server"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Rcode         int      `json:"rcode"`
	RcodeText     string   `json:"rcode_text"`
	Values        []string `json:"values"`
	TTL           uint32   `json:"ttl"`
	Authoritative bool     `json:"authoritative"`
	Authenticated bool     `json:"authenticated"`
	Truncated     bool     `json:"truncated"`
	LatencyMS     int64    `json:"latency_ms"`

	RRs []dns.RR `json:"-"`

	Authority []dns.RR `json:"-"`
}

// Empty reports whether the response carried no records of the asked-for type.
func (a *Answer) Empty() bool { return a == nil || len(a.RRs) == 0 }

// Resolver issues DNS queries. It is safe for concurrent use.
type Resolver struct {
	udp     *dns.Client
	tcp     *dns.Client
	timeout time.Duration
	servers []string
}

// NewResolver builds a resolver with sensible per-query deadlines. Every tool
// takes one of these so tests can point at a fixture server.
func NewResolver() *Resolver {
	timeout := 3 * time.Second
	return &Resolver{
		udp:     &dns.Client{Net: "udp", Timeout: timeout},
		tcp:     &dns.Client{Net: "tcp", Timeout: timeout},
		timeout: timeout,
		servers: primaryResolvers,
	}
}

// WithServers returns a copy that queries the given servers instead of the
// defaults. Entries may omit the port.
func (r *Resolver) WithServers(servers ...string) *Resolver {
	c := *r
	c.servers = make([]string, 0, len(servers))
	for _, s := range servers {
		c.servers = append(c.servers, withPort(s))
	}
	return &c
}

// Query asks the primary resolvers in order and returns the first usable
// response. A definitive NXDOMAIN stops the walk; transport failures fall
// through to the next server.
func (r *Resolver) Query(ctx context.Context, name string, qtype uint16) (*Answer, error) {
	return r.QueryOpts(ctx, name, qtype, QueryOptions{})
}

// QueryOpts is Query with explicit options.
func (r *Resolver) QueryOpts(ctx context.Context, name string, qtype uint16, opts QueryOptions) (*Answer, error) {
	var lastErr error
	for _, server := range r.servers {
		ans, err := r.QueryAt(ctx, server, name, qtype, opts)
		if err == nil {
			return ans, nil
		}

		if errors.Is(err, ErrNXDomain) || errors.Is(err, ErrNoAnswer) {
			return ans, err
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = ErrDNSFailure
	}
	return nil, lastErr
}

// QueryAt sends one query to one specific server. This is the primitive every
// other function in the package is built from.
func (r *Resolver) QueryAt(ctx context.Context, server, name string, qtype uint16, opts QueryOptions) (*Answer, error) {
	server = withPort(server)
	fqdn := dns.Fqdn(name)

	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, qtype)
	msg.RecursionDesired = !opts.NoRecursion

	msg.SetEdns0(4096, opts.DNSSEC)

	client := r.udp
	if opts.Timeout > 0 {
		c := *client
		c.Timeout = opts.Timeout
		client = &c
	}

	start := time.Now()
	resp, rtt, err := client.ExchangeContext(ctx, msg, server)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDNSFailure, server, err)
	}

	if resp.Truncated {
		tcpClient := r.tcp
		if opts.Timeout > 0 {
			c := *tcpClient
			c.Timeout = opts.Timeout
			tcpClient = &c
		}
		if tcpResp, tcpRTT, tcpErr := tcpClient.ExchangeContext(ctx, msg, server); tcpErr == nil {
			resp, rtt = tcpResp, tcpRTT
		}
	}

	ans := &Answer{
		Server:        server,
		Name:          strings.TrimSuffix(fqdn, "."),
		Type:          dns.TypeToString[qtype],
		Rcode:         resp.Rcode,
		RcodeText:     dns.RcodeToString[resp.Rcode],
		Authoritative: resp.Authoritative,
		Authenticated: resp.AuthenticatedData,
		Truncated:     resp.Truncated,
		LatencyMS:     rtt.Milliseconds(),
		Authority:     resp.Ns,
	}
	if ans.LatencyMS == 0 {
		ans.LatencyMS = time.Since(start).Milliseconds()
	}

	minTTL := uint32(0)
	for _, rr := range resp.Answer {
		if rr.Header().Rrtype != qtype {
			continue
		}
		ans.RRs = append(ans.RRs, rr)
		ans.Values = append(ans.Values, rrValue(rr))
		if ttl := rr.Header().Ttl; minTTL == 0 || ttl < minTTL {
			minTTL = ttl
		}
	}
	ans.TTL = minTTL

	switch {
	case resp.Rcode == dns.RcodeNameError:
		return ans, fmt.Errorf("%w: %s", ErrNXDomain, ans.Name)
	case resp.Rcode != dns.RcodeSuccess:
		return ans, fmt.Errorf("%w: %s returned %s", ErrDNSFailure, server, ans.RcodeText)
	case len(ans.RRs) == 0:
		return ans, fmt.Errorf("%w: %s has no %s record", ErrNoAnswer, ans.Name, ans.Type)
	}
	return ans, nil
}

// AuthoritativeServers returns the nameserver hostnames for the closest
// enclosing zone, walking up the labels until a delegation is found.
func (r *Resolver) AuthoritativeServers(ctx context.Context, name string) ([]string, string, error) {
	labels := strings.Split(strings.TrimSuffix(dns.Fqdn(name), "."), ".")
	for i := 0; i < len(labels)-1; i++ {
		zone := strings.Join(labels[i:], ".")
		ans, err := r.Query(ctx, zone, dns.TypeNS)
		if err == nil && len(ans.Values) > 0 {
			servers := append([]string(nil), ans.Values...)
			sort.Strings(servers)
			return servers, zone, nil
		}
	}
	return nil, "", fmt.Errorf("%w: no nameservers found for %s", ErrDNSFailure, name)
}

// HostIPs resolves a hostname to its A and AAAA addresses.
func (r *Resolver) HostIPs(ctx context.Context, host string) []string {
	var ips []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, qtype := range []uint16{dns.TypeA, dns.TypeAAAA} {
		wg.Add(1)
		go func(qt uint16) {
			defer wg.Done()
			ans, err := r.Query(ctx, host, qt)
			if err != nil || ans == nil {
				return
			}
			mu.Lock()
			ips = append(ips, ans.Values...)
			mu.Unlock()
		}(qtype)
	}
	wg.Wait()
	sort.Strings(ips)
	return ips
}

// rrValue renders a record's payload without the leading header, so answers
// compare cleanly across resolvers regardless of TTL differences.
func rrValue(rr dns.RR) string {
	switch v := rr.(type) {
	case *dns.A:
		return v.A.String()
	case *dns.AAAA:
		return v.AAAA.String()
	case *dns.CNAME:
		return strings.TrimSuffix(v.Target, ".")
	case *dns.NS:
		return strings.TrimSuffix(v.Ns, ".")
	case *dns.PTR:
		return strings.TrimSuffix(v.Ptr, ".")
	case *dns.MX:
		return fmt.Sprintf("%d %s", v.Preference, strings.TrimSuffix(v.Mx, "."))
	case *dns.TXT:
		return strings.Join(v.Txt, "")
	case *dns.SRV:
		return fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, strings.TrimSuffix(v.Target, "."))
	case *dns.CAA:
		return fmt.Sprintf("%d %s %q", v.Flag, v.Tag, v.Value)
	case *dns.SOA:
		return fmt.Sprintf("%s %s %d %d %d %d %d",
			strings.TrimSuffix(v.Ns, "."), strings.TrimSuffix(v.Mbox, "."),
			v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
	case *dns.DS:
		return fmt.Sprintf("%d %d %d %s", v.KeyTag, v.Algorithm, v.DigestType, strings.ToUpper(v.Digest))
	case *dns.DNSKEY:
		return fmt.Sprintf("%d %d %d %s", v.Flags, v.Protocol, v.Algorithm, v.PublicKey)
	default:

		s := rr.String()
		if i := strings.Index(s, "\t"); i >= 0 {
			parts := strings.SplitN(s, "\t", 5)
			if len(parts) == 5 {
				return parts[4]
			}
		}
		return s
	}
}

func withPort(server string) string {
	if server == "" {
		return server
	}
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server
	}
	return net.JoinHostPort(server, "53")
}

// TypeFromString maps a user-supplied record type name to its DNS code.
func TypeFromString(s string) (uint16, bool) {
	if s == "" {
		return dns.TypeA, true
	}
	t, ok := dns.StringToType[strings.ToUpper(strings.TrimSpace(s))]
	return t, ok
}
