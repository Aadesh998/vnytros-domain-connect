package netcheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// PropagationNode is one resolver's view of the record.
type PropagationNode struct {
	ResolverNode
	Values    []string `json:"values"`
	TTL       uint32   `json:"ttl"`
	Rcode     string   `json:"rcode"`
	LatencyMS int64    `json:"latency_ms"`
	Error     string   `json:"error,omitempty"`

	GroupID int `json:"group_id"`
}

// PropagationGroup is a distinct answer set and the resolvers serving it.
type PropagationGroup struct {
	ID        int      `json:"id"`
	Values    []string `json:"values"`
	Count     int      `json:"count"`
	Resolvers []string `json:"resolvers"`
	MaxTTL    uint32   `json:"max_ttl"`
}

// PropagationResult is the raw payload attached to the report.
type PropagationResult struct {
	RecordType string             `json:"record_type"`
	Queried    int                `json:"queried"`
	Responded  int                `json:"responded"`
	Consistent bool               `json:"consistent"`
	Groups     []PropagationGroup `json:"groups"`
	Nodes      []PropagationNode  `json:"nodes"`

	Authoritative []string `json:"authoritative,omitempty"`

	AuthoritativeVaries bool `json:"authoritative_varies"`
}

// Propagation queries the same record against every resolver in
// PublicResolvers simultaneously and reports whether they agree.
func Propagation(ctx context.Context, r *Resolver, domain, recordType string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	qtype, ok := TypeFromString(recordType)
	if !ok {
		return nil, fmt.Errorf("%w: unknown record type %q", ErrInvalidTarget, recordType)
	}
	typeName := dns.TypeToString[qtype]

	b := NewReport("dns_propagation", target)

	nodes := make([]PropagationNode, len(PublicResolvers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 12)

	for i, node := range PublicResolvers {
		wg.Add(1)
		go func(i int, node ResolverNode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			out := PropagationNode{ResolverNode: node}

			qctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			ans, err := r.QueryAt(qctx, node.IP, target, qtype, QueryOptions{Timeout: 4 * time.Second})
			if ans != nil {
				out.Rcode = ans.RcodeText
				out.LatencyMS = ans.LatencyMS
				out.TTL = ans.TTL
				out.Values = normalizeValues(ans.Values)
			}
			if err != nil {
				out.Error = shortErr(err)
			}
			nodes[i] = out
		}(i, node)
	}
	wg.Wait()

	groupIDs := map[string]int{}
	var groups []PropagationGroup
	responded := 0

	for i := range nodes {
		n := &nodes[i]
		if n.Error != "" && len(n.Values) == 0 {
			n.GroupID = -1
			continue
		}
		responded++
		key := fingerprint(n.Values)
		id, seen := groupIDs[key]
		if !seen {
			id = len(groups)
			groupIDs[key] = id
			groups = append(groups, PropagationGroup{ID: id, Values: n.Values})
		}
		n.GroupID = id
		groups[id].Count++
		groups[id].Resolvers = append(groups[id].Resolvers, n.Name)
		if n.TTL > groups[id].MaxTTL {
			groups[id].MaxTTL = n.TTL
		}
	}

	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Count > groups[j].Count })

	result := &PropagationResult{
		RecordType: typeName,
		Queried:    len(nodes),
		Responded:  responded,
		Groups:     groups,
		Nodes:      nodes,
		Consistent: len(groups) <= 1,
	}

	result.Authoritative, result.AuthoritativeVaries = authoritativeAnswer(ctx, r, target, qtype)
	b.Raw(result)

	if responded == 0 {
		return b.BuildError("no_response", "No resolver responded",
			"None of the "+itoa(len(nodes))+" public resolvers returned an answer. The domain may not exist, or outbound DNS from this server is blocked."), nil
	}

	if len(groups) == 1 && len(groups[0].Values) == 0 {
		b.Fail("record_missing", "Record not found", SeverityHigh,
			fmt.Sprintf("No %s record for %s on any of the %d resolvers that answered.", typeName, target, responded),
			fmt.Sprintf("Add a %s record for %s at your DNS provider.", typeName, target))
		b.Summary(fmt.Sprintf("No %s record published", typeName))
		return b.Build(), nil
	}

	if len(result.Authoritative) > 0 {
		b.Info("authoritative", "Authoritative answer",
			fmt.Sprintf("The nameservers for this zone currently return %s. This is the value every resolver converges on once its cached copy expires.",
				strings.Join(result.Authoritative, ", ")))
	}

	switch {
	case len(groups) == 1:
		b.Pass("consistent", "Fully propagated",
			fmt.Sprintf("All %d responding resolvers return the same %s record.", responded, typeName))
		b.Summary(fmt.Sprintf("Fully propagated across %d/%d resolvers", responded, len(nodes)))

	default:
		majority := groups[0]
		var maxStale uint32
		for _, g := range groups[1:] {
			if g.MaxTTL > maxStale {
				maxStale = g.MaxTTL
			}
		}

		if result.AuthoritativeVaries {
			b.Info("load_balanced", "Multiple answers by design",
				fmt.Sprintf("%d different answer sets across %d resolvers. The authoritative nameservers also return varying answers, so this is geographic load balancing or round-robin DNS rather than an incomplete change - there is nothing to wait for.", len(groups), responded))
			b.Summary(fmt.Sprintf("%d answer sets across %d resolvers (load balanced)", len(groups), responded))
			break
		}

		detail := fmt.Sprintf("%d distinct answers across %d resolvers. The largest group (%d resolvers) returns %s.",
			len(groups), responded, majority.Count, strings.Join(majority.Values, ", "))

		if len(result.Authoritative) > 0 {
			want := fingerprint(result.Authoritative)
			matched := 0
			for _, g := range groups {
				if fingerprint(g.Values) == want {
					matched += g.Count
				}
			}
			detail = fmt.Sprintf("%d of %d resolvers already return the current authoritative answer (%s); %d return something else.",
				matched, responded, strings.Join(result.Authoritative, ", "), responded-matched)
		}

		cause := "Either the record changed recently and the remaining resolvers still hold a cached copy, or this domain intentionally serves different answers by region."
		if maxStale > 0 {
			cause += fmt.Sprintf(" Any cached copies expire within %s.", humanDuration(maxStale))
		}
		b.Info("inconsistent", "Resolvers return different answers", detail+" "+cause)
		b.Summary(fmt.Sprintf("%d different answers across %d resolvers", len(groups), responded))
	}

	if failed := len(nodes) - responded; failed > 0 {
		b.Info("unreachable", "Some resolvers did not answer",
			fmt.Sprintf("%d of %d resolvers timed out or refused the query. This usually reflects network filtering rather than a problem with the domain.", failed, len(nodes)))
	}

	if len(groups) > 0 && groups[0].MaxTTL > 86400 {
		b.Warn("high_ttl", "Very long TTL", SeverityLow,
			fmt.Sprintf("The record carries a TTL of %s.", humanDuration(groups[0].MaxTTL)),
			"Lower the TTL to 300-600 seconds a day before any planned DNS change, then raise it again afterwards.")
	}

	return b.Build(), nil
}

