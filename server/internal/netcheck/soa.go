package netcheck

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// SOAResult is the raw payload for the SOA and TTL report.
type SOAResult struct {
	Zone        string            `json:"zone"`
	PrimaryNS   string            `json:"primary_ns"`
	Hostmaster  string            `json:"hostmaster"`
	Serial      uint32            `json:"serial"`
	SerialStyle string            `json:"serial_style"`
	Refresh     uint32            `json:"refresh"`
	Retry       uint32            `json:"retry"`
	Expire      uint32            `json:"expire"`
	NegativeTTL uint32            `json:"negative_ttl"`
	RecordTTLs  map[string]uint32 `json:"record_ttls"`
}

func SOACheck(ctx context.Context, r *Resolver, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("soa", target)

	ans, err := r.Query(ctx, target, dns.TypeSOA)
	if err != nil || ans == nil || len(ans.RRs) == 0 {
		return b.BuildError("no_soa", "No SOA record",
			fmt.Sprintf("%s has no SOA record. It is probably a subdomain rather than a zone apex, or the domain does not exist.", target)), nil
	}

	soa, ok := ans.RRs[0].(*dns.SOA)
	if !ok {
		return b.BuildError("bad_soa", "Malformed SOA record", "The SOA response could not be parsed."), nil
	}

	result := &SOAResult{
		Zone:        target,
		PrimaryNS:   strings.TrimSuffix(soa.Ns, "."),
		Hostmaster:  hostmasterEmail(soa.Mbox),
		Serial:      soa.Serial,
		Refresh:     soa.Refresh,
		Retry:       soa.Retry,
		Expire:      soa.Expire,
		NegativeTTL: soa.Minttl,
		RecordTTLs:  map[string]uint32{},
	}
	result.SerialStyle, _ = classifySerial(soa.Serial)

	switch {
	case soa.Refresh < 1200:
		b.Warn("refresh_low", "Refresh interval very short", SeverityLow,
			fmt.Sprintf("Secondaries poll the primary every %s.", humanDuration(soa.Refresh)),
			"RFC 1912 suggests 20 minutes to 12 hours. Very short refresh intervals add needless load; most modern setups rely on NOTIFY for prompt updates rather than aggressive polling.")
	case soa.Refresh > 43200:
		b.Warn("refresh_high", "Refresh interval very long", SeverityMedium,
			fmt.Sprintf("Secondaries only poll the primary every %s.", humanDuration(soa.Refresh)),
			"If NOTIFY messages are lost, secondaries can serve stale data for this long. Reduce refresh to between 20 minutes and 12 hours.")
	default:
		b.Pass("refresh", "Refresh interval reasonable", fmt.Sprintf("Secondaries poll every %s.", humanDuration(soa.Refresh)))
	}

	switch {
	case soa.Retry >= soa.Refresh:
		b.Fail("retry_ge_refresh", "Retry is not shorter than refresh", SeverityMedium,
			fmt.Sprintf("Retry is %s but refresh is %s.", humanDuration(soa.Retry), humanDuration(soa.Refresh)),
			"Retry is how long a secondary waits after a failed transfer before trying again, so it must be shorter than the normal refresh interval. Set it to roughly one tenth of refresh.")
	case soa.Retry < 120:
		b.Warn("retry_low", "Retry interval very short", SeverityLow,
			fmt.Sprintf("Failed transfers are retried after %s.", humanDuration(soa.Retry)),
			"Values under two minutes hammer a primary that is already struggling. RFC 1912 suggests at least a few minutes.")
	default:
		b.Pass("retry", "Retry interval reasonable", fmt.Sprintf("Failed transfers retry after %s.", humanDuration(soa.Retry)))
	}

	switch {
	case soa.Expire < 604800:
		b.Warn("expire_low", "Expire time under one week", SeverityMedium,
			fmt.Sprintf("Secondaries stop answering for this zone after %s without contact from the primary.", humanDuration(soa.Expire)),
			"If the primary is unreachable for longer than this, every secondary goes silent and the domain disappears entirely. RFC 1912 recommends two to four weeks so you have time to recover.")
	case soa.Expire > 2592000:
		b.Warn("expire_high", "Expire time over a month", SeverityLow,
			fmt.Sprintf("Secondaries keep serving for %s after losing contact with the primary.", humanDuration(soa.Expire)),
			"Very long expire times mean secondaries can serve badly stale data for weeks. Two to four weeks is the usual range.")
	default:
		b.Pass("expire", "Expire time reasonable",
			fmt.Sprintf("Secondaries keep serving for %s if the primary becomes unreachable.", humanDuration(soa.Expire)))
	}

	switch {
	case soa.Minttl > 86400:
		b.Fail("negttl_high", "Negative cache time over a day", SeverityMedium,
			fmt.Sprintf("A \"record does not exist\" answer is cached for %s.", humanDuration(soa.Minttl)),
			"After you add a record, resolvers that already asked for it keep returning NXDOMAIN for this long. Lower it to one hour; RFC 2308 recommends between one and three hours.")
	case soa.Minttl > 10800:
		b.Warn("negttl_moderate", "Negative cache time above the recommended range", SeverityLow,
			fmt.Sprintf("Non-existent records are cached for %s.", humanDuration(soa.Minttl)),
			"RFC 2308 recommends one to three hours. Longer values delay the visibility of newly added records.")
	case soa.Minttl < 60:
		b.Warn("negttl_low", "Negative cache time very short", SeverityLow,
			fmt.Sprintf("Non-existent records are cached for only %s.", humanDuration(soa.Minttl)),
			"Very low values push repeated lookups for missing records straight to your nameservers.")
	default:
		b.Pass("negative_ttl", "Negative cache time reasonable",
			fmt.Sprintf("Non-existent records are cached for %s.", humanDuration(soa.Minttl)))
	}

	if style, note := classifySerial(soa.Serial); note != "" {
		b.Warn("serial_format", "Unusual serial number", SeverityLow,
			fmt.Sprintf("Serial %d %s", soa.Serial, note),
			"Most operators use the date-based form YYYYMMDDnn, which makes it obvious when the zone last changed and guarantees the value keeps increasing. Serials must never decrease, or secondaries stop accepting updates.")
	} else {
		b.Pass("serial_format", "Serial number well formed",
			fmt.Sprintf("Serial %d is in %s form.", soa.Serial, style))
	}

	if nsAns, nsErr := r.Query(ctx, target, dns.TypeNS); nsErr == nil {
		found := false
		for _, ns := range nsAns.Values {
			if strings.EqualFold(ns, result.PrimaryNS) {
				found = true
				break
			}
		}
		if !found {
			b.Info("hidden_primary", "Primary nameserver is not in the NS set",
				fmt.Sprintf("The SOA names %s as primary, but it is not among the published nameservers. This is a hidden-primary setup, which is a normal and often deliberate design.", result.PrimaryNS))
		}
	}

	if !strings.Contains(result.Hostmaster, "@") {
		b.Warn("hostmaster", "Hostmaster address looks malformed", SeverityLow,
			fmt.Sprintf("The SOA contact field reads %q.", soa.Mbox),
			"The contact field encodes an email address with the first dot standing in for the @ sign. Set it to a mailbox you actually monitor.")
	} else {
		b.Pass("hostmaster", "Hostmaster address present",
			fmt.Sprintf("Zone contact is %s.", result.Hostmaster))
	}

	for _, qtype := range []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT, dns.TypeNS} {
		if a, e := r.Query(ctx, target, qtype); e == nil && a.TTL > 0 {
			result.RecordTTLs[dns.TypeToString[qtype]] = a.TTL
		}
	}
	for name, ttl := range result.RecordTTLs {
		if ttl > 172800 {
			b.Warn("ttl_"+strings.ToLower(name), name+" TTL very long", SeverityLow,
				fmt.Sprintf("The %s record has a TTL of %s.", name, humanDuration(ttl)),
				fmt.Sprintf("Any change to the %s record will take up to %s to reach everyone. Lower the TTL to 300 seconds a day before a planned change, then restore it afterwards.", name, humanDuration(ttl)))
		}
	}

	b.Raw(result)
	b.Summary(fmt.Sprintf("Serial %d, primary %s", soa.Serial, result.PrimaryNS))
	return b.Build(), nil
}

