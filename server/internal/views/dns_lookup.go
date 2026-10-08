package views

type DNSRecordSet struct {
	Type   string   `json:"type"`
	Values []string `json:"values"`
	Error  string   `json:"error,omitempty"`
}

type DNSLookupResponse struct {
	Domain  string         `json:"domain"`
	Records []DNSRecordSet `json:"records"`
}
