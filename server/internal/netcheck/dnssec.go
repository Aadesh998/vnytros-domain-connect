package netcheck

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSKEYInfo describes one published key.
type DNSKEYInfo struct {
	KeyTag    uint16 `json:"key_tag"`
	Flags     uint16 `json:"flags"`
	Algorithm string `json:"algorithm"`
	Role      string `json:"role"`
	Bits      int    `json:"bits"`
}

// DSInfo describes one delegation signer record held by the parent.
type DSInfo struct {
	KeyTag     uint16 `json:"key_tag"`
	Algorithm  string `json:"algorithm"`
	DigestType string `json:"digest_type"`
	Digest     string `json:"digest"`
	Matched    bool   `json:"matched"`
}

// SignatureInfo describes an RRSIG covering the zone's records.
type SignatureInfo struct {
	Covers    string `json:"covers"`
	KeyTag    uint16 `json:"key_tag"`
	Algorithm string `json:"algorithm"`
	Expires   string `json:"expires"`
	DaysLeft  int    `json:"days_left"`
	Valid     bool   `json:"valid"`
	Error     string `json:"error,omitempty"`
}

// DNSSECResult is the raw payload for the DNSSEC report.
type DNSSECResult struct {
	Zone           string          `json:"zone"`
	Signed         bool            `json:"signed"`
	ParentHasDS    bool            `json:"parent_has_ds"`
	ChainValid     bool            `json:"chain_valid"`
	ResolverAD     bool            `json:"resolver_validates"`
	Keys           []DNSKEYInfo    `json:"keys"`
	DS             []DSInfo        `json:"ds"`
	Signatures     []SignatureInfo `json:"signatures"`
	ValidationErrs []string        `json:"validation_errors,omitempty"`
}