// authoritativeAnswer asks the zone's own nameservers what the record is,
// bypassing every cache. It returns the answer and whether the authoritative
// servers disagreed among themselves, which is the fingerprint of a traffic
// manager rather than of an incomplete change.
func authoritativeAnswer(ctx context.Context, r *Resolver, target string, qtype uint16) ([]string, bool) {
	hosts, _, err := r.AuthoritativeServers(ctx, target)
	if err != nil || len(hosts) == 0 {
		return nil, false
	}

	if len(hosts) > 4 {
		hosts = hosts[:4]
	}

	var mu sync.Mutex
	seen := map[string][]string{}
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			ips := r.HostIPs(ctx, host)
			if len(ips) == 0 {
				return
			}
			qctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			ans, err := r.QueryAt(qctx, ips[0], target, qtype, QueryOptions{NoRecursion: true, Timeout: 4 * time.Second})
			if err != nil || ans == nil || len(ans.Values) == 0 {
				return
			}
			v := normalizeValues(ans.Values)
			mu.Lock()
			seen[fingerprint(v)] = v
			mu.Unlock()
		}(host)
	}
	wg.Wait()

	if len(seen) == 0 {
		return nil, false
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return seen[keys[0]], len(seen) > 1
}

// fingerprint hashes a sorted answer set so equal answers group together
// regardless of the order the resolver returned them in.
func fingerprint(values []string) string {
	if len(values) == 0 {
		return "empty"
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "|")))
	return hex.EncodeToString(sum[:8])
}

func normalizeValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.ToLower(strings.TrimSpace(v)))
	}
	sort.Strings(out)
	return out
}

func humanDuration(seconds uint32) string {
	d := time.Duration(seconds) * time.Second
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%.1f days", d.Hours()/24)
	case d >= time.Hour:
		return fmt.Sprintf("%.0f hours", d.Hours())
	case d >= time.Minute:
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	default:
		return fmt.Sprintf("%d seconds", seconds)
	}
}

func shortErr(err error) string {
	s := err.Error()
	s = strings.TrimPrefix(s, "netcheck: ")
	if i := strings.Index(s, ": "); i > 0 && len(s) > 90 {
		return s[:i]
	}
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
