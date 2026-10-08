package netcheck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

// SPF evaluation limits from RFC 7208. These are not advisory: a receiver that
// hits them stops evaluating and returns permerror, which most large providers
// treat as an authentication failure. Exceeding the lookup limit is the single
// most common reason a domain with an apparently correct SPF record still
// fails authentication, and it is invisible unless the include tree is walked.
const (
	spfMaxLookups = 10

	spfMaxVoid = 2

	spfMaxDepth = 10

	// spfWalkBudget caps how many lookups the walker performs before giving
	// up. It is deliberately above spfMaxLookups so the report can say how
	// far over the limit a record is, not merely that it is over.
	spfWalkBudget = 40
)

// spfSenders maps well-known include targets to the service that owns them, so
// the report can say "Google Workspace" rather than "_spf.google.com".
var spfSenders = map[string]string{
	"_spf.google.com":             "Google Workspace",
	"spf.protection.outlook.com":  "Microsoft 365",
	"spf.messaging.microsoft.com": "Microsoft (legacy)",
	"sendgrid.net":                "SendGrid",
	"spf.mandrillapp.com":         "Mandrill",
	"servers.mcsv.net":            "Mailchimp",
	"mailgun.org":                 "Mailgun",
	"spf.mailjet.com":             "Mailjet",
	"amazonses.com":               "Amazon SES",
	"_spf.salesforce.com":         "Salesforce",
	"spf.zoho.com":                "Zoho Mail",
	"zoho.com":                    "Zoho",
	"spf.sparkpostmail.com":       "SparkPost",
	"spf.mtasv.net":               "Postmark",
	"_spf.freshdesk.com":          "Freshdesk",
	"_spf.hubspot.net":            "HubSpot",
	"_spf.intercom.io":            "Intercom",
	"spf.klaviyodns.com":          "Klaviyo",
	"_spf.brevo.com":              "Brevo",
	"spf.sendinblue.com":          "Brevo (Sendinblue)",
	"_spf.mailerlite.com":         "MailerLite",
	"mail.zendesk.com":            "Zendesk",
	"_spf.atlassian.net":          "Atlassian",
	"spf.constantcontact.com":     "Constant Contact",
	"cmail1.com":                  "Campaign Monitor",
	"_spf.createsend.com":         "Campaign Monitor",
	"helpscoutemail.com":          "Help Scout",
	"_spf.qualtrics.com":          "Qualtrics",
	"spf.protonmail.ch":           "Proton Mail",
	"_spf.fastmail.com":           "Fastmail",
	"_spf.yandex.net":             "Yandex Mail",
	"_spf.pardot.com":             "Pardot",
	"et._spf.pardot.com":          "Salesforce Marketing Cloud",
	"_spf.smtp2go.com":            "SMTP2GO",
	"spf.beehiiv.com":             "beehiiv",
	"_spf.stripe.com":             "Stripe",
	"_spf.shopify.com":            "Shopify",
	"shops.shopify.com":           "Shopify",
	"_spf.squarespace.com":        "Squarespace",
	"_spf.wix.com":                "Wix",
	"_spf.webflow.com":            "Webflow",
	"mktomail.com":                "Marketo",
	"_spf.elasticemail.com":       "Elastic Email",
	"spf.mtasv.net.":              "Postmark",
}

// spfMechanisms is the closed set of mechanism names in RFC 7208. Anything
// else in mechanism position is a syntax error, not an extension point.
var spfMechanisms = map[string]bool{
	"all": true, "include": true, "a": true, "mx": true,
	"ptr": true, "ip4": true, "ip6": true, "exists": true,
}

// SPFTerm is one parsed term of an SPF record: either a mechanism carrying a
// qualifier, or a modifier of the form name=value.
type SPFTerm struct {
	Raw       string `json:"raw"`
	Qualifier string `json:"qualifier,omitempty"`
	Mechanism string `json:"mechanism,omitempty"`
	Modifier  string `json:"modifier,omitempty"`
	Value     string `json:"value,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SPFLookup is one DNS query the walker performed while expanding the record,
// recorded so the report can show exactly which mechanisms consume the budget.
type SPFLookup struct {
	Domain    string `json:"domain"`
	Via       string `json:"via"`
	Depth     int    `json:"depth"`
	Record    string `json:"record,omitempty"`
	Void      bool   `json:"void"`
	Error     string `json:"error,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Sequence  int    `json:"sequence"`
	OverLimit bool   `json:"over_limit"`
}