// classifySerial identifies the serial numbering scheme in use and returns a
// note when it looks wrong. An empty note means the serial is fine.
func classifySerial(serial uint32) (style string, note string) {
	s := strconv.FormatUint(uint64(serial), 10)

	if len(s) == 10 {
		year, _ := strconv.Atoi(s[0:4])
		month, _ := strconv.Atoi(s[4:6])
		day, _ := strconv.Atoi(s[6:8])
		if year >= 1990 && year <= 2100 && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			if t.After(time.Now().AddDate(0, 0, 2)) {
				return "date-based", "encodes a date in the future, which will block future updates until that date passes."
			}
			return "date-based (YYYYMMDDnn)", ""
		}
	}

	if serial == 0 {
		return "zero", "is zero, so secondaries have no way to detect that the zone changed."
	}

	if serial < 4294967295 {
		return "counter", ""
	}
	return "counter", "is at the maximum 32-bit value and cannot be incremented further without a rollover procedure."
}

// hostmasterEmail converts the SOA mailbox form (first unescaped dot is the @)
// into a normal email address.
func hostmasterEmail(mbox string) string {
	m := strings.TrimSuffix(mbox, ".")
	if m == "" {
		return ""
	}

	if i := strings.Index(m, `\.`); i >= 0 {
		local := strings.ReplaceAll(m[:i+2], `\.`, ".")
		rest := m[i+2:]
		if j := strings.Index(rest, "."); j >= 0 {
			return local + rest[:j] + "@" + rest[j+1:]
		}
	}
	if i := strings.Index(m, "."); i >= 0 {
		return m[:i] + "@" + m[i+1:]
	}
	return m
}
