package dns

import "context"

// DNSProvider is a DNS host that Vnytros can write records to. Only AddRecord
// is implemented by the adapters (aws, hostinger); DeleteRecord, ListRecords
// and TestConnection are part of the interface but are no-op stubs in every
// adapter and must not be relied on.
type DNSProvider interface {
	AddRecord(ctx context.Context, domain string, records []DNSRecord) error
	DeleteRecord(ctx context.Context, domain string, record DNSRecord) error
	ListRecords(ctx context.Context, domain string) ([]DNSRecord, error)
	TestConnection(ctx context.Context) error
}
