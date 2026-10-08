package netcheck

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

// DMARCURI is one destination from a rua or ruf tag, together with the result
// of the external authorisation check RFC 7489 section 7.1 requires.
type DMARCURI struct {
	URI        string `json:"uri"`
	Scheme     string `json:"scheme"`
	Address    string `json:"address"`
	Domain     string `json:"domain,omitempty"`
	MaxSize    string `json:"max_size,omitempty"`
	External   bool   `json:"external"`
	Authorised bool   `json:"authorised"`
	AuthRecord string `json:"auth_record,omitempty"`
	AuthError  string `json:"auth_error,omitempty"`
}

// DMARCResult is the raw payload for the DMARC report.
type DMARCResult struct {
	Domain       string            `json:"domain"`
	Found        bool              `json:"found"`
	FoundAt      string            `json:"found_at,omitempty"`
	Inherited    bool              `json:"inherited"`
	OrgDomain    string            `json:"org_domain"`
	Record       string            `json:"record,omitempty"`
	RecordCount  int               `json:"record_count"`
	Tags         map[string]string `json:"tags,omitempty"`
	Policy       string            `json:"policy,omitempty"`
	SubPolicy    string            `json:"subdomain_policy,omitempty"`
	Effective    string            `json:"effective_policy,omitempty"`
	Percent      int               `json:"percent"`
	ADKIM        string            `json:"adkim"`
	ASPF         string            `json:"aspf"`
	Interval     int               `json:"report_interval"`
	Aggregate    []DMARCURI        `json:"aggregate_reports,omitempty"`
	Forensic     []DMARCURI        `json:"forensic_reports,omitempty"`
	SyntaxErrors []string          `json:"syntax_errors,omitempty"`
}

// dmarcPolicies is the closed set of values the p and sp tags accept.
var dmarcPolicies = map[string]bool{"none": true, "quarantine": true, "reject": true}

// DMARCCheck inspects a domain's DMARC policy. Where no record exists on the
// domain itself, the organisational domain is consulted, because that is what
// a receiver does before concluding a domain has no policy.
func DMARCCheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("dmarc", target)
	res := &DMARCResult{Domain: target, Percent: 100, ADKIM: "r", ASPF: "r", Interval: 86400}
	res.OrgDomain = orgDomain(target)
	b.Raw(res)

	records, lookupErr := dmarcRecords(ctx, r, target)
	res.FoundAt = "_dmarc." + target

	// A subdomain with no policy of its own inherits from the organisational
	// domain, where the sp tag - not p - is what applies to it.
	if len(records) == 0 && res.OrgDomain != "" && res.OrgDomain != target {
		if orgRecords, _ := dmarcRecords(ctx, r, res.OrgDomain); len(orgRecords) > 0 {
			records = orgRecords
			res.Inherited = true
			res.FoundAt = "_dmarc." + res.OrgDomain
		}
	}

	res.RecordCount = len(records)
	if len(records) == 0 {
		detail := fmt.Sprintf("Neither %s nor its organisational domain %s publishes a DMARC record.", "_dmarc."+target, res.OrgDomain)
		if res.OrgDomain == target {
			detail = fmt.Sprintf("%s publishes no DMARC record.", "_dmarc."+target)
		}
		if lookupErr != nil {
			detail += " (" + shortErr(lookupErr) + ")"
		}
		b.Fail("missing", "No DMARC record", SeverityCritical,
			detail+" Without one, receivers have no instruction on what to do with mail that fails authentication while claiming to be from this domain, and the domain's owner receives no reports of who is sending as it.",
			`Start in monitoring mode so nothing breaks while you find your legitimate senders: `+"_dmarc."+target+` TXT "v=DMARC1; p=none; rua=mailto:dmarc@`+target+`". Read the aggregate reports for a few weeks, fix the sources that fail, then move to p=quarantine and finally p=reject.`)
		b.Summary("No DMARC record published")
		return b.Build(), nil
	}

	if len(records) > 1 {
		b.Fail("multiple_records", "More than one DMARC record", SeverityCritical,
			fmt.Sprintf("%s publishes %d DMARC records. RFC 7489 requires receivers to ignore the policy entirely when more than one is found, so the domain is treated as having no DMARC at all.", res.FoundAt, len(records)),
			"Delete the extra records, keeping one.")
	}

	res.Found = true
	res.Record = records[0]
	res.Tags = parseDMARCTags(res.Record)

	dmarcApplyTags(res)
	dmarcResolveURIs(ctx, r, res, target)
	dmarcFindings(b, res, target)

	b.Summary(fmt.Sprintf("Policy %s at %s", strings.ToUpper(res.Effective), res.FoundAt))
	return b.Build(), nil
}

