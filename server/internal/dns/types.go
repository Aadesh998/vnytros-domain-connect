package dns

type RecordType string

const (
	TypeA     RecordType = "A"
	TypeAAAA  RecordType = "AAAA"
	TypeCNAME RecordType = "CNAME"
	TypeMX    RecordType = "MX"
	TypeTXT   RecordType = "TXT"
	TypeSRV   RecordType = "SRV"
)

type DNSRecord struct {
	Type     RecordType `json:"type"`
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	TTL      int64      `json:"ttl"`
	Priority int        `json:"priority,omitempty"`
}

type ProviderConfig interface {
	Validate() error
}
