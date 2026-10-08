package netcheck

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// CertInfo describes one certificate in the presented chain.
type CertInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serial_number"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	IsCA         bool      `json:"is_ca"`
	KeyType      string    `json:"key_type"`
	KeyBits      int       `json:"key_bits"`
	SigAlgorithm string    `json:"signature_algorithm"`
	SANs         []string  `json:"sans,omitempty"`
}

// TLSResult is the raw payload for the certificate report.
type TLSResult struct {
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	ResolvedIP      string     `json:"resolved_ip"`
	Chain           []CertInfo `json:"chain"`
	ChainComplete   bool       `json:"chain_complete"`
	Trusted         bool       `json:"trusted"`
	HostnameMatches bool       `json:"hostname_matches"`
	DaysToExpiry    int        `json:"days_to_expiry"`
	NegotiatedTLS   string     `json:"negotiated_version"`
	CipherSuite     string     `json:"cipher_suite"`
	Protocols       []string   `json:"supported_protocols"`
	OCSPStapled     bool       `json:"ocsp_stapled"`
	SCTCount        int        `json:"sct_count"`
	VerifyError     string     `json:"verify_error,omitempty"`
	HandshakeMS     int64      `json:"handshake_ms"`
}

// tlsVersionNames maps the crypto/tls constants to display names.
var tlsVersionNames = map[uint16]string{
	tls.VersionTLS10: "TLS 1.0",
	tls.VersionTLS11: "TLS 1.1",
	tls.VersionTLS12: "TLS 1.2",
	tls.VersionTLS13: "TLS 1.3",
}

