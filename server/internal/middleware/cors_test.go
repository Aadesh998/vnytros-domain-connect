package middleware

import "testing"

func TestOriginMatches(t *testing.T) {
	t.Parallel()

	allowed := []string{"http://localhost:5173", "*.example.com", ""}
	cases := map[string]bool{
		"http://localhost:5173":       true,
		"http://localhost:3000":       false,
		"https://example.com":         true,
		"https://app.example.com":     true,
		"https://evil-example.com":    false,
		"https://example.com.evil.io": false,
		"":                            false,
		"not a url":                   false,
	}
	for origin, want := range cases {
		if got := originMatches(origin, allowed); got != want {
			t.Errorf("originMatches(%q) = %v, want %v", origin, got, want)
		}
	}
}