// SPFResult is the raw payload for the SPF report.
type SPFResult struct {
	Domain       string      `json:"domain"`
	Found        bool        `json:"found"`
	Record       string      `json:"record,omitempty"`
	AllRecords   []string    `json:"all_records,omitempty"`
	RecordCount  int         `json:"record_count"`
	Length       int         `json:"length"`
	Terms        []SPFTerm   `json:"terms,omitempty"`
	AllQualifier string      `json:"all_qualifier,omitempty"`
	RedirectTo   string      `json:"redirect_to,omitempty"`
	Lookups      []SPFLookup `json:"lookups,omitempty"`
	LookupCount  int         `json:"lookup_count"`
	VoidCount    int         `json:"void_count"`
	IPv4Count    int         `json:"ipv4_count"`
	IPv6Count    int         `json:"ipv6_count"`
	Includes     []string    `json:"includes,omitempty"`
	Senders      []string    `json:"senders,omitempty"`
	DenyAll      bool        `json:"deny_all"`
	SyntaxErrors []string    `json:"syntax_errors,omitempty"`
	Truncated    bool        `json:"truncated"`
}

// spfWalkState carries the mutable counters through the recursive expansion.
type spfWalkState struct {
	res     *SPFResult
	visited map[string]bool
	seq     int
}

// SPFCheck fetches a domain's SPF record and expands its entire include tree,
// counting DNS lookups the way a receiving mail server does. A record that
// looks correct in isolation is routinely broken by what its includes pull in,
// so the tree walk - not the syntax check - is the substance of this tool.
func SPFCheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("spf", target)
	res := &SPFResult{Domain: target}
	b.Raw(res)

	records, err := spfRecords(ctx, r, target)
	if err != nil && !errors.Is(err, ErrNoAnswer) && !errors.Is(err, ErrNXDomain) {
		return b.BuildError("lookup_failed", "SPF lookup failed", shortErr(err)), nil
	}

	res.AllRecords = records
	res.RecordCount = len(records)

	if len(records) == 0 {
		b.Fail("missing", "No SPF record", SeverityHigh,
			fmt.Sprintf("%s publishes no TXT record beginning with v=spf1, so no sending source is authorised for this domain.", target),
			"Publish an SPF record listing every service that sends mail as this domain, ending in -all once the list is complete. If the domain never sends mail, publish the deny-all record instead: "+target+` TXT "v=spf1 -all"`)
		b.Summary("No SPF record published")
		return b.Build(), nil
	}

	if len(records) > 1 {
		b.Fail("multiple_records", "More than one SPF record", SeverityCritical,
			fmt.Sprintf("%s publishes %d records beginning with v=spf1. RFC 7208 requires exactly one; a receiver seeing several must return permerror and treat SPF as unusable, which fails authentication for every message regardless of the contents of either record.", target, len(records)),
			"Merge the records into a single TXT record. Terms from the extra records - typically include: entries added by different teams - can be concatenated into one record, subject to the ten-lookup limit.")
	}

	res.Found = true
	res.Record = records[0]
	res.Length = len(res.Record)

	state := &spfWalkState{res: res, visited: map[string]bool{}}
	spfWalk(ctx, r, target, res.Record, 0, state)

	// A record of "v=spf1 -all" and nothing else is a deliberate statement
	// that the domain sends no mail at all, which is a correct configuration
	// rather than an empty one.
	res.DenyAll = res.AllQualifier == "-" && res.LookupCount == 0 &&
		res.IPv4Count == 0 && res.IPv6Count == 0

	res.Includes = dedupe(res.Includes)
	res.Senders = dedupe(res.Senders)

	spfFindings(b, res, target)

	summary := fmt.Sprintf("%s, %d of %d DNS lookups used", spfAllPhrase(res.AllQualifier), res.LookupCount, spfMaxLookups)
	if res.LookupCount > spfMaxLookups {
		summary = fmt.Sprintf("Over the DNS lookup limit: %d of a permitted %d", res.LookupCount, spfMaxLookups)
	}
	b.Summary(summary)
	return b.Build(), nil
}

