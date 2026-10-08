package netcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HeadersResult is the raw payload for the security-header report.
type HeadersResult struct {
	URL           string            `json:"url"`
	FinalURL      string            `json:"final_url"`
	StatusCode    int               `json:"status_code"`
	Server        string            `json:"server,omitempty"`
	Headers       map[string]string `json:"headers"`
	Present       []string          `json:"present"`
	Missing       []string          `json:"missing"`
	HSTSMaxAge    int64             `json:"hsts_max_age,omitempty"`
	PreloadReady  bool              `json:"hsts_preload_ready"`
	CSPDirectives map[string]string `json:"csp_directives,omitempty"`
}

// securityHeaders is the set we grade, in the order they are reported.
var securityHeaders = []string{
	"strict-transport-security",
	"content-security-policy",
	"x-content-type-options",
	"x-frame-options",
	"referrer-policy",
	"permissions-policy",
}

// SecurityHeaders fetches a URL and grades its security response headers.
func SecurityHeaders(ctx context.Context, rawURL string) (*Report, error) {
	u, err := NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	b := NewReport("security_headers", u.String())
	result := &HeadersResult{
		URL:     u.String(),
		Headers: map[string]string{},
	}

	client := SafeHTTPClient(20 * time.Second)

	current := u.String()
	var resp *http.Response
	for hop := 0; hop < 10; hop++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return b.BuildError("bad_request", "Could not build request", err.Error()), nil
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; vnytros-netcheck/1.0)")
		req.Header.Set("Accept", "*/*")

		resp, err = client.Do(req)
		if err != nil {
			if IsBlocked(err) {
				return nil, ErrBlockedTarget
			}
			return b.BuildError("fetch_failed", "Could not fetch the page",
				fmt.Sprintf("Request to %s failed: %v", current, err)), nil
		}
		if loc := resp.Header.Get("Location"); loc != "" && resp.StatusCode >= 300 && resp.StatusCode < 400 {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
			resp.Body.Close()
			next, err := resp.Request.URL.Parse(loc)
			if err != nil {
				break
			}
			current = next.String()
			continue
		}
		break
	}
	defer func() {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		resp.Body.Close()
	}()

	result.FinalURL = current
	result.StatusCode = resp.StatusCode
	result.Server = resp.Header.Get("Server")

	for _, h := range securityHeaders {
		if v := resp.Header.Get(h); v != "" {
			result.Headers[h] = v
			result.Present = append(result.Present, h)
		} else {
			result.Missing = append(result.Missing, h)
		}
	}
	sort.Strings(result.Present)

	if hsts := resp.Header.Get("Strict-Transport-Security"); hsts != "" {
		maxAge, includeSub, preload := parseHSTS(hsts)
		result.HSTSMaxAge = maxAge
		result.PreloadReady = maxAge >= 31536000 && includeSub && preload

		switch {
		case maxAge == 0:
			b.Fail("hsts_zero", "HSTS max-age is zero", SeverityMedium,
				"The Strict-Transport-Security header sets max-age=0, which tells browsers to forget the policy immediately.",
				"Set max-age to at least 31536000 (one year) once you are confident every subdomain serves HTTPS correctly.")
		case maxAge < 15768000:
			b.Warn("hsts_short", "HSTS max-age is short", SeverityLow,
				fmt.Sprintf("max-age is %d seconds (%s).", maxAge, humanDuration(uint32(maxAge))),
				"Six months (15768000) is the minimum most guidance recommends, and one year (31536000) is required for preload eligibility.")
		default:
			b.Pass("hsts", "HSTS enabled",
				fmt.Sprintf("Browsers are told to use HTTPS for %s.", humanDuration(uint32(maxAge))))
		}

		if !includeSub {
			b.Warn("hsts_no_subdomains", "HSTS does not cover subdomains", SeverityLow,
				"The header omits includeSubDomains, so the policy applies only to this exact hostname.",
				"Add includeSubDomains once every subdomain serves HTTPS. Without it, an attacker can still intercept plain HTTP to a subdomain and set cookies that affect the parent domain.")
		}
	} else {
		b.Fail("no_hsts", "No HSTS header", SeverityHigh,
			"The site does not send Strict-Transport-Security, so a browser visiting over plain HTTP for the first time can be intercepted before it is redirected.",
			"Add: Strict-Transport-Security: max-age=31536000; includeSubDomains. Roll it out with a short max-age first and increase it once you are sure every subdomain works over HTTPS, because the policy cannot be revoked early from the browser's cache.")
	}

	if csp := resp.Header.Get("Content-Security-Policy"); csp != "" {
		directives := parseCSP(csp)
		result.CSPDirectives = directives

		scriptSrc := directives["script-src"]
		if scriptSrc == "" {
			scriptSrc = directives["default-src"]
		}

		var weaknesses []string
		if strings.Contains(scriptSrc, "'unsafe-inline'") {
			weaknesses = append(weaknesses, "'unsafe-inline' in script-src allows inline <script> blocks, which is exactly what an XSS payload uses")
		}
		if strings.Contains(scriptSrc, "'unsafe-eval'") {
			weaknesses = append(weaknesses, "'unsafe-eval' permits eval() and string-to-code conversion")
		}
		if strings.Contains(scriptSrc, "*") && !strings.Contains(scriptSrc, "*.") {
			weaknesses = append(weaknesses, "a bare wildcard in script-src allows scripts from any origin")
		}
		if _, ok := directives["default-src"]; !ok {
			weaknesses = append(weaknesses, "no default-src, so directives you have not listed are unrestricted")
		}
		if _, ok := directives["object-src"]; !ok {
			if _, hasDefault := directives["default-src"]; !hasDefault {
				weaknesses = append(weaknesses, "no object-src, so plugin content is unrestricted")
			}
		}
		if _, ok := directives["base-uri"]; !ok {
			weaknesses = append(weaknesses, "no base-uri, so injected <base> tags can redirect every relative URL on the page")
		}

		if len(weaknesses) == 0 {
			b.Pass("csp", "Content Security Policy is restrictive",
				fmt.Sprintf("A CSP is set with %d directives and no obvious weaknesses.", len(directives)))
		} else {
			b.Warn("csp_weak", "Content Security Policy has gaps", SeverityMedium,
				"A CSP is present but weakened: "+strings.Join(weaknesses, "; ")+".",
				"Move inline scripts into files and drop 'unsafe-inline', or adopt a nonce or hash based policy. A CSP with 'unsafe-inline' in script-src stops almost no real XSS, so the header's presence can give false confidence.")
		}
	} else {
		b.Fail("no_csp", "No Content Security Policy", SeverityHigh,
			"The site does not send a Content-Security-Policy header, so the browser places no restriction on where scripts, styles or frames may load from.",
			"CSP is the main defence-in-depth control against cross-site scripting. Start in report-only mode (Content-Security-Policy-Report-Only) to see what would break, then enforce. A reasonable starting point is: default-src 'self'; object-src 'none'; base-uri 'self'.")
	}

	if v := resp.Header.Get("X-Content-Type-Options"); strings.EqualFold(strings.TrimSpace(v), "nosniff") {
		b.Pass("x_content_type_options", "MIME sniffing disabled",
			"X-Content-Type-Options: nosniff stops browsers from second-guessing declared content types.")
	} else {
		b.Warn("no_nosniff", "MIME sniffing not disabled", SeverityMedium,
			"X-Content-Type-Options: nosniff is not set.",
			"Without it, a browser may interpret an uploaded file as script based on its contents rather than its declared type. Add: X-Content-Type-Options: nosniff")
	}

	frameProtected := resp.Header.Get("X-Frame-Options") != ""
	if csp := resp.Header.Get("Content-Security-Policy"); strings.Contains(csp, "frame-ancestors") {
		frameProtected = true
	}
	if frameProtected {
		b.Pass("clickjacking", "Framing is restricted",
			"The page cannot be embedded in a frame by arbitrary sites, which prevents clickjacking.")
	} else {
		b.Warn("no_frame_protection", "No clickjacking protection", SeverityMedium,
			"Neither X-Frame-Options nor a CSP frame-ancestors directive is set, so any site can embed this page in an invisible frame.",
			"Add: X-Frame-Options: DENY, or the modern equivalent frame-ancestors 'none' in your CSP. Use SAMEORIGIN instead if you legitimately frame your own pages.")
	}

	if v := resp.Header.Get("Referrer-Policy"); v != "" {
		b.Pass("referrer_policy", "Referrer policy set", fmt.Sprintf("Referrer-Policy: %s", v))
	} else {
		b.Warn("no_referrer_policy", "No referrer policy", SeverityLow,
			"Referrer-Policy is not set, so full URLs are sent to third-party sites your pages link to or load resources from.",
			"Add: Referrer-Policy: strict-origin-when-cross-origin. This matters when your URLs contain tokens, IDs or search terms.")
	}

	if v := resp.Header.Get("Permissions-Policy"); v != "" {
		b.Pass("permissions_policy", "Permissions policy set", fmt.Sprintf("Permissions-Policy: %s", truncate(v, 120)))
	} else {
		b.Warn("no_permissions_policy", "No permissions policy", SeverityLow,
			"Permissions-Policy is not set, so embedded content may request camera, microphone, geolocation and similar capabilities.",
			"Add a policy denying what you do not use, for example: Permissions-Policy: camera=(), microphone=(), geolocation=()")
	}

	var banners []string
	for _, h := range []string{"Server", "X-Powered-By", "X-AspNet-Version", "X-Generator"} {
		if v := resp.Header.Get(h); v != "" && strings.ContainsAny(v, "0123456789") {
			banners = append(banners, h+": "+v)
		}
	}
	if len(banners) > 0 {
		b.Warn("version_disclosure", "Software versions disclosed", SeverityLow,
			"Response headers reveal specific software versions: "+strings.Join(banners, ", ")+".",
			"Suppress or genericise these headers. They let an attacker match your exact version against known vulnerabilities without probing.")
	}

	b.Raw(result)
	b.Summary(fmt.Sprintf("%d of %d security headers present", len(result.Present), len(securityHeaders)))
	return b.Build(), nil
}

// parseHSTS pulls the directives out of a Strict-Transport-Security header.
func parseHSTS(v string) (maxAge int64, includeSubdomains, preload bool) {
	for _, part := range strings.Split(v, ";") {
		p := strings.ToLower(strings.TrimSpace(part))
		switch {
		case strings.HasPrefix(p, "max-age"):
			if i := strings.Index(p, "="); i >= 0 {
				maxAge, _ = strconv.ParseInt(strings.Trim(strings.TrimSpace(p[i+1:]), `"`), 10, 64)
			}
		case p == "includesubdomains":
			includeSubdomains = true
		case p == "preload":
			preload = true
		}
	}
	return
}

// parseCSP splits a policy into directive name and value.
func parseCSP(v string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(v, ";") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		name := strings.ToLower(fields[0])
		out[name] = strings.Join(fields[1:], " ")
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
