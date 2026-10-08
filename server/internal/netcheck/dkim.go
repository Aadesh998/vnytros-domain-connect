package netcheck

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// dkimSelectorProbes is the difficulty at the heart of any DKIM checker: a
// selector is an arbitrary label chosen by the signer and DNS offers no way to
// enumerate the names under _domainkey. The only workable approach is to probe
// the selectors the major providers use by convention. A domain signing with a
// private selector will not be found here, which is why a miss is reported as
// "not discovered" rather than "not configured".
var dkimSelectorProbes = []struct {
	Selector string
	Provider string
}{
	{"google", "Google Workspace"},
	{"selector1", "Microsoft 365"},
	{"selector2", "Microsoft 365"},
	{"s1", "SendGrid / generic"},
	{"s2", "SendGrid / generic"},
	{"k1", "Mailchimp / Mandrill"},
	{"k2", "Mailchimp / Mandrill"},
	{"k3", "Mailchimp / Mandrill"},
	{"mandrill", "Mandrill"},
	{"mte1", "Mailtrap"},
	{"mte2", "Mailtrap"},
	{"dkim", "generic"},
	{"default", "generic"},
	{"mail", "generic"},
	{"email", "generic"},
	{"smtp", "generic"},
	{"key1", "generic"},
	{"key2", "generic"},
	{"selector", "generic"},
	{"pic", "generic"},
	{"pm", "Postmark"},
	{"20161025", "Postmark (legacy)"},
	{"mailjet", "Mailjet"},
	{"zoho", "Zoho Mail"},
	{"zmail", "Zoho Mail"},
	{"protonmail", "Proton Mail"},
	{"protonmail2", "Proton Mail"},
	{"protonmail3", "Proton Mail"},
	{"fm1", "Fastmail"},
	{"fm2", "Fastmail"},
	{"fm3", "Fastmail"},
	{"mesmtp", "Fastmail (legacy)"},
	{"sig1", "iCloud Mail"},
	{"cm", "Campaign Monitor"},
	{"kl", "Klaviyo"},
	{"kl2", "Klaviyo"},
	{"ml", "MailerLite"},
	{"litesrv", "Litmus"},
	{"everlytickey1", "Everlytic"},
	{"everlytickey2", "Everlytic"},
	{"sm", "Sendinblue / Brevo"},
	{"turbo-smtp", "TurboSMTP"},
	{"bfi", "Benchmark"},
	{"ctct1", "Constant Contact"},
	{"ctct2", "Constant Contact"},
	{"freshdesk", "Freshdesk"},
	{"zendesk1", "Zendesk"},
	{"zendesk2", "Zendesk"},
	{"hs1", "HubSpot"},
	{"hs2", "HubSpot"},
	{"beehiiv", "beehiiv"},
	{"mx", "generic"},
	{"dkim1", "generic"},
	{"dk", "generic"},
}