// spfRecords returns every TXT record on a name that is an SPF record. The
// v=spf1 prefix is matched case-insensitively because receivers do the same.
func spfRecords(ctx context.Context, r *Resolver, name string) ([]string, error) {
	ans, err := r.Query(ctx, name, dns.TypeTXT)
	if ans == nil {
		return nil, err
	}
	var out []string
	for _, v := range ans.Values {
		trimmed := strings.TrimSpace(v)
		if len(trimmed) >= 6 && strings.EqualFold(trimmed[:6], "v=spf1") {
			out = append(out, trimmed)
		}
	}
	return out, err
}

// spfWalk expands one record, recording every DNS lookup its mechanisms cost.
// Sibling lookups are performed in term order so the sequence numbers in the
// report match the order a receiver would evaluate them.
func spfWalk(ctx context.Context, r *Resolver, domain, record string, depth int, st *spfWalkState) {
	if depth > spfMaxDepth || st.res.Truncated {
		return
	}
	if st.visited[domain] {
		st.res.SyntaxErrors = append(st.res.SyntaxErrors,
			fmt.Sprintf("include loop: %s is reached from itself", domain))
		return
	}
	st.visited[domain] = true
	defer delete(st.visited, domain)

	terms := parseSPFRecord(record)
	if depth == 0 {
		st.res.Terms = terms
	}

	hasAll := false
	for _, t := range terms {
		if t.Mechanism == "all" {
			hasAll = true
		}
	}

	for _, t := range terms {
		if ctx.Err() != nil {
			st.res.Truncated = true
			return
		}
		if t.Error != "" && depth == 0 {
			st.res.SyntaxErrors = append(st.res.SyntaxErrors, t.Error)
			continue
		}

		switch {
		case t.Modifier == "redirect":
			// redirect is ignored entirely when an all mechanism is present.
			if hasAll {
				continue
			}
			if depth == 0 {
				st.res.RedirectTo = t.Value
			}
			sub, ok := st.spend(ctx, r, t.Value, "redirect", depth)
			if ok {
				spfWalk(ctx, r, t.Value, sub, depth+1, st)
			}

		case t.Mechanism == "include":
			st.res.Includes = append(st.res.Includes, t.Value)
			sub, ok := st.spend(ctx, r, t.Value, "include", depth)
			if ok {
				spfWalk(ctx, r, t.Value, sub, depth+1, st)
			}

		case t.Mechanism == "a", t.Mechanism == "mx", t.Mechanism == "exists", t.Mechanism == "ptr":
			name := spfTermDomain(t, domain)
			st.spend(ctx, r, name, t.Mechanism, depth)

		case t.Mechanism == "ip4":
			st.res.IPv4Count++

		case t.Mechanism == "ip6":
			st.res.IPv6Count++

		case t.Mechanism == "all":
			if depth == 0 {
				st.res.AllQualifier = t.Qualifier
			} else if t.Qualifier == "+" {
				st.res.SyntaxErrors = append(st.res.SyntaxErrors,
					fmt.Sprintf("%s contains +all, which authorises every sender on the internet through this include", domain))
			}
		}
	}
}