// TLSCheck inspects the certificate and TLS configuration a host presents.
func TLSCheck(ctx context.Context, host string, port int) (*Report, error) {
	target, err := NormalizeDomain(host)
	if err != nil {
		return nil, err
	}
	if port == 0 {
		port = 443
	}

	label := target
	if port != 443 {
		label = fmt.Sprintf("%s:%d", target, port)
	}
	b := NewReport("tls_certificate", label)
	result := &TLSResult{Host: target, Port: port}

	addr := net.JoinHostPort(target, fmt.Sprintf("%d", port))
	dialCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	rawConn, err := safeDialContext(dialCtx, "tcp", addr)
	if err != nil {
		if IsBlocked(err) {
			return nil, ErrBlockedTarget
		}
		return b.BuildError("connect_failed", "Could not connect",
			fmt.Sprintf("Connection to %s failed: %v", addr, err)), nil
	}
	if tcpAddr, ok := rawConn.RemoteAddr().(*net.TCPAddr); ok {
		result.ResolvedIP = tcpAddr.IP.String()
	}

	start := time.Now()
	conn := tls.Client(rawConn, &tls.Config{
		ServerName: target,

		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
	})
	if err := conn.HandshakeContext(dialCtx); err != nil {
		rawConn.Close()
		return b.BuildError("handshake_failed", "TLS handshake failed",
			fmt.Sprintf("The server accepted a TCP connection but the TLS handshake failed: %v. The port may not speak TLS, or the server may require a client certificate.", err)), nil
	}
	result.HandshakeMS = time.Since(start).Milliseconds()

	state := conn.ConnectionState()
	conn.Close()

	if len(state.PeerCertificates) == 0 {
		return b.BuildError("no_certificate", "No certificate presented",
			"The server completed a handshake without presenting any certificate."), nil
	}

	leaf := state.PeerCertificates[0]
	result.NegotiatedTLS = tlsVersionNames[state.Version]
	result.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	result.OCSPStapled = len(state.OCSPResponse) > 0
	result.SCTCount = len(state.SignedCertificateTimestamps)

	for _, c := range state.PeerCertificates {
		result.Chain = append(result.Chain, certInfo(c))
	}

	intermediates := x509.NewCertPool()
	for _, c := range state.PeerCertificates[1:] {
		intermediates.AddCert(c)
	}
	_, verifyErr := leaf.Verify(x509.VerifyOptions{
		DNSName:       target,
		Intermediates: intermediates,
	})
	result.Trusted = verifyErr == nil
	if verifyErr != nil {
		result.VerifyError = verifyErr.Error()
	}
	result.HostnameMatches = leaf.VerifyHostname(target) == nil

	now := time.Now()
	result.DaysToExpiry = int(leaf.NotAfter.Sub(now).Hours() / 24)
	switch {
	case now.Before(leaf.NotBefore):
		b.Fail("not_yet_valid", "Certificate is not valid yet", SeverityCritical,
			fmt.Sprintf("The certificate becomes valid on %s, which is in the future.", leaf.NotBefore.Format("2 January 2006 15:04 MST")),
			"Every client rejects this certificate right now. Check that the server's clock is correct and that the right certificate was installed.")
	case now.After(leaf.NotAfter):
		b.Fail("expired", "Certificate has expired", SeverityCritical,
			fmt.Sprintf("The certificate expired on %s, %d days ago.", leaf.NotAfter.Format("2 January 2006"), -result.DaysToExpiry),
			"Browsers show a full-page security warning and most API clients refuse to connect at all. Renew and install a new certificate now, then set up automatic renewal so it cannot happen again.")
	case result.DaysToExpiry <= 7:
		b.Fail("expiring_urgent", "Certificate expires within a week", SeverityHigh,
			fmt.Sprintf("The certificate expires on %s, in %d days.", leaf.NotAfter.Format("2 January 2006"), result.DaysToExpiry),
			"Renew now. If automatic renewal is configured, check that it is actually running - this is the point at which a silently broken renewal cron becomes an outage.")
	case result.DaysToExpiry <= 30:
		b.Warn("expiring_soon", "Certificate expires within a month", SeverityMedium,
			fmt.Sprintf("The certificate expires on %s, in %d days.", leaf.NotAfter.Format("2 January 2006"), result.DaysToExpiry),
			"Confirm automatic renewal is working. Let's Encrypt certificates renew at 30 days remaining, so a certificate sitting below that threshold suggests renewal is not running.")
	default:
		b.Pass("expiry", "Certificate is current",
			fmt.Sprintf("Valid until %s, %d days away.", leaf.NotAfter.Format("2 January 2006"), result.DaysToExpiry))
	}

	if result.HostnameMatches {
		b.Pass("hostname", "Certificate covers this hostname",
			fmt.Sprintf("%s is listed in the certificate's subject alternative names.", target))
	} else {
		b.Fail("hostname_mismatch", "Certificate does not cover this hostname", SeverityCritical,
			fmt.Sprintf("%s is not listed in the certificate. It covers: %s.", target, strings.Join(truncateList(leaf.DNSNames, 8), ", ")),
			"Clients reject a certificate that does not name the host they asked for. Reissue the certificate including this hostname, or serve the correct certificate for this name - a mismatch often means a default virtual host is answering instead of the intended one.")
	}

	result.ChainComplete = len(state.PeerCertificates) > 1 || isSelfIssued(leaf)
	switch {
	case result.Trusted && len(state.PeerCertificates) > 1:
		b.Pass("chain", "Certificate chain is complete",
			fmt.Sprintf("The server sent %d certificates and the chain validates to a trusted root.", len(state.PeerCertificates)))
	case isSelfIssued(leaf):
		b.Fail("self_signed", "Certificate is self-signed", SeverityCritical,
			"The certificate is signed by itself rather than by a certificate authority, so no client trusts it by default.",
			"Replace it with a certificate from a publicly trusted CA. Let's Encrypt issues them free and automatically.")
	case len(state.PeerCertificates) == 1:
		b.Fail("incomplete_chain", "Intermediate certificate is missing", SeverityHigh,
			"The server sent only the leaf certificate, without the intermediate that links it to a trusted root.",
			"Configure the server to send the full chain (the 'fullchain' file rather than just 'cert'). Chrome and Firefox hide this by fetching the missing certificate themselves, but Android, Java clients, curl and most API libraries do not - so the site appears to work while breaking for a large share of non-browser traffic.")
	case !result.Trusted:
		b.Fail("untrusted", "Certificate is not trusted", SeverityCritical,
			fmt.Sprintf("The chain does not validate against the system trust store: %s", result.VerifyError),
			"Check that the certificate was issued by a publicly trusted CA and that the intermediates the server sends are the correct ones for that issuer.")
	}

	info := result.Chain[0]
	switch {
	case info.KeyType == "RSA" && info.KeyBits < 2048:
		b.Fail("weak_key", "Weak certificate key", SeverityHigh,
			fmt.Sprintf("The certificate uses a %d-bit RSA key.", info.KeyBits),
			"Anything below 2048-bit RSA is rejected by modern browsers. Reissue with at least a 2048-bit RSA key, or an ECDSA P-256 key which is both smaller and faster.")
	case info.KeyType == "RSA":
		b.Pass("key_strength", "Key strength is adequate", fmt.Sprintf("%d-bit RSA key.", info.KeyBits))
	default:
		b.Pass("key_strength", "Key strength is adequate", fmt.Sprintf("%s key (%d-bit).", info.KeyType, info.KeyBits))
	}

	if strings.Contains(strings.ToLower(info.SigAlgorithm), "sha1") {
		b.Fail("weak_signature", "Certificate signed with SHA-1", SeverityHigh,
			fmt.Sprintf("The certificate's signature algorithm is %s.", info.SigAlgorithm),
			"SHA-1 has been considered broken for certificate signatures since 2017 and every major browser rejects it. Reissue with a SHA-256 signature.")
	}

	result.Protocols = probeProtocols(ctx, addr, target)
	var legacy []string
	for _, p := range result.Protocols {
		if p == "TLS 1.0" || p == "TLS 1.1" {
			legacy = append(legacy, p)
		}
	}
	switch {
	case len(result.Protocols) == 0:
		b.Info("protocols", "Protocol support could not be determined",
			"The server did not complete version-specific handshakes, so supported TLS versions are unknown.")
	case len(legacy) > 0:
		b.Fail("legacy_tls", "Obsolete TLS versions enabled", SeverityMedium,
			fmt.Sprintf("The server still accepts %s. Supported versions: %s.", strings.Join(legacy, " and "), strings.Join(result.Protocols, ", ")),
			"TLS 1.0 and 1.1 were deprecated in 2021 and fail PCI DSS compliance. Disable them and require TLS 1.2 as a minimum; there is essentially no client left that needs the older versions.")
	default:
		b.Pass("protocols", "Only modern TLS versions enabled",
			fmt.Sprintf("The server accepts %s.", strings.Join(result.Protocols, ", ")))
	}

	if !contains(result.Protocols, "TLS 1.3") && len(result.Protocols) > 0 {
		b.Warn("no_tls13", "TLS 1.3 not supported", SeverityLow,
			"The server does not accept TLS 1.3.",
			"TLS 1.3 removes the legacy cipher suites entirely and completes the handshake in one round trip instead of two, which is a measurable latency win. Enable it if your server software supports it.")
	}

	if result.OCSPStapled {
		b.Pass("ocsp_stapling", "OCSP stapling enabled",
			"The server staples a revocation response, so clients do not have to contact the CA separately to check whether the certificate is revoked.")
	} else {
		b.Warn("no_ocsp_stapling", "OCSP stapling not enabled", SeverityLow,
			"The server does not staple an OCSP response.",
			"Enabling stapling removes a round trip to the certificate authority during connection setup and avoids leaking which sites your visitors browse to the CA.")
	}

	b.Raw(result)
	b.Summary(fmt.Sprintf("%s, expires in %d days, %s", result.NegotiatedTLS, result.DaysToExpiry, issuerName(leaf)))
	return b.Build(), nil
}

