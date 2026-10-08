package netcheck

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

// NameserverStatus is one authoritative server's response to a direct query.
type NameserverStatus struct {
	Host          string   `json:"host"`
	IPs           []string `json:"ips"`
	Reachable     bool     `json:"reachable"`
	Authoritative bool     `json:"authoritative"`
	Serial        uint32   `json:"serial"`
	LatencyMS     int64    `json:"latency_ms"`
	Error         string   `json:"error,omitempty"`
}

// NameserverResult is the raw payload for the consistency report.
type NameserverResult struct {
	Zone          string             `json:"zone"`
	ParentZone    string             `json:"parent_zone"`
	DelegationNS  []string           `json:"delegation_ns"`
	ZoneNS        []string           `json:"zone_ns"`
	Servers       []NameserverStatus `json:"servers"`
	SerialsAgree  bool               `json:"serials_agree"`
	DelegationOK  bool               `json:"delegation_matches"`
	DistinctNets  int                `json:"distinct_subnets"`
	Providers     []string           `json:"providers"`
	ObservedRoots []string           `json:"observed_serials,omitempty"`
}

// NameserverCheck compares the nameserver set published by the parent zone
// (the delegation the registrar holds) against the set published inside the
// zone itself, then queries each server directly and compares SOA serials.
func NameserverCheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("nameserver_consistency", target)
	result := &NameserverResult{Zone: target}

	zoneAns, zoneErr := r.Query(ctx, target, dns.TypeNS)
	if zoneErr != nil && (zoneAns == nil || len(zoneAns.Values) == 0) {
		return b.BuildError("no_ns", "No nameservers found",
			fmt.Sprintf("Could not retrieve NS records for %s: %s", target, shortErr(zoneErr))), nil
	}
	result.ZoneNS = normalizeValues(zoneAns.Values)

	result.DelegationNS, result.ParentZone = delegationNS(ctx, r, target)

	if len(result.DelegationNS) > 0 {
		missingAtParent := difference(result.ZoneNS, result.DelegationNS)
		missingInZone := difference(result.DelegationNS, result.ZoneNS)
		result.DelegationOK = len(missingAtParent) == 0 && len(missingInZone) == 0

		switch {
		case result.DelegationOK:
			b.Pass("delegation_match", "Delegation matches zone",
				fmt.Sprintf("The %s parent zone and the zone itself both list the same %d nameservers.", result.ParentZone, len(result.ZoneNS)))
		default:
			var parts []string
			if len(missingInZone) > 0 {
				parts = append(parts, "delegated at the parent but absent from the zone: "+strings.Join(missingInZone, ", "))
			}
			if len(missingAtParent) > 0 {
				parts = append(parts, "listed in the zone but not delegated: "+strings.Join(missingAtParent, ", "))
			}
			b.Fail("delegation_mismatch", "Delegation does not match the zone", SeverityHigh,
				"The parent zone and the zone's own NS records disagree - "+strings.Join(parts, "; ")+".",
				"Update the nameserver list at your registrar so it matches the NS records in the zone. Mismatches usually mean a nameserver migration was only half completed, and resolvers may query servers that no longer serve this zone.")
		}
	} else {
		b.Info("delegation_unknown", "Parent delegation not readable",
			"Could not read the delegation from the parent zone, so only the zone's own NS records were checked.")
	}

	hosts := union(result.ZoneNS, result.DelegationNS)
	result.Providers = distinctProviders(hosts)
	result.Servers = probeNameservers(ctx, r, target, hosts)

	var reachable, authoritative int
	serials := map[uint32][]string{}
	var subnets = map[string]bool{}

	for _, s := range result.Servers {
		if s.Reachable {
			reachable++
			serials[s.Serial] = append(serials[s.Serial], s.Host)
		}
		if s.Authoritative {
			authoritative++
		}
		for _, ip := range s.IPs {
			if n := subnetKey(ip); n != "" {
				subnets[n] = true
			}
		}
	}
	result.DistinctNets = len(subnets)
	result.SerialsAgree = len(serials) <= 1

	if reachable == 0 {
		b.Fail("all_unreachable", "No nameserver responded", SeverityCritical,
			fmt.Sprintf("None of the %d nameservers answered an authoritative query for %s.", len(result.Servers), target),
			"Check that the nameserver hostnames resolve and that the servers are running and reachable on UDP/TCP port 53.")
	} else {
		if unreachable := len(result.Servers) - reachable; unreachable > 0 {
			var down []string
			for _, s := range result.Servers {
				if !s.Reachable {
					down = append(down, s.Host)
				}
			}
			b.Fail("ns_unreachable", "Some nameservers are not responding", SeverityHigh,
				fmt.Sprintf("%d of %d nameservers did not answer: %s.", unreachable, len(result.Servers), strings.Join(down, ", ")),
				"Remove dead nameservers from the delegation, or bring them back online. Resolvers that happen to pick a dead server experience slow or failed lookups even though the other servers are healthy.")
		}

		if notAuth := reachable - authoritative; notAuth > 0 {
			b.Fail("not_authoritative", "A nameserver is not authoritative", SeverityHigh,
				fmt.Sprintf("%d responding nameservers did not set the authoritative answer flag for %s.", notAuth, target),
				"A server listed in the delegation must be configured to serve this zone. A non-authoritative response means the zone is missing from that server's configuration.")
		}

		switch {
		case result.SerialsAgree && reachable > 1:
			for serial := range serials {
				b.Pass("serials_match", "Zone data is in sync",
					fmt.Sprintf("All %d responding nameservers report SOA serial %d.", reachable, serial))
			}
		case !result.SerialsAgree:
			var groups []string
			for serial, hostList := range serials {
				sort.Strings(hostList)
				groups = append(groups, fmt.Sprintf("serial %d on %s", serial, strings.Join(hostList, ", ")))
				result.ObservedRoots = append(result.ObservedRoots, fmt.Sprintf("%d", serial))
			}
			sort.Strings(groups)

			if len(result.Providers) > 1 {
				b.Info("multi_provider_serials", "Serials differ across DNS providers",
					fmt.Sprintf("Nameservers span %d independent providers (%s), so their SOA serials are maintained separately and are not expected to match: %s.",
						len(result.Providers), strings.Join(result.Providers, ", "), strings.Join(groups, "; ")))
				b.Pass("multi_provider", "Multi-provider DNS",
					fmt.Sprintf("The zone is served by %d independent DNS providers, which survives an outage at any one of them.", len(result.Providers)))
			} else {
				b.Fail("serial_mismatch", "Nameservers are serving different zone versions", SeverityCritical,
					"Authoritative servers for the same provider disagree on the SOA serial - "+strings.Join(groups, "; ")+".",
					"Zone transfers between your primary and secondary nameservers are failing or lagging. Until they resync, visitors get different DNS answers depending on which server their resolver happens to query. Check the primary's notify/transfer configuration and the secondaries' transfer logs.")
			}
		}
	}

	switch {
	case len(result.Servers) < 2:
		b.Fail("single_ns", "Only one nameserver", SeverityHigh,
			fmt.Sprintf("%s has a single nameserver. RFC 1034 requires at least two.", target),
			"Add at least one more nameserver. With only one, any outage on that host takes the entire domain offline - web, email and everything else.")
	case result.DistinctNets == 1 && len(subnets) > 0:
		b.Warn("single_subnet", "All nameservers share one network", SeverityMedium,
			fmt.Sprintf("All %d nameservers have addresses in the same /24. They are likely on the same physical network.", len(result.Servers)),
			"Spread nameservers across different networks or providers, so a single network outage cannot take out all of them at once.")
	default:
		b.Pass("redundancy", "Nameserver redundancy looks reasonable",
			fmt.Sprintf("%d nameservers across %d distinct networks.", len(result.Servers), result.DistinctNets))
	}

	b.Raw(result)
	if reachable > 0 {
		b.Summary(fmt.Sprintf("%d/%d nameservers responding, serials %s",
			reachable, len(result.Servers), map[bool]string{true: "in sync", false: "out of sync"}[result.SerialsAgree]))
	}
	return b.Build(), nil
}