// spend charges one DNS lookup against the budget and performs it, returning
// the target's SPF record when the mechanism was an include or redirect.
func (st *spfWalkState) spend(ctx context.Context, r *Resolver, name, via string, depth int) (string, bool) {
	st.res.LookupCount++
	st.seq++

	lk := SPFLookup{
		Domain:    name,
		Via:       via,
		Depth:     depth,
		Sequence:  st.seq,
		OverLimit: st.res.LookupCount > spfMaxLookups,
		Sender:    spfSenders[strings.ToLower(name)],
	}

	// A macro-expanded target depends on the connecting client, so the name
	// cannot be resolved here. The lookup is still charged, because a
	// receiver evaluating the record for a real message will spend it.
	if name == "" {
		lk.Domain = "(macro)"
		lk.Error = "target contains a macro; not resolvable without a connecting client"
		st.res.Lookups = append(st.res.Lookups, lk)
		return "", false
	}

	if st.res.LookupCount > spfWalkBudget {
		st.res.Truncated = true
		lk.Error = "not evaluated: expansion budget exhausted"
		st.res.Lookups = append(st.res.Lookups, lk)
		return "", false
	}
	if lk.Sender != "" {
		st.res.Senders = append(st.res.Senders, lk.Sender)
	}

	switch via {
	case "include", "redirect":
		records, err := spfRecords(ctx, r, name)
		switch {
		case len(records) == 0:
			// A dangling include is a permerror at the receiver, not a
			// silently skipped term.
			lk.Void = true
			st.res.VoidCount++
			lk.Error = "no SPF record published"
			if err != nil {
				lk.Error = shortErr(err)
			}
			st.res.Lookups = append(st.res.Lookups, lk)
			return "", false
		case len(records) > 1:
			lk.Error = "target publishes multiple SPF records"
		}
		lk.Record = records[0]
		st.res.Lookups = append(st.res.Lookups, lk)
		return records[0], true

	case "mx":
		ans, err := r.Query(ctx, name, dns.TypeMX)
		if err != nil || ans.Empty() {
			lk.Void = true
			st.res.VoidCount++
			lk.Error = "no MX records"
		} else if len(ans.Values) > spfMaxLookups {
			lk.Error = fmt.Sprintf("%d MX hosts: the mx mechanism may resolve at most %d", len(ans.Values), spfMaxLookups)
		}

	default: // a, exists, ptr
		ans, err := r.Query(ctx, name, dns.TypeA)
		if err != nil || ans.Empty() {
			lk.Void = true
			st.res.VoidCount++
			lk.Error = "does not resolve"
		}
	}

	st.res.Lookups = append(st.res.Lookups, lk)
	return "", false
}

