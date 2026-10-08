package hostinger

import (
	"bytes"
	"context"
	"domain-connect-backend/internal/dns"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const hostingerAPIBase = "https://api.hostinger.com/api/dns/v1"

type HostingerProvider struct {
	config AuthConfig
	client *http.Client
}

func NewHostinger(config AuthConfig) dns.DNSProvider {
	return &HostingerProvider{
		config: config,
		client: &http.Client{},
	}
}

type HostingerUpdateRequest struct {
	Overwrite bool                  `json:"overwrite"`
	Zone      []HostingerZoneRecord `json:"zone"`
}

type HostingerZoneRecord struct {
	Name    string                   `json:"name"`
	Records []HostingerRecordContent `json:"records"`
	TTL     int                      `json:"ttl"`
	Type    string                   `json:"type"`
}

type HostingerRecordContent struct {
	Content string `json:"content"`
}

func (p *HostingerProvider) AddRecord(ctx context.Context, domain string, records []dns.DNSRecord) error {
	url := fmt.Sprintf("%s/zones/%s", hostingerAPIBase, domain)

	var zoneRecords []HostingerZoneRecord
	for _, record := range records {
		zoneRecords = append(zoneRecords, HostingerZoneRecord{
			Name: record.Name,
			Type: string(record.Type),
			TTL:  int(record.TTL),
			Records: []HostingerRecordContent{
				{
					Content: record.Value,
				},
			},
		})
	}

	payload := HostingerUpdateRequest{
		Overwrite: true,
		Zone:      zoneRecords,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hostinger API error: status %d, response: %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("Hostinger: Successfully applied %d records for %s\n", len(records), domain)
	return nil
}

// DeleteRecord is not implemented; it is a no-op that returns nil.
func (p *HostingerProvider) DeleteRecord(ctx context.Context, domain string, record dns.DNSRecord) error {
	return nil
}

// ListRecords is not implemented; it returns no records and no error.
func (p *HostingerProvider) ListRecords(ctx context.Context, domain string) ([]dns.DNSRecord, error) {
	return nil, nil
}

// TestConnection is not implemented; it always returns nil.
func (p *HostingerProvider) TestConnection(ctx context.Context) error {
	return nil
}