// dmarcRecords returns the DMARC records published under a domain's _dmarc name.
func dmarcRecords(ctx context.Context, r *Resolver, name string) ([]string, error) {
	ans, err := r.Query(ctx, "_dmarc."+name, dns.TypeTXT)
	if ans == nil {
		return nil, err
	}
	var out []string
	for _, v := range ans.Values {
		t := strings.TrimSpace(v)
		if len(t) >= 8 && strings.EqualFold(t[:8], "v=DMARC1") {
			out = append(out, t)
		}
	}
	return out, err
}

// dmarcApplyTags validates the parsed tags and derives the effective policy.
func dmarcApplyTags(res *DMARCResult) {
	if v, ok := res.Tags["v"]; !ok || !strings.EqualFold(v, "DMARC1") {
		res.SyntaxErrors = append(res.SyntaxErrors, "the record does not begin with v=DMARC1")
	}

	res.Policy = strings.ToLower(res.Tags["p"])
	if res.Policy == "" {
		res.SyntaxErrors = append(res.SyntaxErrors, "the required p tag is missing")
	} else if !dmarcPolicies[res.Policy] {
		res.SyntaxErrors = append(res.SyntaxErrors, fmt.Sprintf("p=%s is not one of none, quarantine or reject", res.Policy))
		res.Policy = ""
	}

	if sp, ok := res.Tags["sp"]; ok {
		res.SubPolicy = strings.ToLower(sp)
		if !dmarcPolicies[res.SubPolicy] {
			res.SyntaxErrors = append(res.SyntaxErrors, fmt.Sprintf("sp=%s is not one of none, quarantine or reject", sp))
			res.SubPolicy = ""
		}
	}

	// When the policy was inherited from the organisational domain, it is the
	// sp tag that governs this subdomain, falling back to p when sp is absent.
	res.Effective = res.Policy
	if res.Inherited && res.SubPolicy != "" {
		res.Effective = res.SubPolicy
	}
	if res.Effective == "" {
		res.Effective = "none"
	}

	if pct, ok := res.Tags["pct"]; ok {
		n, err := strconv.Atoi(strings.TrimSpace(pct))
		if err != nil || n < 0 || n > 100 {
			res.SyntaxErrors = append(res.SyntaxErrors, fmt.Sprintf("pct=%s is not a number between 0 and 100", pct))
		} else {
			res.Percent = n
		}
	}
	if v, ok := res.Tags["adkim"]; ok {
		res.ADKIM = strings.ToLower(v)
	}
	if v, ok := res.Tags["aspf"]; ok {
		res.ASPF = strings.ToLower(v)
	}
	if v, ok := res.Tags["ri"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			res.Interval = n
		}
	}

	res.Aggregate = parseDMARCURIs(res.Tags["rua"])
	res.Forensic = parseDMARCURIs(res.Tags["ruf"])
}

// dmarcResolveURIs performs the external destination authorisation check. A
// domain may not simply name someone else's mailbox as its report destination:
// the receiving domain must opt in by publishing a record at
// <sender>._report._dmarc.<receiver>. Reports to an unauthorised destination
// are silently discarded by every conforming receiver, which is why a domain
// can appear correctly configured and still receive nothing.
func dmarcResolveURIs(ctx context.Context, r *Resolver, res *DMARCResult, target string) {
	check := func(list []DMARCURI) []DMARCURI {
		for i := range list {
			u := &list[i]
			if u.Domain == "" || u.Scheme != "mailto" {
				continue
			}
			if orgDomain(u.Domain) == orgDomain(target) {
				u.Authorised = true
				continue
			}
			u.External = true

			name := target + "._report._dmarc." + u.Domain
			ans, err := r.Query(ctx, name, dns.TypeTXT)
			if err != nil || ans == nil || len(ans.Values) == 0 {
				u.AuthError = "no authorisation record at " + name
				continue
			}
			for _, v := range ans.Values {
				if t := strings.TrimSpace(v); len(t) >= 8 && strings.EqualFold(t[:8], "v=DMARC1") {
					u.Authorised = true
					u.AuthRecord = t
					break
				}
			}
			if !u.Authorised {
				u.AuthError = "the record at " + name + " is not a valid DMARC authorisation record"
			}
		}
		return list
	}
	res.Aggregate = check(res.Aggregate)
	res.Forensic = check(res.Forensic)
}