// spfFindings turns the expanded result into report findings.
func spfFindings(b *Builder, res *SPFResult, target string) {
	// The lookup budget. This is the finding that matters most: it is the
	// usual cause of a "correct" record failing, and it is not visible
	// without expanding the tree.
	switch {
	case res.LookupCount > spfMaxLookups:
		over := res.LookupCount - spfMaxLookups
		b.Add(Finding{
			ID:       "lookup_limit_exceeded",
			Title:    "Over the ten DNS lookup limit",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Detail: fmt.Sprintf("Expanding the record requires %d DNS lookups, %s over the limit of %d that RFC 7208 permits. Receivers abandon evaluation at the tenth lookup and return permerror, so mail from this domain fails SPF regardless of whether the sending server is actually listed.",
				res.LookupCount, plural(over, "lookup", "lookups"), spfMaxLookups),
			Remediation: "Reduce the tree. Replace include: entries for services you no longer use, flatten a large include into the ip4:/ip6: ranges it resolves to (accepting that you must then track that provider's changes), or move a marketing platform onto a subdomain with its own SPF record so its lookups no longer count against this one.",
			Evidence:    spfOverLimitEvidence(res),
		})
	case res.LookupCount >= 8:
		b.Warn("lookup_limit_near", "Close to the DNS lookup limit", SeverityMedium,
			fmt.Sprintf("The record uses %d of the %d permitted DNS lookups. Adding one more sending service, or a provider expanding their own include, would push it over and break SPF for all mail.", res.LookupCount, spfMaxLookups),
			"Remove includes for services that no longer send as this domain before adding any new ones.")
	default:
		b.Pass("lookup_limit", "Within the DNS lookup limit",
			fmt.Sprintf("Expanding the record costs %d of the %d permitted DNS lookups.", res.LookupCount, spfMaxLookups))
	}

	if res.VoidCount > spfMaxVoid {
		b.Fail("void_lookup_limit", "Too many void lookups", SeverityHigh,
			fmt.Sprintf("%d mechanisms point at names that do not resolve. RFC 7208 permits at most %d such void lookups before a receiver returns permerror.", res.VoidCount, spfMaxVoid),
			"Remove the mechanisms whose targets no longer exist - they are listed in the lookups section of this report.")
	} else if res.VoidCount > 0 {
		b.Warn("void_lookups", "Mechanisms that do not resolve", SeverityMedium,
			fmt.Sprintf("%d mechanisms point at names with no records. They authorise nothing and consume part of the lookup budget; %s permitted before evaluation fails outright.", res.VoidCount, plural(spfMaxVoid, "is", "are")),
			"Delete the dead mechanisms listed in this report.")
	}

	for _, lk := range res.Lookups {
		if lk.Void && (lk.Via == "include" || lk.Via == "redirect") {
			b.Add(Finding{
				ID:          "dangling_" + lk.Via,
				Title:       "Include target has no SPF record",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Detail:      fmt.Sprintf("%s:%s resolves to no SPF record. A receiver treats an include of a domain without a record as a permanent error, not as an empty list, so this term alone can fail authentication for every message.", lk.Via, lk.Domain),
				Remediation: fmt.Sprintf("Remove %s:%s, or restore the SPF record on %s.", lk.Via, lk.Domain, lk.Domain),
				Evidence:    lk,
			})
		}
	}

	// The all mechanism decides what happens to mail from everything not
	// listed, which is the entire point of publishing SPF.
	switch res.AllQualifier {
	case "-":
		if res.DenyAll {
			b.Pass("deny_all", "The domain authorises no sender at all",
				"The record is v=spf1 -all with no mechanisms, which states that this domain sends no mail and that anything claiming to come from it is forged. This is the correct record for a domain that only hosts a website, or for one that is parked.")
		} else {
			b.Pass("all_fail", "Unlisted senders are rejected",
				"The record ends in -all, so a receiver treats mail from any source not listed here as unauthorised.")
		}
	case "~":
		b.Pass("all_softfail", "Unlisted senders soft-fail",
			"The record ends in ~all. Mail from unlisted sources is marked as failing but not rejected on SPF alone, which is the correct setting while DMARC enforcement does the rejecting.")
	case "?":
		b.Warn("all_neutral", "Unlisted senders are neutral", SeverityHigh,
			"The record ends in ?all, which explicitly declines to say anything about sources not listed. Receivers treat this the same as having no SPF policy at all, so the record provides no protection against forgery.",
			"Change ?all to ~all once you are confident the record lists every legitimate sender, then to -all.")
	case "+":
		b.Fail("all_pass", "Every sender is authorised", SeverityCritical,
			"The record ends in +all, which authorises every host on the internet to send mail as this domain. Anyone can pass SPF for this domain, and SPF-aligned DMARC will pass for them too.",
			"Replace +all with -all immediately, keeping the mechanisms that list your real senders.")
	default:
		if res.RedirectTo != "" {
			b.Info("all_via_redirect", "Policy comes from a redirect",
				fmt.Sprintf("The record has no all mechanism and redirects to %s, whose record supplies the default.", res.RedirectTo))
		} else {
			b.Fail("all_missing", "No all mechanism", SeverityHigh,
				"The record does not end in an all mechanism, so it never states what should happen to mail from sources it does not list. Receivers default to neutral, which gives the record no enforcement value.",
				"Append ~all (or -all) as the final term of the record.")
		}
	}

	if len(res.SyntaxErrors) > 0 {
		b.Add(Finding{
			ID:          "syntax_errors",
			Title:       "Syntax errors in the record",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Detail:      fmt.Sprintf("The record contains %s a receiver cannot parse. Parsing stops at the first error and returns permerror.", plural(len(res.SyntaxErrors), "a term", "terms")),
			Remediation: "Correct the terms listed in the evidence. Every mechanism must be one of all, include, a, mx, ptr, ip4, ip6 or exists.",
			Evidence:    res.SyntaxErrors,
		})
	}

	for _, t := range res.Terms {
		if t.Mechanism == "ptr" {
			b.Warn("ptr_mechanism", "Deprecated ptr mechanism", SeverityMedium,
				"The record uses the ptr mechanism, which RFC 7208 deprecates. It relies on reverse DNS the sender does not control, is slow enough that some receivers skip it, and several large providers ignore it entirely.",
				"Replace ptr with explicit ip4:/ip6: ranges or an a: mechanism naming the sending hosts.")
			break
		}
	}

	if res.Length > 255 {
		b.Info("multi_string", "Record spans multiple strings",
			fmt.Sprintf("The record is %d characters, so it must be published as several quoted strings inside one TXT record. Splitting it across two separate TXT records instead would make the domain publish two SPF records and fail authentication.", res.Length))
	}
	if res.Length > 450 {
		b.Warn("record_length", "Unusually long record", SeverityLow,
			fmt.Sprintf("At %d characters the record risks pushing the TXT response past 512 bytes, forcing DNS retries over TCP. Some poorly configured resolvers and middleboxes drop those.", res.Length),
			"Shorten the record by removing unused mechanisms, or move a sending service to its own subdomain.")
	}

	if len(res.Senders) > 0 {
		b.Info("senders", "Authorised sending services",
			fmt.Sprintf("The record authorises %s.", strings.Join(res.Senders, ", ")))
	}
	if res.Truncated {
		b.Warn("truncated", "Expansion incomplete", SeverityLow,
			"The include tree was too large or too slow to expand fully, so the lookup count is a lower bound.",
			"A tree this size is itself the problem: reduce the number of includes.")
	}
}

