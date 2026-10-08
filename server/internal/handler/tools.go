package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/netcheck"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/utils"
)

type ToolsHandler struct {
	service *service.NetcheckService
}

func NewToolsHandler(s *service.NetcheckService) *ToolsHandler {
	return &ToolsHandler{service: s}
}

func (h *ToolsHandler) respond(w http.ResponseWriter, report *netcheck.Report, err error) {
	if err != nil {
		switch {
		case errors.Is(err, netcheck.ErrInvalidTarget):
			errorz.ErrBadRequest.SendError(w)
		case errors.Is(err, netcheck.ErrBlockedTarget):
			errorz.ErrForbidden.SendError(w)
		case errors.Is(err, context.DeadlineExceeded):
			errorz.ErrInternalServer.SendError(w)
		default:
			log.Printf("tools: check failed: %v", err)
			errorz.ErrInternalServer.SendError(w)
		}
		return
	}
	utils.SendSuccess(w, report)
}

func param(r *http.Request, names ...string) string {
	for _, n := range names {
		if v := r.URL.Query().Get(n); v != "" {
			return v
		}
	}
	return ""
}

func (h *ToolsHandler) DNSPropagation(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	recordType := param(r, "type", "record_type")
	if recordType == "" {
		recordType = "A"
	}
	report, err := h.service.Propagation(r.Context(), domain, recordType)
	h.respond(w, report, err)
}

