package mcp

import (
	"context"
	"strings"

	"domain-connect-backend/internal/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PropagationArgs struct {
	Domain string `json:"domain" jsonschema:"The domain or hostname to check (e.g. example.com)."`
	Type   string `json:"type,omitempty" jsonschema:"DNS record type to query: A, AAAA, MX, TXT, NS, CNAME, SOA or CAA. Defaults to A."`
}

type HostArgs struct {
	Domain string `json:"domain" jsonschema:"The domain or hostname to inspect (e.g. example.com)."`
}

type TLSArgs struct {
	Domain string `json:"domain" jsonschema:"The hostname whose TLS certificate should be inspected (e.g. example.com)."`
	Port   int    `json:"port,omitempty" jsonschema:"TCP port to connect to. Defaults to 443."`
}

type URLArgs struct {
	URL string `json:"url" jsonschema:"The URL to fetch (e.g. https://example.com). A bare hostname is treated as https."`
}

type DKIMArgs struct {
	Domain    string `json:"domain" jsonschema:"The domain whose DKIM keys should be inspected (e.g. example.com)."`
	Selectors string `json:"selectors,omitempty" jsonschema:"Optional comma-separated DKIM selectors to check (e.g. google,selector1). When omitted, the selectors used by the major mail providers are probed."`
}

type TargetArgs struct {
	Target string `json:"target" jsonschema:"A hostname or a bare IP address to look up."`
}

var netToolDocs = []ToolDoc{
	{
		Name:        "dns_propagation",
		Description: "Query a DNS record against 25 public resolvers worldwide and report which have picked up a change, including the TTL governing how long the rest keep the old value and what the authoritative nameservers currently answer.",
		Params:      []string{"domain", "type?"},
	},
	{
		Name:        "dns_nameservers",
		Description: "Compare the nameserver delegation held by the registrar against the zone's own NS records, then query every authoritative server directly and compare SOA serials to detect broken or lagging zone transfers.",
		Params:      []string{"domain"},
	},
	{
		Name:        "dns_cname_chain",
		Description: "Follow a CNAME chain hop by hop, detecting loops, illegal apex CNAMEs, dead ends and records that unlawfully coexist with the alias.",
		Params:      []string{"domain"},
	},
	{
		Name:        "dns_caa",
		Description: "Report which certificate authorities are permitted to issue certificates for a domain, following CAA inheritance up the parent zones exactly as a CA does.",
		Params:      []string{"domain"},
	},
	{
		Name:        "dns_soa",
		Description: "Inspect a zone's start-of-authority record: refresh, retry and expire timers, the negative caching TTL, and the serial number format.",
		Params:      []string{"domain"},
	},
	{
		Name:        "dns_dnssec",
		Description: "Validate a zone's DNSSEC chain from the parent's DS record through the DNSKEY set to the signatures, catching half-configured states that make a domain unresolvable for validating resolvers only.",
		Params:      []string{"domain"},
	},
	{
		Name:        "email_authentication",
		Description: "Run SPF, DKIM, DMARC and MX together for a domain and report the cross-checks between them, such as an enforcing DMARC policy over a domain with no working SPF or DKIM - a state in which the domain instructs receivers to discard its own legitimate mail. This is the tool to reach for when asked whether a domain's email is set up correctly or why its mail lands in spam.",
		Params:      []string{"domain", "selectors?"},
	},
	{
		Name:        "email_spf",
		Description: "Expand a domain's entire SPF include tree and count the DNS lookups a receiving mail server would spend, catching records that exceed the ten-lookup limit of RFC 7208 and therefore fail authentication despite listing the correct senders. Also reports the all qualifier, void lookups and dangling includes.",
		Params:      []string{"domain"},
	},
	{
		Name:        "email_dkim",
		Description: "Discover a domain's DKIM keys by probing the selectors the major mail providers use, then validate each published key: algorithm, key length against RFC 8301, revocation and testing mode. Specific selectors can be supplied when they are known.",
		Params:      []string{"domain", "selectors?"},
	},
	{
		Name:        "email_dmarc",
		Description: "Parse a domain's DMARC policy, follow inheritance from the organisational domain for subdomains, and verify that external report destinations have published the authorisation record without which every aggregate report is silently discarded.",
		Params:      []string{"domain"},
	},
	{
		Name:        "domain_registration",
		Description: "Look up a domain's registration record over RDAP: registrar, creation and expiry dates, nameservers, DNSSEC delegation and registry status codes translated into plain English.",
		Params:      []string{"domain"},
	},
	{
		Name:        "tls_certificate",
		Description: "Inspect a host's TLS certificate and configuration: expiry, chain completeness, hostname coverage, key strength, signature algorithm and which TLS versions the server accepts.",
		Params:      []string{"domain", "port?"},
	},
	{
		Name:        "http_security_headers",
		Description: "Fetch a URL and grade its HTTP security headers, parsing the Content-Security-Policy for real weaknesses rather than merely checking the header exists.",
		Params:      []string{"url"},
	},
	{
		Name:        "http_redirects",
		Description: "Walk a URL's redirect chain hop by hop with status codes and timings, detecting loops and chains that downgrade to plain HTTP part way through.",
		Params:      []string{"url"},
	},
	{
		Name:        "ip_info",
		Description: "Resolve a hostname and describe each address behind it: network operator and AS number, routing prefix, country, reverse DNS and IPv6 availability.",
		Params:      []string{"target"},
	},
}

