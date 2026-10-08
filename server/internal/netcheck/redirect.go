package netcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RedirectHop is one step in the chain.
type RedirectHop struct {
	Position   int    `json:"position"`
	URL        string `json:"url"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	Location   string `json:"location,omitempty"`
	Scheme     string `json:"scheme"`
	LatencyMS  int64  `json:"latency_ms"`
	Server     string `json:"server,omitempty"`
}

// RedirectResult is the raw payload for the chain report.
type RedirectResult struct {
	Start     string        `json:"start"`
	FinalURL  string        `json:"final_url"`
	FinalCode int           `json:"final_status"`
	Hops      []RedirectHop `json:"hops"`
	TotalMS   int64         `json:"total_ms"`
	LoopAt    string        `json:"loop_at,omitempty"`
	Truncated bool          `json:"truncated"`
	PlainHTTP bool          `json:"plain_http_in_chain"`
}

// maxRedirectHops caps the walk. Browsers give up around 20; anything past a
// handful is already a problem worth reporting.
const maxRedirectHops = 10

// RedirectChain walks a URL's redirects one hop at a time.
func RedirectChain(ctx context.Context, rawURL string) (*Report, error) {
	u, err := NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	b := NewReport("redirect_chain", u.String())
	result := &RedirectResult{Start: u.String()}

	client := SafeHTTPClient(20 * time.Second)
	current := u.String()
	visited := map[string]bool{}

	for i := 0; i < maxRedirectHops; i++ {
		if visited[current] {
			result.LoopAt = current
			break
		}
		visited[current] = true

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return b.BuildError("bad_url", "Invalid URL in chain", err.Error()), nil
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; vnytros-netcheck/1.0)")

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			if IsBlocked(err) {
				return nil, ErrBlockedTarget
			}
			if len(result.Hops) == 0 {
				return b.BuildError("fetch_failed", "Could not fetch the URL",
					fmt.Sprintf("Request to %s failed: %v", current, err)), nil
			}
			b.Fail("hop_failed", "A hop in the chain failed", SeverityHigh,
				fmt.Sprintf("Following the redirect to %s failed: %v", current, err),
				"The chain does not reach a final destination, so visitors see an error page part way through.")
			break
		}
		latency := time.Since(start).Milliseconds()
		result.TotalMS += latency

		hop := RedirectHop{
			Position:   i + 1,
			URL:        current,
			Status:     resp.StatusCode,
			StatusText: http.StatusText(resp.StatusCode),
			Scheme:     schemeOf(current),
			LatencyMS:  latency,
			Server:     resp.Header.Get("Server"),
		}
		if hop.Scheme == "http" {
			result.PlainHTTP = true
		}

		location := resp.Header.Get("Location")
		isRedirect := resp.StatusCode >= 300 && resp.StatusCode < 400 && location != ""
		if isRedirect {
			hop.Location = location
		}
		result.Hops = append(result.Hops, hop)

		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		resp.Body.Close()

		if !isRedirect {
			result.FinalURL = current
			result.FinalCode = resp.StatusCode
			break
		}

		next, err := resp.Request.URL.Parse(location)
		if err != nil {
			b.Fail("bad_location", "Malformed redirect target", SeverityHigh,
				fmt.Sprintf("The Location header at %s could not be parsed: %q", current, location),
				"Send an absolute URL, or a correctly formed relative path, in the Location header.")
			break
		}
		current = next.String()

		if i == maxRedirectHops-1 {
			result.Truncated = true
		}
	}

	b.Raw(result)

	if result.LoopAt != "" {
		b.Fail("loop", "Redirect loop", SeverityCritical,
			fmt.Sprintf("The chain returns to %s, so it never reaches a final page.", result.LoopAt),
			"Browsers abort with ERR_TOO_MANY_REDIRECTS and the page is completely inaccessible. This is usually caused by two rules fighting - a www/non-www rule and an HTTPS rule each undoing the other, or an application redirect racing a web-server redirect.")
		b.Summary("Redirect loop detected")
		return b.Build(), nil
	}

	if result.Truncated {
		b.Fail("too_many", "Too many redirects", SeverityCritical,
			fmt.Sprintf("The chain was still redirecting after %d hops.", maxRedirectHops),
			"Browsers give up at around 20 hops. Collapse the chain into a single redirect to the final destination.")
		b.Summary("Chain exceeded the hop limit")
		return b.Build(), nil
	}

	redirects := len(result.Hops) - 1
	switch {
	case redirects <= 0:
		b.Pass("no_redirects", "No redirects",
			fmt.Sprintf("%s serves content directly with status %d.", result.Start, result.FinalCode))
	case redirects == 1:
		b.Pass("chain_length", "Single redirect",
			fmt.Sprintf("One redirect leads to %s.", result.FinalURL))
	case redirects <= 3:
		b.Warn("chain_length", "Multiple redirects", SeverityLow,
			fmt.Sprintf("%d redirects before reaching %s, costing %dms in total.", redirects, result.FinalURL, result.TotalMS),
			"Each redirect is a full round trip before the page starts loading. Point the first URL straight at the final destination where possible - a common cause is chaining http -> https -> www rather than doing both in one step.")
	default:
		b.Fail("chain_length", "Long redirect chain", SeverityMedium,
			fmt.Sprintf("%d redirects before reaching %s, costing %dms in total.", redirects, result.FinalURL, result.TotalMS),
			"Long chains are slow and dilute the link signals search engines pass through. Collapse them into a single hop.")
	}

	var midChainHTTP []string
	for _, h := range result.Hops {
		if h.Scheme == "http" && h.Position > 1 {
			midChainHTTP = append(midChainHTTP, h.URL)
		}
	}

	switch {
	case len(midChainHTTP) > 0:
		b.Fail("http_downgrade", "Chain downgrades to plain HTTP", SeverityHigh,
			"After the first hop the chain drops back to unencrypted HTTP at: "+strings.Join(midChainHTTP, ", ")+".",
			"Those requests, including any cookies already set for the domain, are readable and modifiable by anyone on the network path. Redirect straight to the final HTTPS URL in one step instead of bouncing through HTTP, and set HSTS so browsers refuse the plain-HTTP hop entirely.")

	case len(result.Hops) > 1 && result.Hops[0].Scheme == "http" && strings.HasPrefix(result.FinalURL, "https://"):
		b.Pass("https_upgrade", "HTTP is upgraded to HTTPS",
			fmt.Sprintf("The plain-HTTP entry point redirects to %s without any further unencrypted hops.", result.FinalURL))

	case strings.HasPrefix(result.Start, "http://") && strings.HasPrefix(result.FinalURL, "http://"):
		b.Fail("no_https", "Site does not redirect to HTTPS", SeverityHigh,
			fmt.Sprintf("%s serves content over plain HTTP and never redirects to an encrypted connection.", result.FinalURL),
			"Everything sent to and from this site travels in clear text. Obtain a certificate (Let's Encrypt issues them free) and redirect all HTTP traffic to HTTPS with a 301.")
	}

	for _, h := range result.Hops {
		if h.Status == http.StatusFound || h.Status == http.StatusTemporaryRedirect {
			b.Warn("temporary_redirect", "Temporary redirect used", SeverityLow,
				fmt.Sprintf("%s returns %d %s, a temporary redirect.", h.URL, h.Status, h.StatusText),
				"If the move is permanent, use 301 (or 308 to preserve the request method). Search engines do not transfer ranking through a temporary redirect, and browsers do not cache it.")
			break
		}
	}

	switch {
	case result.FinalCode == 0:

	case result.FinalCode >= 500:
		b.Fail("final_server_error", "Destination returns a server error", SeverityCritical,
			fmt.Sprintf("The chain ends at %s with status %d.", result.FinalURL, result.FinalCode),
			"The redirect works but the page it leads to is broken.")
	case result.FinalCode >= 400:
		b.Fail("final_client_error", "Destination returns an error", SeverityHigh,
			fmt.Sprintf("The chain ends at %s with status %d %s.", result.FinalURL, result.FinalCode, http.StatusText(result.FinalCode)),
			"The redirect points at a URL that does not serve content. Check the target path still exists.")
	default:
		b.Pass("final_status", "Destination responds successfully",
			fmt.Sprintf("The chain ends at %s with status %d.", result.FinalURL, result.FinalCode))
	}

	if redirects > 0 {
		b.Summary(fmt.Sprintf("%d redirect(s) to %s in %dms", redirects, result.FinalURL, result.TotalMS))
	} else {
		b.Summary(fmt.Sprintf("No redirects, status %d", result.FinalCode))
	}
	return b.Build(), nil
}

func schemeOf(rawURL string) string {
	if i := strings.Index(rawURL, "://"); i > 0 {
		return strings.ToLower(rawURL[:i])
	}
	return ""
}
