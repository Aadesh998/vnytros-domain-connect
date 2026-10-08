package netcheck

import "testing"

// The parsers are the part of the email checks that can be tested without a
// resolver, and they are where a mistake is silent: a mis-parsed term does not
// error, it produces a confidently wrong report.

func TestParseSPFTerm(t *testing.T) {
	cases := []struct {
		raw       string
		qualifier string
		mechanism string
		modifier  string
		value     string
		wantErr   bool
	}{
		{raw: "-all", qualifier: "-", mechanism: "all"},
		{raw: "all", qualifier: "+", mechanism: "all"},
		{raw: "~all", qualifier: "~", mechanism: "all"},
		{raw: "include:_spf.google.com", qualifier: "+", mechanism: "include", value: "_spf.google.com"},
		{raw: "-include:bad.example", qualifier: "-", mechanism: "include", value: "bad.example"},
		{raw: "ip4:192.0.2.0/24", qualifier: "+", mechanism: "ip4", value: "192.0.2.0/24"},
		{raw: "ip6:2001:db8::/32", qualifier: "+", mechanism: "ip6", value: "2001:db8::/32"},
		{raw: "a", qualifier: "+", mechanism: "a"},
		{raw: "a/24", qualifier: "+", mechanism: "a", value: "/24"},
		{raw: "a:mail.example.com/24", qualifier: "+", mechanism: "a", value: "mail.example.com/24"},
		{raw: "mx", qualifier: "+", mechanism: "mx"},
		{raw: "ptr:example.com", qualifier: "+", mechanism: "ptr", value: "example.com"},
		{raw: "exists:%{i}._spf.example.com", qualifier: "+", mechanism: "exists", value: "%{i}._spf.example.com"},
		{raw: "redirect=_spf.example.com", modifier: "redirect", value: "_spf.example.com"},
		{raw: "exp=explain.example.com", modifier: "exp", value: "explain.example.com"},
		// An unknown modifier is ignored by receivers, not treated as an error.
		{raw: "unknown=value", modifier: "unknown", value: "value"},
		{raw: "banana", wantErr: true},
		{raw: "-banana:x", wantErr: true},
	}

	for _, c := range cases {
		got := parseSPFTerm(c.raw)
		if (got.Error != "") != c.wantErr {
			t.Errorf("%q: error = %q, want error: %v", c.raw, got.Error, c.wantErr)
			continue
		}
		if c.wantErr {
			continue
		}
		if got.Qualifier != c.qualifier || got.Mechanism != c.mechanism ||
			got.Modifier != c.modifier || got.Value != c.value {
			t.Errorf("%q: got qualifier=%q mechanism=%q modifier=%q value=%q, want %q/%q/%q/%q",
				c.raw, got.Qualifier, got.Mechanism, got.Modifier, got.Value,
				c.qualifier, c.mechanism, c.modifier, c.value)
		}
	}
}

// ip4 must not be mistaken for a modifier despite containing no "=", and
// redirect must not be mistaken for a mechanism despite containing no ":".
func TestParseSPFRecordSkipsVersion(t *testing.T) {
	terms := parseSPFRecord("v=spf1 include:example.net ip4:198.51.100.0/24 ~all")
	if len(terms) != 3 {
		t.Fatalf("got %d terms, want 3: %+v", len(terms), terms)
	}
	if terms[0].Mechanism != "include" || terms[2].Mechanism != "all" {
		t.Errorf("unexpected terms: %+v", terms)
	}
}

func TestSPFTermDomain(t *testing.T) {
	cases := []struct{ value, want string }{
		{"", "example.com"},
		{"/24", "example.com"},
		{"mail.example.net", "mail.example.net"},
		{"mail.example.net/24", "mail.example.net"},
		{"mail.example.net.", "mail.example.net"},
		{"%{i}._spf.example.net", ""}, // macros cannot be resolved statically
	}
	for _, c := range cases {
		got := spfTermDomain(SPFTerm{Value: c.value}, "example.com")
		if got != c.want {
			t.Errorf("value %q: got %q, want %q", c.value, got, c.want)
		}
	}
}