func dmarcFindings(b *Builder, res *DMARCResult, target string) {
	if len(res.SyntaxErrors) > 0 {
		b.Add(Finding{
			ID:          "syntax_errors",
			Title:       "Invalid DMARC record",
			Status:      StatusFail,
			Severity:    SeverityCritical,
			Detail:      fmt.Sprintf("The record at %s has %s. A record a receiver cannot parse is discarded, leaving the domain with no policy in force.", res.FoundAt, plural(len(res.SyntaxErrors), "an error", "errors")),
			Remediation: "Correct the record. The minimum valid form is v=DMARC1; p=none; with v first and p second.",
			Evidence:    res.SyntaxErrors,
		})
	}

	if res.Inherited {
		which := "p"
		if res.SubPolicy != "" {
			which = "sp"
		}
		b.Info("inherited", "Policy inherited from the organisational domain",
			fmt.Sprintf("%s has no DMARC record of its own, so receivers apply the record at %s, where the %s tag governs this subdomain.", target, res.FoundAt, which))
	}

	switch res.Effective {
	case "reject":
		b.Pass("policy_reject", "Policy is reject",
			"Mail that fails authentication while claiming to be from this domain is rejected outright, which is the strongest DMARC setting and the one that actually stops direct-domain spoofing.")
	case "quarantine":
		b.Warn("policy_quarantine", "Policy is quarantine", SeverityLow,
			"Failing mail is delivered to the spam folder rather than rejected. This is the right intermediate step, but forged mail still reaches the recipient's mailbox where it can be found.",
			"Move to p=reject once the aggregate reports show no legitimate source failing.")
	default:
		b.Fail("policy_none", "Policy is monitoring only", SeverityHigh,
			fmt.Sprintf("The effective policy at %s is p=none, which asks receivers to report on failures but to deliver the mail anyway. The domain can still be spoofed exactly as if it published no policy; the only thing DMARC is doing here is generating reports.", res.FoundAt),
			"p=none is the correct starting point. Use the aggregate reports to identify every legitimate sender, fix their SPF and DKIM alignment, then move to p=quarantine and on to p=reject. A domain left at p=none indefinitely gets no protection from DMARC.")
	}

	if res.Percent < 100 {
		sev := SeverityMedium
		if res.Effective == "none" {
			sev = SeverityLow
		}
		b.Warn("partial_enforcement", "Policy applied to only part of the mail", sev,
			fmt.Sprintf("pct=%d tells receivers to apply the %s policy to %d%% of failing messages and to fall back to the next weaker policy for the rest. %d%% of forged mail is handled as though the policy were weaker than it reads.", res.Percent, res.Effective, res.Percent, 100-res.Percent),
			"pct is a ramp for rolling out enforcement gradually. Raise it to 100 once reports show no legitimate source failing, and remove the tag.")
	}

	// Subdomain policy. A weaker sp than p is a real gap: attackers forge
	// subdomains precisely because they are so often left unprotected.
	if res.SubPolicy != "" && !res.Inherited {
		if dmarcStrength(res.SubPolicy) < dmarcStrength(res.Policy) {
			b.Warn("subdomain_weaker", "Subdomains have a weaker policy", SeverityHigh,
				fmt.Sprintf("p=%s protects %s itself, but sp=%s applies to every subdomain. Forged mail from names like billing.%s is handled under the weaker policy, and a receiver cannot tell that such a subdomain was never meant to exist.", res.Policy, target, res.SubPolicy, target),
				fmt.Sprintf("Unless a subdomain genuinely needs the looser setting, set sp=%s to match, or remove the sp tag so subdomains inherit p.", res.Policy))
		} else {
			b.Pass("subdomain_policy", "Subdomain policy set",
				fmt.Sprintf("sp=%s governs mail from subdomains of %s.", res.SubPolicy, target))
		}
	} else if !res.Inherited && res.Effective != "none" {
		b.Info("subdomain_inherit", "Subdomains inherit the policy",
			fmt.Sprintf("No sp tag is present, so subdomains of %s are governed by p=%s.", target, res.Effective))
	}

	// Reporting. Without rua there is no way to know who is sending as the
	// domain, which makes moving off p=none guesswork.
	if len(res.Aggregate) == 0 {
		b.Fail("no_rua", "No aggregate report destination", SeverityMedium,
			"The record has no rua tag, so no receiver sends aggregate reports about mail claiming to be from this domain. There is no way to discover which legitimate services are failing authentication, and no warning when the domain is being spoofed.",
			fmt.Sprintf(`Add a destination: rua=mailto:dmarc@%s. Reports arrive as gzipped XML once a day from each receiver.`, target))
	} else {
		authorised := 0
		for _, u := range res.Aggregate {
			if u.Authorised {
				authorised++
			}
		}
		if authorised > 0 {
			b.Pass("rua", "Aggregate reporting configured",
				fmt.Sprintf("Aggregate reports are sent to %s.", dmarcURIList(res.Aggregate)))
		}
		for _, u := range res.Aggregate {
			if u.External && !u.Authorised {
				b.Add(Finding{
					ID:       "rua_unauthorised",
					Title:    "Aggregate reports go to an unauthorised destination",
					Status:   StatusFail,
					Severity: SeverityHigh,
					Detail: fmt.Sprintf("Reports are addressed to %s, which is outside %s. RFC 7489 requires the receiving domain to consent by publishing an authorisation record at %s._report._dmarc.%s, and %s. Conforming receivers discard the reports silently, so the domain appears to be configured for reporting while receiving nothing.",
						u.Address, res.OrgDomain, target, u.Domain, u.AuthError),
					Remediation: fmt.Sprintf(`Publish the consent record on %s: %s._report._dmarc.%s TXT "v=DMARC1". If %s is a hosted DMARC service, they normally publish it for you once the domain is added to your account - check that the domain was actually added.`, u.Domain, target, u.Domain, u.Domain),
					Evidence:    u,
				})
			}
		}
	}

	for _, u := range res.Forensic {
		if u.External && !u.Authorised {
			b.Warn("ruf_unauthorised", "Forensic reports go to an unauthorised destination", SeverityLow,
				fmt.Sprintf("Failure reports are addressed to %s without the authorisation record at %s._report._dmarc.%s. Few receivers send failure reports at all, so this is a smaller loss than the equivalent rua problem.", u.Address, target, u.Domain),
				fmt.Sprintf(`Publish %s._report._dmarc.%s TXT "v=DMARC1" or drop the ruf tag.`, target, u.Domain))
		}
	}
	if len(res.Forensic) > 0 {
		b.Info("ruf", "Forensic reporting requested",
			fmt.Sprintf("Failure reports are requested to %s. These contain message headers and sometimes body content, so the destination mailbox receives personal data from third parties; most large receivers decline to send them for that reason.", dmarcURIList(res.Forensic)))
	}

	if res.ADKIM == "s" || res.ASPF == "s" {
		b.Info("strict_alignment", "Strict alignment required",
			fmt.Sprintf("adkim=%s and aspf=%s. Under strict alignment the authenticated domain must match %s exactly; a subdomain no longer counts, which breaks any service that signs or sends as a subdomain.", res.ADKIM, res.ASPF, target))
	}

	if res.Interval != 86400 && res.Interval > 0 {
		b.Info("report_interval", "Non-default report interval",
			fmt.Sprintf("ri=%d requests aggregate reports every %s. Most receivers honour only the daily default and will ignore this.", res.Interval, humanDuration(uint32(res.Interval))))
	}
}

