package netcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// DomainInfoResult is the raw payload for the registration report.
type DomainInfoResult struct {
	Domain       string     `json:"domain"`
	TLD          string     `json:"tld"`
	Source       string     `json:"source"`
	Registrar    string     `json:"registrar"`
	RegistrarID  string     `json:"registrar_id,omitempty"`
	Statuses     []string   `json:"statuses"`
	Nameservers  []string   `json:"nameservers"`
	DNSSEC       bool       `json:"dnssec"`
	Registered   *time.Time `json:"registered,omitempty"`
	Updated      *time.Time `json:"updated,omitempty"`
	Expires      *time.Time `json:"expires,omitempty"`
	DaysToExpiry int        `json:"days_to_expiry,omitempty"`
	AbuseEmail   string     `json:"abuse_email,omitempty"`
	AbusePhone   string     `json:"abuse_phone,omitempty"`
}

// statusMeaning translates EPP status codes, which are the single most
// commonly misread part of a WHOIS record.
var statusMeaning = map[string]string{
	"clienttransferprohibited": "transfer lock is on at the registrar (normal, and a good anti-hijacking measure)",
	"servertransferprohibited": "transfer lock is on at the registry",
	"clientdeleteprohibited":   "the domain cannot be deleted without first removing this lock",
	"serverdeleteprohibited":   "the registry blocks deletion",
	"clientupdateprohibited":   "the domain's details cannot be changed without removing this lock",
	"serverupdateprohibited":   "the registry blocks updates",
	"clientrenewprohibited":    "renewal is blocked at the registrar",
	"clienthold":               "the registrar has removed the domain from DNS - it will not resolve",
	"serverhold":               "the registry has removed the domain from DNS - it will not resolve",
	"pendingdelete":            "the domain is scheduled for deletion",
	"pendingtransfer":          "a transfer to another registrar is in progress",
	"redemptionperiod":         "the domain expired and is in the redemption grace period",
	"autorenewperiod":          "the domain recently auto-renewed",
	"addperiod":                "the domain was registered very recently",
	"inactive":                 "no nameservers are delegated, so the domain does not resolve",
	"ok":                       "no restrictions are set",
	"active":                   "the domain is active",
}

// rdapBootstrap caches IANA's TLD-to-RDAP-endpoint map. The file changes
// rarely, so a daily refresh is plenty and avoids a round trip per lookup.
var rdapBootstrap = struct {
	sync.RWMutex
	services map[string]string
	fetched  time.Time
}{services: map[string]string{}}