// spfOverLimitEvidence lists the lookups that fall beyond the tenth, which are
// precisely the mechanisms a receiver never reaches.
func spfOverLimitEvidence(res *SPFResult) []SPFLookup {
	var out []SPFLookup
	for _, lk := range res.Lookups {
		if lk.OverLimit {
			out = append(out, lk)
		}
	}
	return out
}

func spfAllPhrase(q string) string {
	switch q {
	case "-":
		return "Unlisted senders rejected"
	case "~":
		return "Unlisted senders soft-failed"
	case "?":
		return "Unlisted senders neutral"
	case "+":
		return "All senders authorised"
	default:
		return "No all mechanism"
	}
}

// parseSPFRecord splits a record into terms. The v=spf1 version token is
// consumed here; anything before it is a malformed record.
func parseSPFRecord(record string) []SPFTerm {
	fields := strings.Fields(record)
	var terms []SPFTerm
	for i, f := range fields {
		if i == 0 {
			continue // the v=spf1 version token
		}
		terms = append(terms, parseSPFTerm(f))
	}
	return terms
}

// parseSPFTerm parses one whitespace-delimited term. Modifiers are name=value
// pairs with no qualifier; everything else is a mechanism.
func parseSPFTerm(raw string) SPFTerm {
	t := SPFTerm{Raw: raw}
	s := raw

	// A modifier is distinguished from a mechanism by an "=" appearing before
	// any ":" or "/", which is what separates redirect=x from ip4:1.2.3.4.
	if i := strings.Index(s, "="); i > 0 {
		if j := strings.IndexAny(s, ":/"); j < 0 || i < j {
			t.Modifier = strings.ToLower(s[:i])
			// Unknown modifiers are ignored by receivers rather than treated
			// as errors, so no validation of the name happens here.
			t.Value = strings.TrimSuffix(s[i+1:], ".")
			return t
		}
	}

	t.Qualifier = "+"
	if len(s) > 0 {
		switch s[0] {
		case '+', '-', '~', '?':
			t.Qualifier = string(s[0])
			s = s[1:]
		}
	}

	name := s
	value := ""
	if i := strings.IndexAny(s, ":/"); i >= 0 {
		name = s[:i]
		if s[i] == ':' {
			value = s[i+1:]
		} else {
			value = s[i:] // a CIDR suffix, e.g. a/24
		}
	}

	t.Mechanism = strings.ToLower(name)
	t.Value = value
	if !spfMechanisms[t.Mechanism] {
		t.Error = fmt.Sprintf("%q is not a valid SPF mechanism", raw)
		t.Mechanism = ""
	}
	return t
}

// spfTermDomain resolves what name an a/mx/exists/ptr mechanism queries,
// which is the current domain when the term carries no explicit target.
func spfTermDomain(t SPFTerm, current string) string {
	v := t.Value
	if i := strings.Index(v, "/"); i >= 0 {
		v = v[:i]
	}
	v = strings.TrimSpace(strings.TrimSuffix(v, "."))
	if v == "" {
		return current
	}
	if strings.Contains(v, "%") {
		// Macro expansion depends on the connecting client, so the name
		// cannot be resolved statically. Charge the lookup, skip the query.
		return ""
	}
	return v
}