// dmarcStrength orders the policies so they can be compared.
func dmarcStrength(p string) int {
	switch p {
	case "reject":
		return 3
	case "quarantine":
		return 2
	case "none":
		return 1
	}
	return 0
}

// parseDMARCTags splits a record into its semicolon-separated tag=value pairs.
func parseDMARCTags(record string) map[string]string {
	tags := map[string]string{}
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(part[:eq]))
		tags[key] = strings.TrimSpace(part[eq+1:])
	}
	return tags
}

// parseDMARCURIs splits a rua or ruf value into destinations, keeping the
// optional size limit suffix (mailto:x@y!10m).
func parseDMARCURIs(value string) []DMARCURI {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var out []DMARCURI
	for _, raw := range strings.Split(value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u := DMARCURI{URI: raw}

		body := raw
		if i := strings.Index(body, "!"); i >= 0 {
			u.MaxSize = body[i+1:]
			body = body[:i]
		}
		if i := strings.Index(body, ":"); i >= 0 {
			u.Scheme = strings.ToLower(body[:i])
			u.Address = body[i+1:]
		} else {
			u.Scheme = "mailto"
			u.Address = body
		}
		if at := strings.LastIndex(u.Address, "@"); at >= 0 {
			u.Domain = strings.ToLower(strings.TrimSuffix(u.Address[at+1:], "."))
		}
		out = append(out, u)
	}
	return out
}

func dmarcURIList(list []DMARCURI) string {
	out := make([]string, 0, len(list))
	for _, u := range list {
		out = append(out, u.Address)
	}
	return strings.Join(out, ", ")
}

// orgDomain returns the registrable domain - the public suffix plus one label -
// which is the unit DMARC policy inheritance and report authorisation work on.
func orgDomain(domain string) string {
	if org, err := publicsuffix.EffectiveTLDPlusOne(domain); err == nil {
		return org
	}
	return domain
}