// DomainInfo looks up a domain's registration record over RDAP, the
// structured JSON protocol that replaced WHOIS.
func DomainInfo(ctx context.Context, domain string) (*Report, error) {
	target, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	b := NewReport("domain_info", target)
	result := &DomainInfoResult{Domain: target, Source: "rdap"}

	labels := strings.Split(target, ".")
	result.TLD = labels[len(labels)-1]

	base, err := rdapEndpoint(ctx, result.TLD)
	if err != nil {
		return b.BuildError("no_rdap", "No RDAP service for this TLD",
			fmt.Sprintf("The registry for .%s does not publish an RDAP endpoint, so registration data could not be retrieved. Some country-code TLDs still only offer legacy WHOIS on port 43.", result.TLD)), nil
	}

	raw, err := fetchRDAP(ctx, base, target)
	if err != nil {
		return b.BuildError("rdap_failed", "Registration lookup failed",
			fmt.Sprintf("The RDAP service for .%s did not return a usable record: %v", result.TLD, err)), nil
	}

	parseRDAP(raw, result)
	b.Raw(result)

	if result.Expires != nil {
		days := int(time.Until(*result.Expires).Hours() / 24)
		result.DaysToExpiry = days
		switch {
		case days < 0:
			b.Fail("expired", "Domain has expired", SeverityCritical,
				fmt.Sprintf("The registration expired %d days ago, on %s.", -days, result.Expires.Format("2 January 2006")),
				"Renew immediately. Expired domains stop resolving, and once the redemption period ends the name is released for anyone to register.")
		case days <= 7:
			b.Fail("expiring_urgent", "Domain expires within a week", SeverityCritical,
				fmt.Sprintf("The registration expires in %d days, on %s.", days, result.Expires.Format("2 January 2006")),
				"Renew now. If it lapses, the domain stops resolving and every service attached to it - website, email, logins - goes down at once.")
		case days <= 30:
			b.Warn("expiring_soon", "Domain expires within a month", SeverityHigh,
				fmt.Sprintf("The registration expires in %d days, on %s.", days, result.Expires.Format("2 January 2006")),
				"Renew, or confirm auto-renew is enabled and the card on file is valid.")
		case days <= 90:
			b.Warn("expiring", "Domain expires within three months", SeverityLow,
				fmt.Sprintf("The registration expires in %d days, on %s.", days, result.Expires.Format("2 January 2006")),
				"No action needed yet, but check that auto-renew is on.")
		default:
			b.Pass("expiry", "Registration is current",
				fmt.Sprintf("The domain is registered until %s, %d days away.", result.Expires.Format("2 January 2006"), days))
		}
	} else {
		b.Info("no_expiry", "Expiry date not published",
			fmt.Sprintf("The .%s registry does not publish an expiry date in its RDAP record.", result.TLD))
	}

	if result.Registrar != "" {
		b.Info("registrar", "Registrar", fmt.Sprintf("The domain is registered through %s.", result.Registrar))
	}
	if result.Registered != nil {
		age := int(time.Since(*result.Registered).Hours() / 24 / 365)
		b.Info("registered", "First registered",
			fmt.Sprintf("The domain was first registered on %s, about %d years ago.", result.Registered.Format("2 January 2006"), age))
	}

	locked := false
	var serving []string
	for _, s := range result.Statuses {
		key := strings.ToLower(strings.ReplaceAll(s, " ", ""))
		if strings.Contains(key, "transferprohibited") {
			locked = true
		}
		if key == "clienthold" || key == "serverhold" || key == "inactive" || key == "pendingdelete" || key == "redemptionperiod" {
			serving = append(serving, s)
		}
	}

	if locked {
		b.Pass("transfer_lock", "Transfer lock enabled",
			"The domain carries a transfer-prohibited status, so it cannot be moved to another registrar without first unlocking it.")
	} else if len(result.Statuses) > 0 {
		b.Warn("no_transfer_lock", "No transfer lock", SeverityMedium,
			"The domain does not carry a transfer-prohibited status.",
			"Enable the registrar lock in your account. It is free, takes one click, and is the main defence against domain hijacking - an attacker who gets into your registrar account cannot transfer the domain away while it is locked.")
	}

	for _, s := range serving {
		key := strings.ToLower(strings.ReplaceAll(s, " ", ""))
		b.Fail("status_"+key, "Domain status: "+s, SeverityCritical,
			fmt.Sprintf("The registry reports status %q - %s.", s, statusMeaning[key]),
			"This status prevents the domain from resolving normally. Contact your registrar to find out why it was applied.")
	}

	if len(result.Statuses) > 0 {
		var explained []string
		for _, s := range result.Statuses {
			key := strings.ToLower(strings.ReplaceAll(s, " ", ""))
			if m, ok := statusMeaning[key]; ok {
				explained = append(explained, fmt.Sprintf("%s - %s", s, m))
			} else {
				explained = append(explained, s)
			}
		}
		b.Info("statuses", "Registry status codes", strings.Join(explained, "; ")+".")
	}

	if result.DNSSEC {
		b.Pass("dnssec_delegated", "DNSSEC is delegated",
			"The registry holds a DS record for this domain, so the DNSSEC chain of trust is published.")
	}

	if result.AbuseEmail == "" {
		b.Info("no_abuse_contact", "No abuse contact published",
			"The RDAP record does not expose a registrar abuse contact, which is normal since privacy rules redacted most contact data.")
	}

	b.Summary(rdapSummary(result))
	return b.Build(), nil
}

// rdapEndpoint resolves a TLD to its RDAP base URL using IANA's bootstrap
// registry, refreshed daily.
func rdapEndpoint(ctx context.Context, tld string) (string, error) {
	rdapBootstrap.RLock()
	fresh := time.Since(rdapBootstrap.fetched) < 24*time.Hour
	base, ok := rdapBootstrap.services[tld]
	rdapBootstrap.RUnlock()

	if fresh {
		if !ok {
			return "", fmt.Errorf("no RDAP service for .%s", tld)
		}
		return base, nil
	}

	if err := refreshBootstrap(ctx); err != nil {
		rdapBootstrap.RLock()
		base, ok = rdapBootstrap.services[tld]
		rdapBootstrap.RUnlock()
		if ok {
			return base, nil
		}
		return "", err
	}

	rdapBootstrap.RLock()
	base, ok = rdapBootstrap.services[tld]
	rdapBootstrap.RUnlock()
	if !ok {
		return "", fmt.Errorf("no RDAP service for .%s", tld)
	}
	return base, nil
}