// DNSSECCheck validates the signing chain for a zone.
func DNSSECCheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("dnssec", target)
	result := &DNSSECResult{Zone: target}

	keyAns, keyErr := r.QueryOpts(ctx, target, dns.TypeDNSKEY, QueryOptions{DNSSEC: true})
	var keys []*dns.DNSKEY
	if keyErr == nil && keyAns != nil {
		for _, rr := range keyAns.RRs {
			if k, ok := rr.(*dns.DNSKEY); ok {
				keys = append(keys, k)
				role := "ZSK (zone-signing key)"
				if k.Flags&dns.SEP != 0 {
					role = "KSK (key-signing key)"
				}
				result.Keys = append(result.Keys, DNSKEYInfo{
					KeyTag:    k.KeyTag(),
					Flags:     k.Flags,
					Algorithm: dns.AlgorithmToString[k.Algorithm],
					Role:      role,
					Bits:      keyBits(k),
				})
			}
		}
	}
	result.Signed = len(keys) > 0

	dsAns, _ := r.QueryOpts(ctx, target, dns.TypeDS, QueryOptions{DNSSEC: true})
	var dsRecords []*dns.DS
	if dsAns != nil {
		for _, rr := range dsAns.RRs {
			if ds, ok := rr.(*dns.DS); ok {
				dsRecords = append(dsRecords, ds)
			}
		}
	}
	result.ParentHasDS = len(dsRecords) > 0

	if !result.Signed && !result.ParentHasDS {
		b.Warn("not_signed", "DNSSEC is not enabled", SeverityLow,
			fmt.Sprintf("%s publishes no DNSKEY records and its parent holds no DS record, so the zone is unsigned.", target),
			"DNSSEC lets resolvers detect forged DNS answers. Enabling it is a one-time setup at most DNS providers, though it must be turned on at the DNS host and the registrar together - enabling only one side breaks the domain.")
		b.Raw(result)
		b.Summary("Zone is not signed")
		return b.Build(), nil
	}

	switch {
	case result.ParentHasDS && !result.Signed:
		b.Fail("ds_without_key", "Parent has a DS record but the zone is unsigned", SeverityCritical,
			fmt.Sprintf("The parent zone publishes %d DS record(s) for %s, but the zone itself publishes no DNSKEY. Every validating resolver treats this as an attack and refuses to resolve the domain.", len(dsRecords), target),
			"This is an active outage for anyone using a validating resolver, even though non-validating resolvers still work. Either remove the DS record at your registrar, or re-enable DNSSEC signing at your DNS host so the DNSKEY comes back.")
		b.Raw(result)
		b.Summary("Broken: DS published but zone unsigned")
		return b.Build(), nil

	case result.Signed && !result.ParentHasDS:
		b.Warn("key_without_ds", "Zone is signed but the parent has no DS record", SeverityMedium,
			fmt.Sprintf("%s publishes DNSKEY records, but no DS record exists at the parent, so resolvers have no trusted starting point and cannot validate the signatures.", target),
			"DNSSEC is effectively off. Publish the DS record at your registrar to complete the chain - most registrars accept either the DS values or the DNSKEY directly.")
	}

	if result.ParentHasDS && result.Signed {
		matchedAny := false
		for _, ds := range dsRecords {
			info := DSInfo{
				KeyTag:     ds.KeyTag,
				Algorithm:  dns.AlgorithmToString[ds.Algorithm],
				DigestType: dns.HashToString[ds.DigestType],
				Digest:     strings.ToUpper(ds.Digest),
			}
			for _, k := range keys {
				if k.Flags&dns.SEP == 0 {
					continue
				}
				computed := k.ToDS(ds.DigestType)
				if computed != nil && strings.EqualFold(computed.Digest, ds.Digest) && computed.KeyTag == ds.KeyTag {
					info.Matched = true
					matchedAny = true
					break
				}
			}
			result.DS = append(result.DS, info)
		}

		if matchedAny {
			b.Pass("ds_match", "Chain of trust is intact",
				fmt.Sprintf("The DS record at the parent matches the key-signing key published by %s.", target))
		} else {
			b.Fail("ds_mismatch", "DS record does not match any key in the zone", SeverityCritical,
				fmt.Sprintf("The parent publishes DS records for key tags %s, but none matches a key-signing key currently in the zone. Validating resolvers cannot build a chain of trust and will refuse to resolve %s.", dsKeyTags(dsRecords), target),
				"This normally happens when keys were rolled at the DNS host without updating the DS record at the registrar, or when DNS providers were changed. Publish the DS values matching the current DNSKEY at your registrar. Until then the domain is unreachable for anyone behind a validating resolver.")
		}
	}

	if result.Signed {
		checkSignatures(ctx, r, target, keys, result, b)
	}

	if adAns, adErr := r.WithServers("1.1.1.1").QueryOpts(ctx, target, dns.TypeSOA, QueryOptions{DNSSEC: true}); adErr == nil && adAns != nil {
		result.ResolverAD = adAns.Authenticated
		if adAns.Authenticated {
			b.Pass("resolver_validates", "Validating resolvers accept this zone",
				"Cloudflare's validating resolver returns the authenticated-data flag, confirming the signatures verify end to end from the root.")
		} else if result.ParentHasDS {
			b.Fail("resolver_rejects", "Validating resolvers do not accept this zone", SeverityHigh,
				"A validating resolver did not set the authenticated-data flag even though a DS record exists, so the chain does not verify from the root.",
				"Re-check that the DS record at the registrar matches the current key-signing key and that all RRSIGs are current. Users behind validating resolvers may be unable to reach this domain.")
		}
	}

	result.ChainValid = b.Has("ds_match") && !b.Has("sig_expired") && !b.Has("sig_invalid")
	b.Raw(result)

	switch {
	case result.ChainValid:
		b.Summary("DNSSEC valid, chain of trust intact")
	case result.Signed:
		b.Summary("DNSSEC enabled but the chain has problems")
	}
	return b.Build(), nil
}