// delegationNS reads the NS records the parent zone hands out. It queries a
// parent nameserver directly with recursion disabled, so the delegation
// arrives in the authority section rather than being resolved away.
func delegationNS(ctx context.Context, r *Resolver, domain string) ([]string, string) {
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return nil, ""
	}
	parent := strings.Join(labels[1:], ".")

	parentNS, _, err := r.AuthoritativeServers(ctx, parent)
	if err != nil || len(parentNS) == 0 {
		return nil, parent
	}

	for _, ns := range parentNS {
		ips := r.HostIPs(ctx, ns)
		if len(ips) == 0 {
			continue
		}
		qctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		ans, _ := r.QueryAt(qctx, ips[0], domain, dns.TypeNS, QueryOptions{NoRecursion: true, Timeout: 4 * time.Second})
		cancel()
		if ans == nil {
			continue
		}
		var out []string

		for _, rr := range append(append([]dns.RR{}, ans.Authority...), ans.RRs...) {
			if nsRR, ok := rr.(*dns.NS); ok {
				out = append(out, strings.ToLower(strings.TrimSuffix(nsRR.Ns, ".")))
			}
		}
		if len(out) > 0 {
			sort.Strings(out)
			return dedupe(out), parent
		}
	}
	return nil, parent
}

// probeNameservers asks each authoritative server for the zone's SOA directly,
// with recursion disabled so the AA flag is meaningful.
func probeNameservers(ctx context.Context, r *Resolver, zone string, hosts []string) []NameserverStatus {
	out := make([]NameserverStatus, len(hosts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i, host := range hosts {
		wg.Add(1)
		go func(i int, host string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			st := NameserverStatus{Host: host}
			st.IPs = r.HostIPs(ctx, host)
			if len(st.IPs) == 0 {
				st.Error = "hostname does not resolve"
				out[i] = st
				return
			}

			var ans *Answer
			var lastErr error
		probe:
			for _, ip := range st.IPs {
				for attempt := 0; attempt < 2; attempt++ {
					qctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					a, err := r.QueryAt(qctx, ip, zone, dns.TypeSOA, QueryOptions{NoRecursion: true, Timeout: 5 * time.Second})
					cancel()
					if err == nil || (a != nil && len(a.RRs) > 0) {
						ans = a
						break probe
					}
					lastErr = err
				}
			}
			if ans == nil {
				st.Error = shortErr(lastErr)
				out[i] = st
				return
			}

			st.Reachable = true
			st.Authoritative = ans.Authoritative
			st.LatencyMS = ans.LatencyMS
			for _, rr := range ans.RRs {
				if soa, ok := rr.(*dns.SOA); ok {
					st.Serial = soa.Serial
				}
			}
			out[i] = st
		}(i, host)
	}
	wg.Wait()
	return out
}

// distinctProviders reduces nameserver hostnames to the registrable domains
// operating them, which is how we tell one provider's replica set apart from a
// deliberate multi-provider deployment.
func distinctProviders(hosts []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hosts {
		base, err := publicsuffix.EffectiveTLDPlusOne(strings.TrimSuffix(h, "."))
		if err != nil || base == "" {
			continue
		}
		if !seen[base] {
			seen[base] = true
			out = append(out, base)
		}
	}
	sort.Strings(out)
	return out
}

// subnetKey groups an address by its /24 (v4) or /48 (v6) so we can tell
// whether nameservers are really on separate networks.
func subnetKey(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	return fmt.Sprintf("%x:%x:%x::/48", ip[0:2], ip[2:4], ip[4:6])
}

func difference(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, v := range b {
		set[v] = true
	}
	var out []string
	for _, v := range a {
		if !set[v] {
			out = append(out, v)
		}
	}
	return out
}

func union(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, v := range list {
			if v != "" && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
