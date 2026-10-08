package service

import (
	"context"
	"strings"
	"time"

	"domain-connect-backend/internal/netcheck"
)

// NetcheckService exposes the DNS, TLS and HTTP inspection tools.
type NetcheckService struct {
	resolver *netcheck.Resolver
	cache    *netcheck.Cache
}

// toolTimeout bounds a single tool run. The propagation checker fans out to
// 25 resolvers and the TLS checker opens five connections, so this needs
// headroom over a normal request, but it must stay under the server's own
// 60s timeout.
const toolTimeout = 30 * time.Second

// NewNetcheckService builds the service with a 60 second report cache, which
// is short enough that a user re-running a check after a DNS change sees the
// new result quickly.
func NewNetcheckService() *NetcheckService {
	return &NetcheckService{
		resolver: netcheck.NewResolver(),
		cache:    netcheck.NewCache(60*time.Second, 5000),
	}
}

// run wraps a check with caching and a deadline.
func (s *NetcheckService) run(ctx context.Context, key string, fn func(context.Context) (*netcheck.Report, error)) (*netcheck.Report, error) {
	key = strings.ToLower(key)
	if cached, ok := s.cache.Get(key); ok {
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	report, err := fn(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, report)
	return report, nil
}

// Propagation checks a record against resolvers worldwide.
func (s *NetcheckService) Propagation(ctx context.Context, domain, recordType string) (*netcheck.Report, error) {
	return s.run(ctx, "propagation:"+domain+":"+recordType, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.Propagation(c, s.resolver, domain, recordType)
	})
}

// Nameservers compares the delegation, the zone's NS records and SOA serials.
func (s *NetcheckService) Nameservers(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "ns:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.NameserverCheck(c, s.resolver, domain)
	})
}

// CNAME traces the alias chain.
func (s *NetcheckService) CNAME(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "cname:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.CNAMEChain(c, s.resolver, domain)
	})
}

// CAA reports which certificate authorities may issue for the domain.
func (s *NetcheckService) CAA(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "caa:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.CAACheck(c, s.resolver, domain)
	})
}

// SOA inspects the zone's start-of-authority timers and TTLs.
func (s *NetcheckService) SOA(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "soa:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.SOACheck(c, s.resolver, domain)
	})
}

// DNSSEC validates the signing chain.
func (s *NetcheckService) DNSSEC(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "dnssec:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.DNSSECCheck(c, s.resolver, domain)
	})
}

// SPF expands the domain's SPF record and counts its DNS lookups.
func (s *NetcheckService) SPF(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "spf:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.SPFCheck(c, s.resolver, domain)
	})
}

// DKIM discovers and inspects the domain's DKIM keys. Selectors are optional;
// with none supplied the common provider selectors are probed. They form part
// of the cache key so a probe and a named-selector run do not collide.
func (s *NetcheckService) DKIM(ctx context.Context, domain string, selectors []string) (*netcheck.Report, error) {
	return s.run(ctx, "dkim:"+domain+":"+strings.Join(selectors, ","), func(c context.Context) (*netcheck.Report, error) {
		return netcheck.DKIMCheck(c, s.resolver, domain, selectors)
	})
}

// DMARC inspects the domain's DMARC policy and report destinations.
func (s *NetcheckService) DMARC(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "dmarc:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.DMARCCheck(c, s.resolver, domain)
	})
}

// EmailAuth runs SPF, DKIM, DMARC and MX together with the cross-checks
// between them.
func (s *NetcheckService) EmailAuth(ctx context.Context, domain string, selectors []string) (*netcheck.Report, error) {
	return s.run(ctx, "emailauth:"+domain+":"+strings.Join(selectors, ","), func(c context.Context) (*netcheck.Report, error) {
		return netcheck.EmailAuthCheck(c, s.resolver, domain, selectors)
	})
}

// DomainInfo fetches the registration record over RDAP.
func (s *NetcheckService) DomainInfo(ctx context.Context, domain string) (*netcheck.Report, error) {
	return s.run(ctx, "rdap:"+domain, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.DomainInfo(c, domain)
	})
}

// TLS inspects the certificate and TLS configuration a host presents.
func (s *NetcheckService) TLS(ctx context.Context, domain string, port int) (*netcheck.Report, error) {
	return s.run(ctx, "tls:"+domain+":"+itoa(port), func(c context.Context) (*netcheck.Report, error) {
		return netcheck.TLSCheck(c, domain, port)
	})
}

// Headers grades a page's security response headers.
func (s *NetcheckService) Headers(ctx context.Context, url string) (*netcheck.Report, error) {
	return s.run(ctx, "headers:"+url, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.SecurityHeaders(c, url)
	})
}

// Redirects walks a URL's redirect chain.
func (s *NetcheckService) Redirects(ctx context.Context, url string) (*netcheck.Report, error) {
	return s.run(ctx, "redirect:"+url, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.RedirectChain(c, url)
	})
}

// IPInfo describes the addresses behind a hostname.
func (s *NetcheckService) IPInfo(ctx context.Context, target string) (*netcheck.Report, error) {
	return s.run(ctx, "ipinfo:"+target, func(c context.Context) (*netcheck.Report, error) {
		return netcheck.IPInfo(c, s.resolver, target)
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