// DKIMKey is one discovered selector and the key published under it.
type DKIMKey struct {
	Selector   string `json:"selector"`
	Name       string `json:"name"`
	Provider   string `json:"provider,omitempty"`
	Record     string `json:"record"`
	Version    string `json:"version,omitempty"`
	KeyType    string `json:"key_type"`
	KeyBits    int    `json:"key_bits,omitempty"`
	Flags      string `json:"flags,omitempty"`
	Hashes     string `json:"hashes,omitempty"`
	Service    string `json:"service,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Testing    bool   `json:"testing"`
	StrictSDID bool   `json:"strict_sdid"`
	Revoked    bool   `json:"revoked"`
	Valid      bool   `json:"valid"`
	Error      string `json:"error,omitempty"`
}

// DKIMResult is the raw payload for the DKIM report.
type DKIMResult struct {
	Domain          string    `json:"domain"`
	Keys            []DKIMKey `json:"keys,omitempty"`
	SelectorsProbed int       `json:"selectors_probed"`
	Wildcard        string    `json:"wildcard_record,omitempty"`
	WildcardDeny    bool      `json:"wildcard_deny"`
	Requested       []string  `json:"requested_selectors,omitempty"`
	Providers       []string  `json:"providers,omitempty"`
	Discovered      bool      `json:"discovered"`
}

// DKIMCheck discovers the DKIM keys published for a domain and inspects each
// one. Selectors may be supplied by the caller; when none are given, a list of
// the selectors the major providers use is probed in parallel.
func DKIMCheck(ctx context.Context, r *Resolver, domain string, selectors []string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("dkim", target)
	res := &DKIMResult{Domain: target}
	b.Raw(res)

	probes := dkimSelectorProbes
	if len(selectors) > 0 {
		probes = probes[:0:0]
		for _, s := range selectors {
			s = strings.TrimSpace(strings.ToLower(s))
			s = strings.TrimSuffix(s, "._domainkey")
			if s == "" {
				continue
			}
			res.Requested = append(res.Requested, s)
			probes = append(probes, struct {
				Selector string
				Provider string
			}{Selector: s})
		}
	}
	res.SelectorsProbed = len(probes)

	// RFC 6376 lets a domain declare that it signs nothing by publishing a
	// revoked key on the *._domainkey wildcard. Every selector probed below
	// then answers with that same record, so the wildcard has to be read
	// first or the probe reports dozens of revoked selectors that do not
	// exist.
	if ans, err := r.Query(ctx, "*._domainkey."+target, dns.TypeTXT); err == nil && ans != nil && len(ans.Values) > 0 {
		res.Wildcard = strings.Join(ans.Values, "")
		wildKey := parseDKIMKey("*", "*._domainkey."+target, "", res.Wildcard)
		res.WildcardDeny = wildKey.Revoked
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		sem  = make(chan struct{}, 12)
		keys []DKIMKey
	)
	for _, p := range probes {
		wg.Add(1)
		go func(selector, provider string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			name := selector + "._domainkey." + target
			ans, err := r.Query(ctx, name, dns.TypeTXT)
			if err != nil || ans == nil || len(ans.Values) == 0 {
				return
			}
			key := parseDKIMKey(selector, name, provider, strings.Join(ans.Values, ""))
			mu.Lock()
			keys = append(keys, key)
			mu.Unlock()
		}(p.Selector, p.Provider)
	}
	wg.Wait()

	if res.Wildcard != "" {
		real := keys[:0]
		for _, k := range keys {
			if k.Record != res.Wildcard {
				real = append(real, k)
			}
		}
		keys = real
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].Selector < keys[j].Selector })
	res.Keys = keys
	res.Discovered = len(keys) > 0
	for _, k := range keys {
		if k.Provider != "" && k.Provider != "generic" {
			res.Providers = append(res.Providers, k.Provider)
		}
	}
	res.Providers = dedupe(res.Providers)

	dkimFindings(b, res, target, len(selectors) > 0)

	switch {
	case len(keys) == 0:
		b.Summary("No DKIM key discovered")
	default:
		b.Summary(fmt.Sprintf("%s published", plural(len(keys), "DKIM key", "DKIM keys")))
	}
	return b.Build(), nil
}

func dkimFindings(b *Builder, res *DKIMResult, target string, explicit bool) {
	if len(res.Keys) == 0 && res.WildcardDeny {
		b.Pass("wildcard_deny", "The domain declares that it signs no mail",
			fmt.Sprintf("*._domainkey.%s publishes a revoked key, which is how RFC 6376 lets a domain state that it signs nothing and that any DKIM signature claiming to be from it should be disbelieved. No individual selector is published.", target))
		return
	}

	if len(res.Keys) == 0 {
		if explicit {
			b.Fail("selector_missing", "No key at the requested selectors", SeverityHigh,
				fmt.Sprintf("None of the selectors you supplied (%s) has a TXT record under _domainkey.%s.", strings.Join(res.Requested, ", "), target),
				"Check the selector name against the one your sending platform shows, then publish the key record it gives you.")
		} else {
			b.Warn("not_discovered", "No DKIM key discovered", SeverityHigh,
				fmt.Sprintf("None of the %d selectors commonly used by mail providers resolves under _domainkey.%s. DNS provides no way to list the selectors a domain publishes, so a key under a private selector would not be found by this probe - but if the domain genuinely has no DKIM key, its mail is signed by nothing and cannot pass DMARC through DKIM alignment.", res.SelectorsProbed, target),
				"Confirm with your sending platform which selector it uses and re-run this check naming it. If no key exists, enable DKIM signing in the platform and publish the record it supplies.")
		}
		return
	}

	live := 0
	for _, k := range res.Keys {
		switch {
		case k.Revoked:
			b.Add(Finding{
				ID:          "revoked_" + k.Selector,
				Title:       fmt.Sprintf("Selector %s is revoked", k.Selector),
				Status:      StatusWarn,
				Severity:    SeverityMedium,
				Detail:      fmt.Sprintf("The record at %s has an empty p= tag, which revokes the key. Signatures made with it are treated as invalid.", k.Name),
				Remediation: "This is correct for a key you have rotated away from, and the record can be deleted once no mail signed with it is still in flight. If this selector is still in use, republish its public key.",
				Evidence:    k,
			})
		case k.Error != "":
			b.Add(Finding{
				ID:          "invalid_" + k.Selector,
				Title:       fmt.Sprintf("Selector %s is malformed", k.Selector),
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Detail:      fmt.Sprintf("The record at %s could not be parsed as a DKIM key: %s. A verifier that cannot read the key treats every signature made with this selector as a failure.", k.Name, k.Error),
				Remediation: "Republish the record exactly as your sending platform supplies it. The most common cause is a key split across multiple TXT records, or one whose base64 payload lost characters when pasted into a DNS editor.",
				Evidence:    k,
			})
		default:
			live++
			switch {
			case k.KeyType == "rsa" && k.KeyBits > 0 && k.KeyBits < 1024:
				b.Add(Finding{
					ID:          "weak_key_" + k.Selector,
					Title:       fmt.Sprintf("Selector %s uses a %d-bit key", k.Selector, k.KeyBits),
					Status:      StatusFail,
					Severity:    SeverityCritical,
					Detail:      fmt.Sprintf("The RSA key at %s is %d bits. RFC 8301 requires verifiers to treat keys below 1024 bits as invalid, so signatures made with this key fail at any conforming receiver.", k.Name, k.KeyBits),
					Remediation: "Rotate to a 2048-bit key. Generate a new selector, publish it, switch signing over, then revoke the old one.",
					Evidence:    k,
				})
			case k.KeyType == "rsa" && k.KeyBits == 1024:
				b.Add(Finding{
					ID:          "key_1024_" + k.Selector,
					Title:       fmt.Sprintf("Selector %s uses a 1024-bit key", k.Selector),
					Status:      StatusWarn,
					Severity:    SeverityMedium,
					Detail:      fmt.Sprintf("The RSA key at %s is 1024 bits, the minimum RFC 8301 accepts and below the 2048 bits it recommends.", k.Name),
					Remediation: "Rotate to 2048 bits when the platform allows it. Some DNS providers cannot hold a 2048-bit key in a single TXT string, in which case it must be published as several quoted strings within one record.",
					Evidence:    k,
				})
			default:
				b.Add(Finding{
					ID:       "key_" + k.Selector,
					Title:    fmt.Sprintf("Selector %s publishes a valid key", k.Selector),
					Status:   StatusPass,
					Severity: SeverityInfo,
					Detail:   fmt.Sprintf("%s carries a valid %s key%s.", k.Name, dkimKeyLabel(k), dkimProviderSuffix(k)),
					Evidence: k,
				})
			}

			if k.Testing {
				b.Warn("testing_"+k.Selector, fmt.Sprintf("Selector %s is in testing mode", k.Selector), SeverityMedium,
					fmt.Sprintf("The record at %s sets t=y, which tells verifiers the domain is still testing DKIM and that they should not treat a failed signature differently from unsigned mail. Left in place, it weakens the signature to the point of being decorative.", k.Name),
					"Remove the t=y tag once signing is confirmed to be working.")
			}
		}
	}

	if live == 0 {
		b.Fail("no_usable_key", "No usable DKIM key", SeverityCritical,
			fmt.Sprintf("Every selector found on %s is revoked or malformed, so no signature this domain produces can be verified.", target),
			"Publish a working key for the selector your sending platform is currently signing with.")
		return
	}

	if len(res.Providers) > 0 {
		b.Info("providers", "Signing services detected",
			fmt.Sprintf("Keys were found for %s.", strings.Join(res.Providers, ", ")))
	}
	if res.Wildcard != "" && !res.WildcardDeny {
		b.Warn("wildcard_key", "A wildcard DKIM key is published", SeverityMedium,
			fmt.Sprintf("*._domainkey.%s publishes a live key, so every selector name resolves to it. Any signature naming any selector verifies against this one key, which removes the ability to rotate or revoke a single sending platform's key without affecting all of them.", target),
			"Publish keys under named selectors and remove the wildcard, keeping it only as a revoked record if you want to declare that unnamed selectors sign nothing.")
	}

	if live == 1 {
		b.Info("single_key", "One active key",
			"Only one usable selector is published. Publishing the replacement selector before switching signing over is what makes a key rotation possible without a gap in verifiable mail.")
	}
}

// parseDKIMKey parses a DKIM key record into its tags and validates the public
// key material, which is what distinguishes a key that verifiers accept from
// one that merely looks present in DNS.
func parseDKIMKey(selector, name, provider, record string) DKIMKey {
	k := DKIMKey{
		Selector: selector,
		Name:     name,
		Provider: provider,
		Record:   record,
		KeyType:  "rsa", // the default when no k= tag is present
	}

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
		tags[strings.TrimSpace(strings.ToLower(part[:eq]))] = strings.TrimSpace(part[eq+1:])
	}

	k.Version = tags["v"]
	if v, ok := tags["k"]; ok && v != "" {
		k.KeyType = strings.ToLower(v)
	}
	k.Hashes = tags["h"]
	k.Service = tags["s"]
	k.Notes = tags["n"]
	k.Flags = tags["t"]
	for _, f := range strings.Split(strings.ToLower(k.Flags), ":") {
		switch strings.TrimSpace(f) {
		case "y":
			k.Testing = true
		case "s":
			k.StrictSDID = true
		}
	}

	if k.Version != "" && !strings.EqualFold(k.Version, "DKIM1") {
		k.Error = fmt.Sprintf("unrecognised version %q", k.Version)
		return k
	}

	pub, hasP := tags["p"]
	pub = strings.Join(strings.Fields(pub), "")
	switch {
	case !hasP:
		k.Error = "no p= tag, so the record carries no public key"
		return k
	case pub == "":
		k.Revoked = true
		return k
	}

	der, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		k.Error = "the p= tag is not valid base64, which usually means the record was truncated or re-wrapped when it was pasted into DNS"
		return k
	}

	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		// Some providers publish a bare PKCS#1 RSA key rather than the
		// SubjectPublicKeyInfo form the RFC asks for. Verifiers in the wild
		// accept both, so this check does too.
		if rsaKey, pkcs1Err := x509.ParsePKCS1PublicKey(der); pkcs1Err == nil {
			k.Valid = true
			k.KeyBits = rsaKey.N.BitLen()
			return k
		}
		k.Error = "the public key is not a well-formed DER key"
		return k
	}

	switch key := parsed.(type) {
	case *rsa.PublicKey:
		k.Valid = true
		k.KeyType = "rsa"
		k.KeyBits = key.N.BitLen()
	case ed25519.PublicKey:
		k.Valid = true
		k.KeyType = "ed25519"
		k.KeyBits = 256
	default:
		k.Error = "the public key is not an algorithm DKIM permits"
	}
	return k
}

func dkimKeyLabel(k DKIMKey) string {
	if k.KeyType == "ed25519" {
		return "Ed25519"
	}
	if k.KeyBits > 0 {
		return fmt.Sprintf("%d-bit RSA", k.KeyBits)
	}
	return strings.ToUpper(k.KeyType)
}

func dkimProviderSuffix(k DKIMKey) string {
	if k.Provider == "" || k.Provider == "generic" {
		return ""
	}
	return " (" + k.Provider + ")"
}
