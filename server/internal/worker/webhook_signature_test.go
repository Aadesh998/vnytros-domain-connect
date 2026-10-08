package worker

import (
	"encoding/json"
	"os"
	"testing"
)

// These constants are mirrored in the JS SDK (sdk/ at the monorepo root) in
// sdk/tests/fixtures/go-signature.json. The JS SDK verifies against that fixture, so if the signing scheme changes on
// either side, one of the two suites fails instead of webhook verification
// silently breaking for every customer.
const (
	fixtureSecret    = "whsec_test_2f8a1c4e6b90d3175ace2846bd09f713"
	fixtureTimestamp = int64(1755500000)
	fixturePayload   = `{"event":"domain.connected","data":{"domain":"shop.acme.com","user_id":42},"timestamp":"2026-08-18T12:00:00Z"}`
	fixtureHeader    = "t=1755500000,v1=43803c27cb22556fcb63678b541bad7285f1694ccdf035d81e0906bd46183552"
)

func TestSignWebhookIsStable(t *testing.T) {
	got := signWebhook(fixtureSecret, fixtureTimestamp, fixturePayload)
	if got != fixtureHeader {
		t.Fatalf("signing scheme changed.\n got: %s\nwant: %s\nIf this is intentional, regenerate the JS fixture from server/ with:\n  FIXTURE_OUT=$PWD/../sdk/tests/fixtures/go-signature.json go test ./internal/worker/ -run TestWriteSignatureFixture", got, fixtureHeader)
	}
}

func TestSignWebhookBindsTimestampAndBody(t *testing.T) {
	base := signWebhook(fixtureSecret, fixtureTimestamp, fixturePayload)

	if signWebhook(fixtureSecret, fixtureTimestamp+1, fixturePayload) == base {
		t.Error("signature ignored the timestamp, so a captured body could be replayed")
	}
	if signWebhook(fixtureSecret, fixtureTimestamp, fixturePayload+" ") == base {
		t.Error("signature ignored the body")
	}
	if signWebhook("other-secret", fixtureTimestamp, fixturePayload) == base {
		t.Error("signature ignored the secret")
	}
}

// TestWriteSignatureFixture regenerates the JS fixture. Opt-in via FIXTURE_OUT.
func TestWriteSignatureFixture(t *testing.T) {
	out := os.Getenv("FIXTURE_OUT")
	if out == "" {
		t.Skip("set FIXTURE_OUT to regenerate the JS SDK fixture")
	}
	b, err := json.MarshalIndent(map[string]any{
		"secret":    fixtureSecret,
		"timestamp": fixtureTimestamp,
		"payload":   fixturePayload,
		"header":    signWebhook(fixtureSecret, fixtureTimestamp, fixturePayload),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
