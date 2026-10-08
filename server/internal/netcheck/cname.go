package netcheck

import (
	"context"
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

type CNAMEHop struct {
	Position  int    `json:"position"`
	Name      string `json:"name"`
	Target    string `json:"target"`
	TTL       uint32 `json:"ttl"`
	LatencyMS int64  `json:"latency_ms"`
}

type CNAMEResult struct {
	Start       string     `json:"start"`
	Hops        []CNAMEHop `json:"hops"`
	FinalTarget string     `json:"final_target"`
	Addresses   []string   `json:"addresses"`
	IsApex      bool       `json:"is_apex"`
	LoopAt      string     `json:"loop_at,omitempty"`
	TotalMS     int64      `json:"total_ms"`
}

const maxCNAMEHops = 10

func CNAMEChain(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("cname_chain", target)
	result := &CNAMEResult{Start: target}

	if apex, err := publicsuffix.EffectiveTLDPlusOne(target); err == nil && apex == target {
		result.IsApex = true
	}

	current := target
	visited := map[string]bool{target: true}

	for i := 0; i < maxCNAMEHops; i++ {
		ans, err := r.Query(ctx, current, dns.TypeCNAME)
		if err != nil || ans == nil || len(ans.Values) == 0 {
			break
		}
		next := strings.ToLower(strings.TrimSuffix(ans.Values[0], "."))

		result.Hops = append(result.Hops, CNAMEHop{
			Position:  i + 1,
			Name:      current,
			Target:    next,
			TTL:       ans.TTL,
			LatencyMS: ans.LatencyMS,
		})
		result.TotalMS += ans.LatencyMS

		if visited[next] {
			result.LoopAt = next
			break
		}
		visited[next] = true
		current = next
	}

	result.FinalTarget = current
	if result.LoopAt == "" {
		result.Addresses = r.HostIPs(ctx, current)
	}
	b.Raw(result)

	if len(result.Hops) == 0 {
		b.Info("no_cname", "No CNAME record",
			fmt.Sprintf("%s is not an alias. It resolves directly or has no records.", target))
		if addrs := result.Addresses; len(addrs) > 0 {
			b.Pass("resolves", "Name resolves",
				fmt.Sprintf("%s resolves to %s.", target, strings.Join(addrs, ", ")))
		}
		b.Summary("No CNAME chain")
		return b.Build(), nil
	}

	if result.LoopAt != "" {
		b.Fail("loop", "CNAME loop", SeverityCritical,
			fmt.Sprintf("The chain returns to %s, so it never resolves to an address.", result.LoopAt),
			"Break the cycle: one of the names in the chain must point at a real host rather than back into the chain. Clients following this alias fail immediately.")
		b.Summary("CNAME loop detected")
		return b.Build(), nil
	}

	if result.IsApex {
		b.Fail("apex_cname", "CNAME at the zone apex", SeverityCritical,
			fmt.Sprintf("%s is the zone apex and has a CNAME record. RFC 1034 forbids a CNAME alongside the SOA and NS records that every apex must carry.", target),
			"Use an ALIAS, ANAME or flattened-CNAME record if your DNS provider offers one (Cloudflare, Route 53 and DNSimple all do), or point the apex at an A record instead. Some resolvers and many mail servers behave unpredictably against an apex CNAME.")
	}

	switch {
	case len(result.Hops) == 1:
		b.Pass("chain_length", "Direct alias",
			fmt.Sprintf("%s is a single-hop alias for %s.", target, result.FinalTarget))
	case len(result.Hops) <= 3:
		b.Warn("chain_length", "Multi-hop alias chain", SeverityLow,
			fmt.Sprintf("The chain is %d hops long, ending at %s.", len(result.Hops), result.FinalTarget),
			"Each hop is an extra DNS lookup before a client can connect. Point the first name directly at the final target where you can.")
	default:
		b.Fail("chain_length", "Excessively long alias chain", SeverityMedium,
			fmt.Sprintf("The chain is %d hops long before reaching %s.", len(result.Hops), result.FinalTarget),
			"Collapse the chain. Some resolvers stop following after a small number of hops, so long chains fail intermittently and are slow for everyone else.")
	}

	if len(result.Addresses) == 0 {
		b.Fail("dead_end", "Chain does not resolve", SeverityHigh,
			fmt.Sprintf("The chain ends at %s, which has no A or AAAA record.", result.FinalTarget),
			"The final target must resolve to an address. Check that the destination hostname is correct and still exists - this is what happens when a service is decommissioned but the alias pointing at it is left behind.")
	} else {
		b.Pass("resolves", "Chain resolves",
			fmt.Sprintf("%s resolves to %s.", result.FinalTarget, strings.Join(result.Addresses, ", ")))
	}

	for _, qtype := range []uint16{dns.TypeMX, dns.TypeTXT, dns.TypeNS} {
		if ans, err := r.Query(ctx, target, qtype); err == nil && ownedBy(ans.RRs, target) {
			b.Fail("cname_with_other_records", "Other records alongside the CNAME", SeverityHigh,
				fmt.Sprintf("%s has both a CNAME and %s records. RFC 1034 does not allow a CNAME to coexist with other record types.", target, dns.TypeToString[qtype]),
				fmt.Sprintf("Remove either the CNAME or the %s records at this name. Resolvers handle the conflict inconsistently, so the behaviour varies by client.", dns.TypeToString[qtype]))
			break
		}
	}

	b.Summary(fmt.Sprintf("%d-hop chain ending at %s", len(result.Hops), result.FinalTarget))
	return b.Build(), nil
}

func ownedBy(rrs []dns.RR, name string) bool {
	want := dns.Fqdn(strings.ToLower(name))
	for _, rr := range rrs {
		if strings.EqualFold(rr.Header().Name, want) {
			return true
		}
	}
	return false
}