func (h *ToolsHandler) Nameservers(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.Nameservers(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) CNAMEChain(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.CNAME(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) CAA(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.CAA(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) SOA(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.SOA(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) DNSSEC(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.DNSSEC(r.Context(), domain)
	h.respond(w, report, err)
}

// selectorParam reads the optional comma-separated DKIM selector list.
func selectorParam(r *http.Request) []string {
	raw := param(r, "selector", "selectors")
	if raw == "" {
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

func (h *ToolsHandler) SPF(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.SPF(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) DKIM(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.DKIM(r.Context(), domain, selectorParam(r))
	h.respond(w, report, err)
}

func (h *ToolsHandler) DMARC(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.DMARC(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) EmailAuth(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.EmailAuth(r.Context(), domain, selectorParam(r))
	h.respond(w, report, err)
}

func (h *ToolsHandler) DomainInfo(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.DomainInfo(r.Context(), domain)
	h.respond(w, report, err)
}

func (h *ToolsHandler) TLSCertificate(w http.ResponseWriter, r *http.Request) {
	domain := param(r, "domain", "host", "target")
	if domain == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	port := 443
	if p := r.URL.Query().Get("port"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed < 1 || parsed > 65535 {
			errorz.ErrBadRequest.SendError(w)
			return
		}
		port = parsed
	}

	report, err := h.service.TLS(r.Context(), domain, port)
	h.respond(w, report, err)
}

func (h *ToolsHandler) SecurityHeaders(w http.ResponseWriter, r *http.Request) {
	target := param(r, "url", "domain", "target")
	if target == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.Headers(r.Context(), target)
	h.respond(w, report, err)
}

func (h *ToolsHandler) RedirectChain(w http.ResponseWriter, r *http.Request) {
	target := param(r, "url", "domain", "target")
	if target == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.Redirects(r.Context(), target)
	h.respond(w, report, err)
}

func (h *ToolsHandler) IPInfo(w http.ResponseWriter, r *http.Request) {
	target := param(r, "target", "domain", "ip")
	if target == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	report, err := h.service.IPInfo(r.Context(), target)
	h.respond(w, report, err)
}

type ToolDescriptor struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
}

var toolCatalogue = []ToolDescriptor{
	{"dns_propagation", "DNS Propagation Checker", "/v1/tools/dns/propagation", "dns",
		"Query a record against 25 public resolvers worldwide and see which have picked up a change, with the TTL that governs how long the rest will keep the old value.",
		[]string{"domain", "type?"}},
	{"nameserver_consistency", "Nameserver Consistency", "/v1/tools/dns/nameservers", "dns",
		"Compare the delegation held by the registrar against the zone's own NS records, then query every authoritative server directly and compare SOA serials to detect broken zone transfers.",
		[]string{"domain"}},
	{"cname_chain", "CNAME Chain Tracer", "/v1/tools/dns/cname", "dns",
		"Follow an alias chain hop by hop, detecting loops, apex CNAMEs and records that illegally coexist with the alias.",
		[]string{"domain"}},
	{"caa", "CAA Checker", "/v1/tools/dns/caa", "dns",
		"Report which certificate authorities are permitted to issue for a domain, following CAA's inheritance up the parent zones exactly as a CA does.",
		[]string{"domain"}},
	{"soa", "SOA & TTL Analyzer", "/v1/tools/dns/soa", "dns",
		"Inspect the zone's start-of-authority timers, negative caching TTL and serial format, and flag TTLs that would slow a future migration.",
		[]string{"domain"}},
	{"dnssec", "DNSSEC Validator", "/v1/tools/dns/dnssec", "dns",
		"Validate the signing chain from the parent's DS record through the zone's keys to the signatures, catching the half-configured states that make a domain unresolvable for validating resolvers only.",
		[]string{"domain"}},
	{"email_auth", "Email Authentication Report", "/v1/tools/email/auth", "email",
		"Run SPF, DKIM, DMARC and MX together and add the cross-checks between them, such as an enforcing DMARC policy over a domain that has nothing to authenticate with - a state in which the domain rejects its own legitimate mail.",
		[]string{"domain", "selector?"}},
	{"spf", "SPF Record Checker", "/v1/tools/email/spf", "email",
		"Expand the entire include tree and count the DNS lookups a receiver would spend, catching records that exceed the ten-lookup limit and fail authentication despite listing the right senders.",
		[]string{"domain"}},
	{"dkim", "DKIM Key Inspector", "/v1/tools/email/dkim", "email",
		"Discover DKIM selectors by probing those the major providers use, then validate each published key: type, length, revocation and testing mode.",
		[]string{"domain", "selector?"}},
	{"dmarc", "DMARC Policy Checker", "/v1/tools/email/dmarc", "email",
		"Parse the DMARC policy, follow inheritance from the organisational domain, and verify that external report destinations have published the authorisation record without which the reports are silently discarded.",
		[]string{"domain"}},
	{"domain_info", "Domain Registration Lookup", "/v1/tools/domain/info", "domain",
		"Fetch registrar, creation and expiry dates, nameservers and registry status codes over RDAP, with the status codes translated into plain English.",
		[]string{"domain"}},
	{"tls_certificate", "SSL/TLS Certificate Checker", "/v1/tools/tls/certificate", "tls",
		"Inspect the certificate chain, expiry, hostname coverage, key strength and supported TLS versions, including the missing-intermediate fault that browsers hide but other clients do not.",
		[]string{"domain", "port?"}},
	{"security_headers", "HTTP Security Headers", "/v1/tools/http/headers", "http",
		"Grade a page's security headers, with the Content-Security-Policy actually parsed for weaknesses rather than merely detected.",
		[]string{"url"}},
	{"redirect_chain", "Redirect Chain Tracer", "/v1/tools/http/redirects", "http",
		"Walk every redirect hop with its status and timing, detecting loops, and flagging chains that downgrade to plain HTTP part way through.",
		[]string{"url"}},
	{"ip_info", "IP & ASN Lookup", "/v1/tools/ip/info", "network",
		"Resolve a hostname and describe each address: the network operator, routing prefix, country, reverse DNS and IPv6 availability.",
		[]string{"target"}},
}

func (h *ToolsHandler) Catalogue(w http.ResponseWriter, r *http.Request) {
	utils.SendSuccess(w, map[string]any{
		"count": len(toolCatalogue),
		"tools": toolCatalogue,
	})
}