func refreshBootstrap(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://data.iana.org/rdap/dns.json", nil)
	if err != nil {
		return err
	}
	resp, err := SafeHTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var doc struct {
		Services [][][]string `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}

	services := make(map[string]string, 1500)
	for _, entry := range doc.Services {
		if len(entry) < 2 || len(entry[1]) == 0 {
			continue
		}
		url := entry[1][0]
		for _, tld := range entry[0] {
			services[strings.ToLower(tld)] = strings.TrimSuffix(url, "/")
		}
	}
	if len(services) == 0 {
		return fmt.Errorf("bootstrap registry was empty")
	}

	rdapBootstrap.Lock()
	rdapBootstrap.services = services
	rdapBootstrap.fetched = time.Now()
	rdapBootstrap.Unlock()
	return nil
}

func fetchRDAP(ctx context.Context, base, domain string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/domain/"+domain, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/rdap+json")
	req.Header.Set("User-Agent", "vnytros-netcheck/1.0")

	resp, err := SafeHTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("domain is not registered")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// parseRDAP pulls the fields we care about out of the RDAP document. The
// format nests contact details inside jCard arrays, which is why the entity
// walk below is more involved than the rest.
func parseRDAP(doc map[string]any, out *DomainInfoResult) {
	if statuses, ok := doc["status"].([]any); ok {
		for _, s := range statuses {
			if str, ok := s.(string); ok {
				out.Statuses = append(out.Statuses, str)
			}
		}
	}

	if events, ok := doc["events"].([]any); ok {
		for _, e := range events {
			ev, ok := e.(map[string]any)
			if !ok {
				continue
			}
			action, _ := ev["eventAction"].(string)
			dateStr, _ := ev["eventDate"].(string)
			t, err := time.Parse(time.RFC3339, dateStr)
			if err != nil {
				continue
			}
			switch strings.ToLower(action) {
			case "registration":
				out.Registered = &t
			case "expiration":
				out.Expires = &t
			case "last changed", "last update of rdap database":
				if out.Updated == nil {
					out.Updated = &t
				}
			}
		}
	}

	if nss, ok := doc["nameservers"].([]any); ok {
		for _, n := range nss {
			ns, ok := n.(map[string]any)
			if !ok {
				continue
			}
			if name, ok := ns["ldhName"].(string); ok {
				out.Nameservers = append(out.Nameservers, strings.ToLower(strings.TrimSuffix(name, ".")))
			}
		}
		sort.Strings(out.Nameservers)
	}

	if sec, ok := doc["secureDNS"].(map[string]any); ok {
		if signed, ok := sec["delegationSigned"].(bool); ok {
			out.DNSSEC = signed
		}
	}

	if entities, ok := doc["entities"].([]any); ok {
		for _, e := range entities {
			ent, ok := e.(map[string]any)
			if !ok {
				continue
			}
			roles := entityRoles(ent)
			if roles["registrar"] {
				out.Registrar = vcardValue(ent, "fn")
				if ids, ok := ent["publicIds"].([]any); ok {
					for _, id := range ids {
						if m, ok := id.(map[string]any); ok {
							if v, ok := m["identifier"].(string); ok {
								out.RegistrarID = v
							}
						}
					}
				}

				if sub, ok := ent["entities"].([]any); ok {
					for _, s := range sub {
						if se, ok := s.(map[string]any); ok && entityRoles(se)["abuse"] {
							out.AbuseEmail = vcardValue(se, "email")
							out.AbusePhone = vcardValue(se, "tel")
						}
					}
				}
			}
			if roles["abuse"] && out.AbuseEmail == "" {
				out.AbuseEmail = vcardValue(ent, "email")
			}
		}
	}
}

func entityRoles(ent map[string]any) map[string]bool {
	out := map[string]bool{}
	if roles, ok := ent["roles"].([]any); ok {
		for _, r := range roles {
			if s, ok := r.(string); ok {
				out[strings.ToLower(s)] = true
			}
		}
	}
	return out
}

// vcardValue digs a named property out of the jCard array RDAP uses for
// contact data: ["vcard", [ ["fn", {}, "text", "Example Registrar"], ... ]].
func vcardValue(ent map[string]any, field string) string {
	arr, ok := ent["vcardArray"].([]any)
	if !ok || len(arr) < 2 {
		return ""
	}
	props, ok := arr[1].([]any)
	if !ok {
		return ""
	}
	for _, p := range props {
		prop, ok := p.([]any)
		if !ok || len(prop) < 4 {
			continue
		}
		name, _ := prop[0].(string)
		if !strings.EqualFold(name, field) {
			continue
		}
		if v, ok := prop[3].(string); ok {
			return v
		}
	}
	return ""
}

func rdapSummary(r *DomainInfoResult) string {
	parts := []string{}
	if r.Registrar != "" {
		parts = append(parts, "registered with "+r.Registrar)
	}
	if r.Expires != nil {
		parts = append(parts, fmt.Sprintf("expires %s", r.Expires.Format("2 Jan 2006")))
	}
	if len(parts) == 0 {
		return "Registration record retrieved"
	}
	return strings.ToUpper(parts[0][:1]) + parts[0][1:] + ", " + strings.Join(parts[1:], ", ")
}