func RegisterNetTools(s *mcp.Server, nc *service.NetcheckService) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_propagation",
		Description: describeNet("dns_propagation"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args PropagationArgs) (*mcp.CallToolResult, any, error) {
		recordType := args.Type
		if recordType == "" {
			recordType = "A"
		}
		return reportResult(nc.Propagation(ctx, args.Domain, recordType))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_nameservers",
		Description: describeNet("dns_nameservers"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.Nameservers(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_cname_chain",
		Description: describeNet("dns_cname_chain"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.CNAME(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_caa",
		Description: describeNet("dns_caa"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.CAA(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_soa",
		Description: describeNet("dns_soa"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.SOA(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dns_dnssec",
		Description: describeNet("dns_dnssec"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.DNSSEC(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "email_authentication",
		Description: describeNet("email_authentication"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DKIMArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.EmailAuth(ctx, args.Domain, splitSelectors(args.Selectors)))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "email_spf",
		Description: describeNet("email_spf"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.SPF(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "email_dkim",
		Description: describeNet("email_dkim"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DKIMArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.DKIM(ctx, args.Domain, splitSelectors(args.Selectors)))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "email_dmarc",
		Description: describeNet("email_dmarc"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.DMARC(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_registration",
		Description: describeNet("domain_registration"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HostArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.DomainInfo(ctx, args.Domain))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "tls_certificate",
		Description: describeNet("tls_certificate"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TLSArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.TLS(ctx, args.Domain, args.Port))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "http_security_headers",
		Description: describeNet("http_security_headers"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args URLArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.Headers(ctx, args.URL))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "http_redirects",
		Description: describeNet("http_redirects"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args URLArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.Redirects(ctx, args.URL))
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ip_info",
		Description: describeNet("ip_info"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TargetArgs) (*mcp.CallToolResult, any, error) {
		return reportResult(nc.IPInfo(ctx, args.Target))
	})
}

// splitSelectors turns the comma-separated selector argument into a list.
func splitSelectors(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func reportResult(report any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return errResult(err.Error()), nil, nil
	}
	res, marshalErr := jsonResult(report)
	return res, nil, marshalErr
}

// NetToolDocs returns documentation for the network tools.
func NetToolDocs() []ToolDoc { return netToolDocs }

func describeNet(name string) string {
	for _, d := range netToolDocs {
		if d.Name == name {
			return d.Description
		}
	}
	return ""
}
