package views

import (
	"strings"
)

type DomainStatusRequest struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
	Target string `json:"target"`
	TXT    string `json:"txt"`
}

func (d *DomainStatusRequest) Valid() bool {
	d.Domain = strings.TrimSpace(d.Domain)
	return d.Domain != ""
}

type RecordStatus struct {
	Type     string   `json:"type"`
	Host     string   `json:"host"`
	Expected string   `json:"expected"`
	Current  []string `json:"current"`
	Status   bool     `json:"status"`
}

type DomainStatusResponse struct {
	Domain       string         `json:"domain"`
	IsConfigured bool           `json:"is_configured"`
	Records      []RecordStatus `json:"records"`
}

type DomainVerifyRequest struct {
	Domain string `json:"domain"`
}

func (d *DomainVerifyRequest) Valid() bool {
	d.Domain = strings.TrimSpace(d.Domain)
	return d.Domain != ""
}

type DomainVerifyResponse struct {
	Domain     string `json:"domain"`
	Status     string `json:"status"`
	IPMatched  bool   `json:"ip_matched"`
	TXTMatched bool   `json:"txt_matched"`
	CNAMEMatch bool   `json:"cname_matched"`
	Message    string `json:"message"`
}
