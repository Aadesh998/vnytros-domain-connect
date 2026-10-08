// Package migrations holds reference data that cmd/migration loads after the
// schema is in place.
package migrations

import _ "embed"

// DNSProviders upserts the DNS provider table used by provider detection
// (nameserver patterns per provider). It uses ON CONFLICT (id) DO UPDATE, so
// it is idempotent: safe on fresh installs and on every re-run.
//
//go:embed dns_provider.sql
var DNSProviders string
