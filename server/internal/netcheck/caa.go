package netcheck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

// CAARecord is one parsed CAA entry.
type CAARecord struct {
	Flag     uint8  `json:"flag"`
	Tag      string `json:"tag"`
	Value    string `json:"value"`
	Critical bool   `json:"critical"`
}

// CAAResult is the raw payload for the CAA report.
type CAAResult struct {
	FoundAt      string      `json:"found_at,omitempty"`
	Inherited    bool        `json:"inherited"`
	Records      []CAARecord `json:"records"`
	AllowedCAs   []string    `json:"allowed_cas"`
	AllowedWild  []string    `json:"allowed_wildcard_cas"`
	ReportTo     []string    `json:"report_to"`
	IssuanceOff  bool        `json:"issuance_blocked"`
	SearchedFrom string      `json:"searched_from"`
}

// wellKnownCAs maps the identifiers used in CAA issue tags to readable names.
var wellKnownCAs = map[string]string{
	"letsencrypt.org":      "Let's Encrypt",
	"pki.goog":             "Google Trust Services",
	"digicert.com":         "DigiCert",
	"sectigo.com":          "Sectigo",
	"comodoca.com":         "Sectigo (Comodo)",
	"globalsign.com":       "GlobalSign",
	"amazon.com":           "Amazon (ACM)",
	"amazontrust.com":      "Amazon Trust",
	"awstrust.com":         "Amazon Trust",
	"certainly.com":        "Certainly (Fastly)",
	"buypass.com":          "Buypass",
	"zerossl.com":          "ZeroSSL",
	"actalis.it":           "Actalis",
	"entrust.net":          "Entrust",
	"godaddy.com":          "GoDaddy",
	"starfieldtech.com":    "Starfield",
	"ssl.com":              "SSL.com",
	"trust-provider.com":   "Sectigo",
	"cloudflare.com":       "Cloudflare",
	"apple.com":            "Apple",
	"quovadisglobal.com":   "QuoVadis",
	"identrust.com":        "IdenTrust",
	"harica.gr":            "HARICA",
	"e-szigno.hu":          "Microsec",
	"certum.pl":            "Certum",
	"sslcom.cn":            "SSL.com",
	"trustasia.com":        "TrustAsia",
	"emsign.com":           "eMudhra",
	"firmaprofesional.com": "Firmaprofesional",
}

