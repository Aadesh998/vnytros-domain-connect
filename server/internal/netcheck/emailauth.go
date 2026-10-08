package netcheck

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// MXHost is one mail exchanger, with the addresses it resolves to.
type MXHost struct {
	Preference uint16   `json:"preference"`
	Host       string   `json:"host"`
	IPs        []string `json:"ips,omitempty"`
	Provider   string   `json:"provider,omitempty"`
	Resolves   bool     `json:"resolves"`
}

// MXResult describes where a domain receives mail, and is what tells the
// combined report whether the domain is a real mailbox domain or a parked name.
type MXResult struct {
	Domain    string   `json:"domain"`
	Hosts     []MXHost `json:"hosts,omitempty"`
	NullMX    bool     `json:"null_mx"`
	Providers []string `json:"providers,omitempty"`
	Receives  bool     `json:"receives_mail"`
}

// EmailAuthResult is the raw payload for the combined report: the three
// authentication mechanisms plus where the domain receives mail.
type EmailAuthResult struct {
	Domain string       `json:"domain"`
	MX     *MXResult    `json:"mx,omitempty"`
	SPF    *SPFResult   `json:"spf,omitempty"`
	DKIM   *DKIMResult  `json:"dkim,omitempty"`
	DMARC  *DMARCResult `json:"dmarc,omitempty"`
	Scores struct {
		SPF   int `json:"spf"`
		DKIM  int `json:"dkim"`
		DMARC int `json:"dmarc"`
	} `json:"component_scores"`
}

// mxProviders maps mail exchanger hostname suffixes to the service behind them.
var mxProviders = map[string]string{
	"google.com":             "Google Workspace",
	"googlemail.com":         "Google Workspace",
	"outlook.com":            "Microsoft 365",
	"protection.outlook.com": "Microsoft 365",
	"zoho.com":               "Zoho Mail",
	"zoho.eu":                "Zoho Mail",
	"messagingengine.com":    "Fastmail",
	"protonmail.ch":          "Proton Mail",
	"pphosted.com":           "Proofpoint",
	"mimecast.com":           "Mimecast",
	"barracudanetworks.com":  "Barracuda",
	"mailgun.org":            "Mailgun",
	"improvmx.com":           "ImprovMX",
	"forwardemail.net":       "Forward Email",
	"yandex.net":             "Yandex Mail",
	"qq.com":                 "Tencent Exmail",
	"secureserver.net":       "GoDaddy",
	"hostinger.com":          "Hostinger",
	"titan.email":            "Titan",
	"registrar-servers.com":  "Namecheap Private Email",
	"emailsrvr.com":          "Rackspace",
	"amazonaws.com":          "Amazon SES",
	"cloudflare.net":         "Cloudflare Email Routing",
}

// EmailAuthCheck runs SPF, DKIM and DMARC together and adds the cross-checks
// that only make sense across all three: an enforcing DMARC policy over a
// domain that cannot authenticate anything rejects its own legitimate mail,
// and a domain that receives mail but publishes no policy can be spoofed. Each
// mechanism read alone looks fine in both cases.
func EmailAuthCheck(ctx context.Context, r *Resolver, domain string, selectors []string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("email_auth", target)
	res := &EmailAuthResult{Domain: target}
	b.Raw(res)

	var (
		wg                     sync.WaitGroup
		spfRep, dkimRep, dmRep *Report
		spfErr, dkimErr, dmErr error
	)

	wg.Add(4)
	go func() { defer wg.Done(); res.MX = mxLookup(ctx, r, target) }()
	go func() { defer wg.Done(); spfRep, spfErr = SPFCheck(ctx, r, target) }()
	go func() { defer wg.Done(); dkimRep, dkimErr = DKIMCheck(ctx, r, target, selectors) }()
	go func() { defer wg.Done(); dmRep, dmErr = DMARCCheck(ctx, r, target) }()
	wg.Wait()

	if spfErr != nil || dkimErr != nil || dmErr != nil {
		return b.BuildError("check_failed", "Email authentication check failed",
			shortErr(firstErr(spfErr, dkimErr, dmErr))), nil
	}

	res.SPF, _ = spfRep.Raw.(*SPFResult)
	res.DKIM, _ = dkimRep.Raw.(*DKIMResult)
	res.DMARC, _ = dmRep.Raw.(*DMARCResult)
	res.Scores.SPF = spfRep.Score
	res.Scores.DKIM = dkimRep.Score
	res.Scores.DMARC = dmRep.Score

	// A domain that states it sends no mail needs no DKIM key and no sending
	// sources, so the checks that would demand them are dropped rather than
	// reported against a configuration that is already correct.
	mxFindings(b, res.MX, target)
	adopt(b, "spf", spfRep)
	if sendsNoMail(res) {
		adopt(b, "dkim", dkimRep, "not_discovered", "single_key")
	} else {
		adopt(b, "dkim", dkimRep)
	}
	adopt(b, "dmarc", dmRep)
	emailAuthCrossChecks(b, res, target)

	b.Summary(emailAuthSummary(res))
	return b.Build(), nil
}