// checkSignatures verifies the RRSIG over the zone's SOA and reports how long
// the signatures remain valid. Signatures expire on a schedule, so a signer
// that silently stops running takes the zone down days later.
func checkSignatures(ctx context.Context, r *Resolver, target string, keys []*dns.DNSKEY, result *DNSSECResult, b *Builder) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(target), dns.TypeSOA)
	msg.SetEdns0(4096, true)

	client := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := client.ExchangeContext(ctx, msg, "1.1.1.1:53")
	if err != nil || resp == nil {
		return
	}
	if resp.Truncated {
		tcp := &dns.Client{Net: "tcp", Timeout: 5 * time.Second}
		if r2, _, e2 := tcp.ExchangeContext(ctx, msg, "1.1.1.1:53"); e2 == nil {
			resp = r2
		}
	}

	var rrset []dns.RR
	var sigs []*dns.RRSIG
	for _, rr := range resp.Answer {
		if sig, ok := rr.(*dns.RRSIG); ok {
			sigs = append(sigs, sig)
			continue
		}
		if rr.Header().Rrtype == dns.TypeSOA {
			rrset = append(rrset, rr)
		}
	}

	if len(sigs) == 0 {
		b.Fail("no_signatures", "Zone records are not signed", SeverityHigh,
			fmt.Sprintf("%s publishes DNSKEY records but its SOA record carries no RRSIG, so there is nothing for a resolver to verify.", target),
			"The signer is not running or has not signed the zone. Re-sign the zone at your DNS provider.")
		return
	}

	keysByTag := map[uint16]*dns.DNSKEY{}
	for _, k := range keys {
		keysByTag[k.KeyTag()] = k
	}

	soonest := 1 << 30
	for _, sig := range sigs {
		info := SignatureInfo{
			Covers:    dns.TypeToString[sig.TypeCovered],
			KeyTag:    sig.KeyTag,
			Algorithm: dns.AlgorithmToString[sig.Algorithm],
		}

		expiry := time.Unix(int64(sig.Expiration), 0).UTC()
		info.Expires = expiry.Format(time.RFC3339)
		info.DaysLeft = int(time.Until(expiry).Hours() / 24)
		if info.DaysLeft < soonest {
			soonest = info.DaysLeft
		}

		if key, ok := keysByTag[sig.KeyTag]; ok && len(rrset) > 0 {
			if verr := sig.Verify(key, rrset); verr != nil {
				info.Error = verr.Error()
				result.ValidationErrs = append(result.ValidationErrs, verr.Error())
			} else {
				info.Valid = true
			}
		} else {
			info.Error = "no matching DNSKEY published for this signature"
			result.ValidationErrs = append(result.ValidationErrs, info.Error)
		}
		result.Signatures = append(result.Signatures, info)
	}

	anyValid := false
	for _, s := range result.Signatures {
		if s.Valid {
			anyValid = true
		}
	}

	switch {
	case !anyValid:
		b.Fail("sig_invalid", "Signatures do not verify", SeverityCritical,
			"None of the RRSIG records over the SOA verify against the published DNSKEY set: "+strings.Join(dedupe(result.ValidationErrs), "; ")+".",
			"Validating resolvers reject every answer from this zone. Re-sign the zone and make sure the DNSKEY set published matches the keys the signer is using.")
	case soonest < 0:
		b.Fail("sig_expired", "Signatures have expired", SeverityCritical,
			fmt.Sprintf("The RRSIG covering the SOA expired %d days ago.", -soonest),
			"Expired signatures are treated as invalid, so validating resolvers refuse to resolve this domain right now. Re-sign the zone immediately and check why the automatic re-signing stopped.")
	case soonest <= 7:

		b.Info("sig_short_lived", "Signatures are short-lived",
			fmt.Sprintf("The RRSIG covering the SOA verifies and expires in %d day(s). Several large providers re-sign continuously with short validity windows, so this is only a problem if the figure keeps dropping across repeated checks rather than resetting.", soonest))
	default:
		b.Pass("signatures_valid", "Signatures are valid",
			fmt.Sprintf("The RRSIG over the SOA verifies and remains valid for %d more days.", soonest))
	}
}

// keyBits estimates the key size for the common algorithms so weak keys are
// visible. Elliptic-curve algorithms have fixed sizes.
func keyBits(k *dns.DNSKEY) int {
	switch k.Algorithm {
	case dns.ECDSAP256SHA256:
		return 256
	case dns.ECDSAP384SHA384:
		return 384
	case dns.ED25519:
		return 256
	case dns.ED448:
		return 448
	}

	raw, err := base64.StdEncoding.DecodeString(k.PublicKey)
	if err != nil || len(raw) < 3 {
		return 0
	}
	expLen, offset := int(raw[0]), 1
	if expLen == 0 {
		expLen = int(raw[1])<<8 | int(raw[2])
		offset = 3
	}
	if expLen == 0 || len(raw) <= offset+expLen {
		return 0
	}
	return len(raw[offset+expLen:]) * 8
}

func dsKeyTags(ds []*dns.DS) string {
	tags := make([]string, 0, len(ds))
	for _, d := range ds {
		tags = append(tags, fmt.Sprintf("%d", d.KeyTag))
	}
	return strings.Join(tags, ", ")
}