// CAACheck reports which certificate authorities are permitted to issue for a
// domain.
func CAACheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("caa", target)
	result := &CAAResult{SearchedFrom: target}

	labels := strings.Split(target, ".")
	for i := 0; i < len(labels)-1; i++ {
		name := strings.Join(labels[i:], ".")
		ans, err := r.Query(ctx, name, dns.TypeCAA)
		if err != nil || ans == nil || len(ans.RRs) == 0 {
			continue
		}
		result.FoundAt = name
		result.Inherited = name != target
		for _, rr := range ans.RRs {
			caa, ok := rr.(*dns.CAA)
			if !ok {
				continue
			}
			rec := CAARecord{
				Flag:     caa.Flag,
				Tag:      strings.ToLower(caa.Tag),
				Value:    caa.Value,
				Critical: caa.Flag&0x80 != 0,
			}
			result.Records = append(result.Records, rec)

			switch rec.Tag {
			case "issue":
				if ca := caIdentifier(rec.Value); ca != "" {
					result.AllowedCAs = append(result.AllowedCAs, ca)
				} else {
					result.IssuanceOff = true
				}
			case "issuewild":
				if ca := caIdentifier(rec.Value); ca != "" {
					result.AllowedWild = append(result.AllowedWild, ca)
				}
			case "iodef":
				result.ReportTo = append(result.ReportTo, rec.Value)
			}
		}
		break
	}

	b.Raw(result)

	if len(result.Records) == 0 {
		b.Warn("no_caa", "No CAA record", SeverityMedium,
			fmt.Sprintf("Neither %s nor any of its parent domains publishes a CAA record.", target),
			"Without CAA, any of the ~50 publicly trusted certificate authorities may issue a certificate for this domain. Publishing CAA restricts issuance to the CAs you actually use, which limits the damage from a mis-issuance. Start with a record permitting your current CA, for example: "+target+` CAA 0 issue "letsencrypt.org"`)
		b.Summary("No CAA policy - any CA may issue")
		return b.Build(), nil
	}

	if result.Inherited {
		b.Info("inherited", "CAA policy inherited from a parent",
			fmt.Sprintf("%s has no CAA record of its own, so certificate authorities apply the policy published at %s.", target, result.FoundAt))
	}

	if result.IssuanceOff && len(result.AllowedCAs) == 0 {
		b.Fail("issuance_blocked", "All certificate issuance is blocked", SeverityHigh,
			fmt.Sprintf(`The CAA policy at %s contains issue ";", which forbids every certificate authority from issuing for this domain.`, result.FoundAt),
			"If this is deliberate, nothing to do. If not, no CA can issue or renew a certificate for this domain - renewals will fail and the site will eventually serve an expired certificate. Replace the empty issue value with your CA's identifier.")
	} else if len(result.AllowedCAs) > 0 {
		sort.Strings(result.AllowedCAs)
		b.Pass("issue_policy", "Issuance restricted",
			fmt.Sprintf("Only %s may issue certificates for this domain.", strings.Join(prettyCAs(result.AllowedCAs), ", ")))
	}

	if len(result.AllowedWild) == 0 {
		hasEmptyWild := false
		for _, rec := range result.Records {
			if rec.Tag == "issuewild" && caIdentifier(rec.Value) == "" {
				hasEmptyWild = true
			}
		}
		if hasEmptyWild {
			b.Info("wildcard_blocked", "Wildcard certificates blocked",
				"The policy explicitly forbids wildcard certificate issuance for this domain.")
		} else {
			b.Info("wildcard_inherits", "Wildcards follow the issue policy",
				"No issuewild tag is present, so wildcard certificates are governed by the same CAs listed in the issue tags.")
		}
	} else {
		sort.Strings(result.AllowedWild)
		b.Pass("issuewild_policy", "Wildcard issuance restricted",
			fmt.Sprintf("Only %s may issue wildcard certificates.", strings.Join(prettyCAs(result.AllowedWild), ", ")))
	}

	if len(result.ReportTo) == 0 {
		b.Warn("no_iodef", "No violation reporting address", SeverityLow,
			"The policy has no iodef tag, so you will not be told when a certificate authority is asked to issue a certificate that your policy forbids.",
			`Add an iodef record to be notified of blocked issuance attempts, which are an early warning of an attempted mis-issuance: `+result.FoundAt+` CAA 0 iodef "mailto:security@`+target+`"`)
	} else {
		b.Pass("iodef", "Violation reporting configured",
			fmt.Sprintf("Blocked issuance attempts are reported to %s.", strings.Join(result.ReportTo, ", ")))
	}

	for _, rec := range result.Records {
		if rec.Critical && rec.Tag != "issue" && rec.Tag != "issuewild" && rec.Tag != "iodef" {
			b.Warn("unknown_critical", "Unrecognised critical tag", SeverityMedium,
				fmt.Sprintf("The tag %q is marked critical. A certificate authority that does not understand it is required to refuse issuance entirely.", rec.Tag),
				"Clear the critical flag unless you specifically intend every CA that does not implement this tag to decline issuing.")
		}
	}

	b.Summary(fmt.Sprintf("CAA policy at %s permits %d issuer(s)", result.FoundAt, len(result.AllowedCAs)))
	return b.Build(), nil
}

// caIdentifier extracts the CA domain from an issue value, dropping any
// parameters after the semicolon. An empty result means "no issuer permitted".
func caIdentifier(value string) string {
	v := strings.TrimSpace(value)
	if i := strings.Index(v, ";"); i >= 0 {
		v = v[:i]
	}
	return strings.ToLower(strings.TrimSpace(v))
}

func prettyCAs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := wellKnownCAs[id]; ok {
			out = append(out, fmt.Sprintf("%s (%s)", name, id))
			continue
		}
		out = append(out, id)
	}
	return out
}
