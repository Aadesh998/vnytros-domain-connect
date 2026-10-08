package views

import (
	"encoding/json"
	"strings"
)

type DirectConnectRequest struct {
	Domain         string          `json:"domain"`
	Provider       string          `json:"provider"`
	IP             string          `json:"ip"`
	Target         string          `json:"target"`
	TXT            string          `json:"txt"`
	ProviderConfig json.RawMessage `json:"provider_config"`
}

func (d *DirectConnectRequest) Valid() bool {
	d.Domain = strings.TrimSpace(d.Domain)
	d.Provider = strings.ToLower(strings.TrimSpace(d.Provider))
	return d.Domain != "" && d.Provider != ""
}

type AutoConnectResponse struct {
	ProviderId   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	SettingsURL  string `json:"settings_url"`
	URL          string `json:"url"`
	Type         string `json:"type"`
}

type DomainConnectSettings struct {
	ProviderId          string `json:"providerId"`
	ProviderName        string `json:"providerName"`
	ProviderDisplayName string `json:"providerDisplayName"`
	URLSyncUX           string `json:"urlSyncUX"`
	URLAsyncUX          string `json:"urlAsyncUX"`
	URLAPI              string `json:"urlAPI"`
}