func TestParseDKIMKey(t *testing.T) {
	// A real 1024-bit RSA key in SubjectPublicKeyInfo form.
	const p1024 = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDXLybkpDQnCyQYlQc46kL2sPMsYDqwkjPyFRSDiaq6qUeujAlU775+f8rzxIrjHc8sPstAZOuJzqcgDPiidRxdE2Th/lcnfbN4btWvJwHKK6KD/IazNWNqQtunYyPzfmSRTtOE4Nii4Oxwx1glTJAVhCTObD2uh2U44lCG+pgYswIDAQAB"

	t.Run("valid", func(t *testing.T) {
		k := parseDKIMKey("s1", "s1._domainkey.example.com", "", "v=DKIM1; k=rsa; p="+p1024)
		if !k.Valid || k.Revoked || k.Error != "" {
			t.Fatalf("got valid=%v revoked=%v err=%q", k.Valid, k.Revoked, k.Error)
		}
		if k.KeyBits != 1024 || k.KeyType != "rsa" {
			t.Errorf("got %d-bit %s, want 1024-bit rsa", k.KeyBits, k.KeyType)
		}
	})

	t.Run("key material wrapped across lines", func(t *testing.T) {
		// DNS editors routinely re-wrap the payload; verifiers strip the
		// whitespace, and so must this.
		k := parseDKIMKey("s1", "n", "", "v=DKIM1; k=rsa; p="+p1024[:40]+" "+p1024[40:])
		if !k.Valid {
			t.Fatalf("wrapped key rejected: %q", k.Error)
		}
	})

	t.Run("revoked", func(t *testing.T) {
		k := parseDKIMKey("s1", "n", "", "v=DKIM1; k=rsa; p=")
		if !k.Revoked || k.Error != "" {
			t.Errorf("got revoked=%v err=%q, want a clean revocation", k.Revoked, k.Error)
		}
	})

	t.Run("testing and strict flags", func(t *testing.T) {
		k := parseDKIMKey("s1", "n", "", "v=DKIM1; t=y:s; p="+p1024)
		if !k.Testing || !k.StrictSDID {
			t.Errorf("got testing=%v strict=%v, want both", k.Testing, k.StrictSDID)
		}
	})

	t.Run("no p tag", func(t *testing.T) {
		if k := parseDKIMKey("s1", "n", "", "v=DKIM1; k=rsa"); k.Error == "" {
			t.Error("a record with no p tag should be an error, not a key")
		}
	})

	t.Run("malformed base64", func(t *testing.T) {
		if k := parseDKIMKey("s1", "n", "", "v=DKIM1; p=!!!not base64!!!"); k.Error == "" {
			t.Error("malformed base64 should be reported")
		}
	})

	t.Run("truncated key material", func(t *testing.T) {
		if k := parseDKIMKey("s1", "n", "", "v=DKIM1; p="+p1024[:80]); k.Valid {
			t.Error("a truncated key parsed as valid")
		}
	})
}

func TestParseDMARCURIs(t *testing.T) {
	got := parseDMARCURIs("mailto:a@example.com,mailto:b@reports.example.net!10m")
	if len(got) != 2 {
		t.Fatalf("got %d destinations, want 2", len(got))
	}
	if got[0].Scheme != "mailto" || got[0].Domain != "example.com" {
		t.Errorf("first destination: %+v", got[0])
	}
	if got[1].Domain != "reports.example.net" || got[1].MaxSize != "10m" {
		t.Errorf("second destination: %+v", got[1])
	}
	if parseDMARCURIs("  ") != nil {
		t.Error("an empty tag should yield no destinations")
	}
}

func TestDMARCApplyTags(t *testing.T) {
	t.Run("subdomain inherits sp", func(t *testing.T) {
		// A subdomain falling back to the organisational domain's record is
		// governed by sp, not p. Reading p here would overstate protection.
		res := &DMARCResult{Inherited: true, Percent: 100}
		res.Tags = parseDMARCTags("v=DMARC1; p=reject; sp=none; rua=mailto:d@example.com")
		dmarcApplyTags(res)
		if res.Effective != "none" {
			t.Errorf("effective policy = %q, want none", res.Effective)
		}
	})

	t.Run("own record uses p", func(t *testing.T) {
		res := &DMARCResult{Percent: 100}
		res.Tags = parseDMARCTags("v=DMARC1; p=quarantine; pct=50")
		dmarcApplyTags(res)
		if res.Effective != "quarantine" || res.Percent != 50 {
			t.Errorf("got %q at pct=%d, want quarantine at 50", res.Effective, res.Percent)
		}
	})

	t.Run("invalid policy is rejected", func(t *testing.T) {
		res := &DMARCResult{Percent: 100}
		res.Tags = parseDMARCTags("v=DMARC1; p=block")
		dmarcApplyTags(res)
		if len(res.SyntaxErrors) == 0 {
			t.Error("p=block should be a syntax error")
		}
		if res.Effective != "none" {
			t.Errorf("effective policy = %q, want the safe default none", res.Effective)
		}
	})

	t.Run("missing p is reported", func(t *testing.T) {
		res := &DMARCResult{Percent: 100}
		res.Tags = parseDMARCTags("v=DMARC1; rua=mailto:d@example.com")
		dmarcApplyTags(res)
		if len(res.SyntaxErrors) == 0 {
			t.Error("a record without p should be a syntax error")
		}
	})
}

func TestOrgDomain(t *testing.T) {
	cases := map[string]string{
		"example.com":                "example.com",
		"mail.example.com":           "example.com",
		"a.b.c.example.co.uk":        "example.co.uk",
		"deep.sub.example.github.io": "example.github.io",
	}
	for in, want := range cases {
		if got := orgDomain(in); got != want {
			t.Errorf("orgDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMXProviderFor(t *testing.T) {
	cases := map[string]string{
		"aspmx.l.google.com":                      "Google Workspace",
		"example-com.mail.protection.outlook.com": "Microsoft 365",
		"mx.notaprovider.example":                 "",
	}
	for in, want := range cases {
		if got := mxProviderFor(in); got != want {
			t.Errorf("mxProviderFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDMARCStrengthOrdering(t *testing.T) {
	if dmarcStrength("none") >= dmarcStrength("quarantine") ||
		dmarcStrength("quarantine") >= dmarcStrength("reject") {
		t.Error("policies must order none < quarantine < reject")
	}
	if dmarcStrength("nonsense") != 0 {
		t.Error("an unrecognised policy must not outrank a real one")
	}
}