// adopt copies a sub-report's findings into the combined report, namespacing
// the IDs so they stay stable and diffable per mechanism. Findings named in
// skip are dropped: a mechanism can look deficient in isolation and be
// perfectly correct once the other two are known, and reporting it anyway
// would tell the user to fix something that is not broken.
func adopt(b *Builder, prefix string, rep *Report, skip ...string) {
	for _, f := range rep.Findings {
		if contains(skip, f.ID) {
			continue
		}
		f.ID = prefix + "." + f.ID
		f.Title = strings.ToUpper(prefix) + ": " + f.Title
		b.Add(f)
	}
}

// mxLookup finds where the domain receives mail.
func mxLookup(ctx context.Context, r *Resolver, target string) *MXResult {
	out := &MXResult{Domain: target}

	ans, err := r.Query(ctx, target, dns.TypeMX)
	if err != nil || ans == nil {
		return out
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, rr := range ans.RRs {
		mx, ok := rr.(*dns.MX)
		if !ok {
			continue
		}
		host := strings.TrimSuffix(mx.Mx, ".")

		// RFC 7505: a single MX of "." declares that the domain accepts no
		// mail at all, which is a deliberate configuration rather than a fault.
		if host == "" || host == "." {
			out.NullMX = true
			continue
		}

		h := MXHost{Preference: mx.Preference, Host: host, Provider: mxProviderFor(host)}
		wg.Add(1)
		go func(h MXHost) {
			defer wg.Done()
			ips := r.HostIPs(ctx, h.Host)
			h.IPs = ips
			h.Resolves = len(ips) > 0
			mu.Lock()
			out.Hosts = append(out.Hosts, h)
			mu.Unlock()
		}(h)
	}
	wg.Wait()

	for _, h := range out.Hosts {
		if h.Provider != "" {
			out.Providers = append(out.Providers, h.Provider)
		}
	}
	out.Providers = dedupe(out.Providers)
	out.Receives = len(out.Hosts) > 0
	return out
}

func mxFindings(b *Builder, mx *MXResult, target string) {
	switch {
	case mx.NullMX && len(mx.Hosts) == 0:
		b.Pass("mx.null", "MX: domain accepts no mail",
			"A null MX record declares that this domain receives no mail, so receivers reject undeliverable mail immediately instead of retrying for days.")
	case len(mx.Hosts) == 0:
		b.Info("mx.none", "MX: no mail exchangers",
			fmt.Sprintf("%s publishes no MX records, so it receives no mail. If the domain is not meant to, publishing a null MX record makes that explicit: %s MX 0 .", target, target))
	default:
		dead := 0
		for _, h := range mx.Hosts {
			if !h.Resolves {
				dead++
			}
		}
		if dead > 0 {
			b.Fail("mx.unresolvable", "MX: mail exchangers that do not resolve", SeverityHigh,
				fmt.Sprintf("%d of %d mail exchangers have no A or AAAA record. Mail to this domain that hits one of them is deferred until it bounces.", dead, len(mx.Hosts)),
				"Correct or remove the MX records whose hostnames no longer resolve.")
		}
		detail := fmt.Sprintf("%s receives mail through %s.", target, plural(len(mx.Hosts), "one mail exchanger", "mail exchangers"))
		if len(mx.Providers) > 0 {
			detail = fmt.Sprintf("%s receives mail through %s.", target, strings.Join(mx.Providers, ", "))
		}
		if dead == 0 {
			b.Pass("mx.ok", "MX: mail exchangers resolve", detail)
		}
		if mx.NullMX {
			b.Fail("mx.null_conflict", "MX: null MX alongside real ones", SeverityMedium,
				"The domain publishes both a null MX and real mail exchangers. RFC 7505 requires the null MX to be the only record, so receivers may take the domain as accepting no mail and reject everything.",
				"Delete the MX 0 . record, keeping the real exchangers.")
		}
	}
}

// emailAuthCrossChecks holds the findings that no single-mechanism check can
// produce, because each depends on the state of two or three of them at once.
func emailAuthCrossChecks(b *Builder, res *EmailAuthResult, target string) {
	spfUsable := res.SPF != nil && res.SPF.Found &&
		res.SPF.LookupCount <= spfMaxLookups && len(res.SPF.SyntaxErrors) == 0

	dkimUsable := false
	if res.DKIM != nil {
		for _, k := range res.DKIM.Keys {
			if k.Valid && !k.Revoked {
				dkimUsable = true
				break
			}
		}
	}

	enforcing := res.DMARC != nil && (res.DMARC.Effective == "reject" || res.DMARC.Effective == "quarantine")
	receives := res.MX != nil && res.MX.Receives

	// The dangerous combination: enforcement over nothing to authenticate
	// with. Each mechanism looks individually explicable; together they mean
	// the domain instructs receivers to discard its own mail.
	if enforcing && !spfUsable && !dkimUsable {
		verb := "rejected"
		if res.DMARC.Effective == "quarantine" {
			verb = "sent to spam"
		}
		b.Fail("cross.enforcing_without_auth", "Enforcing DMARC with no working authentication", SeverityCritical,
			fmt.Sprintf("The DMARC policy is p=%s, but SPF is missing or unusable and no valid DKIM key was found. Nothing this domain sends can pass either mechanism, so every message it sends is %s at any receiver honouring DMARC - including its own legitimate mail.",
				res.DMARC.Effective, verb),
			"Drop the policy to p=none until authentication works, then fix SPF and DKIM, confirm from the aggregate reports that legitimate mail is passing, and only then restore enforcement.")
	}

	if enforcing && res.SPF != nil && res.SPF.LookupCount > spfMaxLookups && !dkimUsable {
		b.Fail("cross.enforcement_over_broken_spf", "Enforcement relies on an SPF record that fails", SeverityCritical,
			fmt.Sprintf("SPF needs %d DNS lookups against a limit of %d, so it returns permerror rather than pass, and no valid DKIM key was found to authenticate mail instead. With p=%s in force, legitimate mail has nothing left to pass DMARC on.",
				res.SPF.LookupCount, spfMaxLookups, res.DMARC.Effective),
			"Bring the SPF record back under ten lookups, or enable DKIM signing so mail can align through DKIM while SPF is being repaired.")
	}

	// A domain that carries mail but publishes no enforcing policy is the
	// case attackers actually exploit.
	if receives && res.DMARC != nil && res.DMARC.Effective == "none" {
		b.Warn("cross.spoofable", "The domain receives mail and can be spoofed", SeverityHigh,
			fmt.Sprintf("%s has working mail exchangers, so it is a live mailbox domain that correspondents recognise, and its DMARC policy is p=none. Anyone can send mail that appears to come from it and receivers will deliver it.", target),
			"Work towards p=reject: read the aggregate reports, bring every legitimate sender into SPF or DKIM alignment, then raise the policy through quarantine to reject.")
	}

	// A parked domain that sends nothing is spoofable too, and is cheap to
	// close off completely because there is no legitimate mail to break.
	if !receives && res.MX != nil && !res.MX.NullMX && (res.SPF == nil || !res.SPF.Found) {
		b.Warn("cross.unprotected_parked", "Unused domain left open to forgery", SeverityMedium,
			fmt.Sprintf("%s has no mail exchangers and no SPF record. It appears to neither send nor receive mail, but nothing stops a forger from sending as it.", target),
			fmt.Sprintf(`Close it off - there is no legitimate mail to break: %s TXT "v=spf1 -all", %s MX 0 . and _dmarc.%s TXT "v=DMARC1; p=reject; rua=mailto:dmarc@%s".`, target, target, target, orgDomain(target)))
	}

	// A domain locked down for mail is a finished configuration, not a
	// deficient one, and the remaining cross-checks do not apply to it.
	if sendsNoMail(res) {
		if enforcing {
			b.Pass("cross.no_mail", "The domain is locked down for mail",
				fmt.Sprintf("%s authorises no sending source (v=spf1 -all), publishes no DKIM key, and carries a DMARC policy of p=%s. Mail forged in its name is refused, and there is no legitimate mail for that to break.", target, res.DMARC.Effective))
		} else {
			b.Warn("cross.no_mail_unenforced", "Sends no mail, but does not say so under DMARC", SeverityMedium,
				fmt.Sprintf("The SPF record authorises no sender, so %s sends no mail - but its DMARC policy is p=%s, which asks receivers to deliver forged mail anyway. SPF alone is ignored by receivers that honour DMARC.", target, res.DMARC.Effective),
				fmt.Sprintf(`Set the policy to reject: _dmarc.%s TXT "v=DMARC1; p=reject; rua=mailto:dmarc@%s". Nothing legitimate can break, because the domain sends nothing.`, target, orgDomain(target)))
		}
		return
	}

	if spfUsable && dkimUsable && enforcing {
		b.Pass("cross.aligned", "All three mechanisms are in place",
			fmt.Sprintf("SPF resolves within the lookup limit, at least one valid DKIM key is published, and DMARC is at p=%s. Mail from an unauthorised source claiming to be this domain does not reach the recipient.", res.DMARC.Effective))
	}

	// DKIM is what survives forwarding; SPF alone does not.
	if spfUsable && !dkimUsable && res.DMARC != nil && res.DMARC.Found {
		b.Warn("cross.spf_only", "Authentication depends on SPF alone", SeverityMedium,
			"No valid DKIM key was found, so DMARC can only pass through SPF. SPF breaks whenever a message is forwarded, because the forwarding server is not in the record - mailing lists and address-forwarding services are the usual casualties. A DKIM signature travels with the message and survives that.",
			"Enable DKIM signing on every platform that sends as this domain and publish the keys it supplies.")
	}
}

// sendsNoMail reports whether the domain declares that it sends nothing: an
// SPF record authorising no source, and no DKIM key beyond a wildcard denial.
func sendsNoMail(res *EmailAuthResult) bool {
	if res.SPF == nil || !res.SPF.DenyAll || res.DMARC == nil {
		return false
	}
	return res.DKIM == nil || len(res.DKIM.Keys) == 0
}

func emailAuthSummary(res *EmailAuthResult) string {
	parts := make([]string, 0, 3)

	switch {
	case res.SPF == nil || !res.SPF.Found:
		parts = append(parts, "SPF missing")
	case res.SPF.LookupCount > spfMaxLookups:
		parts = append(parts, fmt.Sprintf("SPF over the lookup limit (%d)", res.SPF.LookupCount))
	default:
		parts = append(parts, "SPF ok")
	}

	switch {
	case sendsNoMail(res):
		parts = append(parts, "sends no mail")
	case res.DKIM == nil || len(res.DKIM.Keys) == 0:
		parts = append(parts, "no DKIM key found")
	default:
		parts = append(parts, plural(len(res.DKIM.Keys), "DKIM key", "DKIM keys"))
	}

	if res.DMARC == nil || !res.DMARC.Found {
		parts = append(parts, "DMARC missing")
	} else {
		parts = append(parts, "DMARC p="+res.DMARC.Effective)
	}
	return strings.Join(parts, ", ")
}

func mxProviderFor(host string) string {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	for suffix, name := range mxProviders {
		if h == suffix || strings.HasSuffix(h, "."+suffix) {
			return name
		}
	}
	return ""
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