// probeProtocols opens one connection per TLS version, pinning min and max to
// the same value, which is the only reliable way to learn what a server
// accepts rather than what it prefers.
func probeProtocols(ctx context.Context, addr, serverName string) []string {
	versions := []uint16{tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13}
	var mu sync.Mutex
	var supported []string
	var wg sync.WaitGroup

	for _, v := range versions {
		wg.Add(1)
		go func(v uint16) {
			defer wg.Done()
			dctx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()

			raw, err := safeDialContext(dctx, "tcp", addr)
			if err != nil {
				return
			}
			defer raw.Close()

			c := tls.Client(raw, &tls.Config{
				ServerName:         serverName,
				InsecureSkipVerify: true,
				MinVersion:         v,
				MaxVersion:         v,
			})
			if err := c.HandshakeContext(dctx); err != nil {
				return
			}
			c.Close()

			mu.Lock()
			supported = append(supported, tlsVersionNames[v])
			mu.Unlock()
		}(v)
	}
	wg.Wait()

	sort.Strings(supported)
	return supported
}

func certInfo(c *x509.Certificate) CertInfo {
	info := CertInfo{
		Subject:      c.Subject.CommonName,
		Issuer:       c.Issuer.CommonName,
		SerialNumber: c.SerialNumber.String(),
		NotBefore:    c.NotBefore,
		NotAfter:     c.NotAfter,
		IsCA:         c.IsCA,
		SigAlgorithm: c.SignatureAlgorithm.String(),
		SANs:         truncateList(c.DNSNames, 50),
	}
	if info.Subject == "" && len(c.DNSNames) > 0 {
		info.Subject = c.DNSNames[0]
	}

	switch pub := c.PublicKey.(type) {
	case *rsa.PublicKey:
		info.KeyType, info.KeyBits = "RSA", pub.N.BitLen()
	case *ecdsa.PublicKey:
		info.KeyType, info.KeyBits = "ECDSA", pub.Curve.Params().BitSize
	case ed25519.PublicKey:
		info.KeyType, info.KeyBits = "Ed25519", 256
	default:
		info.KeyType = "unknown"
	}
	return info
}

// isSelfIssued reports whether a certificate's subject equals its issuer,
// which is what a self-signed certificate looks like from the outside.
func isSelfIssued(c *x509.Certificate) bool {
	return c.Subject.String() == c.Issuer.String()
}

func issuerName(c *x509.Certificate) string {
	if len(c.Issuer.Organization) > 0 {
		return "issued by " + c.Issuer.Organization[0]
	}
	if c.Issuer.CommonName != "" {
		return "issued by " + c.Issuer.CommonName
	}
	return "issuer unknown"
}

func truncateList(in []string, max int) []string {
	if len(in) <= max {
		return in
	}
	out := append([]string(nil), in[:max]...)
	return append(out, fmt.Sprintf("and %d more", len(in)-max))
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
