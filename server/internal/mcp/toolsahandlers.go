package mcp

import (
	"context"
	"domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/views"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	out, err := json.Marshal(v)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(string(out)), nil
}

func userIDFromCtx(ctx context.Context) (uint, error) {
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return 0, fmt.Errorf("authenticated user not found in context")
	}
	return user.UserID, nil
}

type DomainArgs struct {
	Domain string `json:"domain" jsonschema:"The domain to operate on (e.g. example.com)."`
}

type DomainConnectSettingsArgs struct {
	Domain      string `json:"domain" jsonschema:"The domain to query Domain Connect settings for."`
	DNSProvider string `json:"dns_provider" jsonschema:"The DNS provider name (as returned by domain_provider)."`
}

type DomainStatusArgs struct {
	Domain    string `json:"domain" jsonschema:"The domain to check record status for."`
	IP        string `json:"ip,omitempty" jsonschema:"Expected IPv4 address for the A record."`
	Target    string `json:"target,omitempty" jsonschema:"Expected CNAME target."`
	TXTRecord string `json:"txt_record,omitempty" jsonschema:"Expected TXT record value."`
}

type NoArgs struct{}

func DomainConnectSettingsHandler(ctx context.Context, args DomainConnectSettingsArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	settings, err := domainService.GetDomainConnectSettings(args.Domain, args.DNSProvider)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(settings)
}

func DomainStatusHandler(ctx context.Context, args DomainStatusArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	resp, err := domainService.CheckDomainStatus(views.DomainStatusRequest{
		Domain: args.Domain,
		IP:     args.IP,
		Target: args.Target,
		TXT:    args.TXTRecord,
	})
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(resp)
}

func DomainVerifyHandler(ctx context.Context, args DomainArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return errResult(err.Error()), nil
	}
	resp, err := domainService.VerifyDomain(userID, args.Domain)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(resp)
}

func ListUserDomainsHandler(ctx context.Context, args NoArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return errResult(err.Error()), nil
	}
	domains, err := domainService.ListUserDomains(userID)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(domains)
}

func GetUserDomainHandler(ctx context.Context, args DomainArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return errResult(err.Error()), nil
	}
	d, err := domainService.GetUserDomain(userID, args.Domain)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(d)
}

func DomainTemplateHandler(ctx context.Context, args NoArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	tmpl, err := domainService.GetTemplate()
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(tmpl)
}

func DomainDiscoveryHandler(ctx context.Context, args NoArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	disc, err := domainService.GetDiscovery()
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(disc)
}

func DomainPublicKeyHandler(ctx context.Context, args NoArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	key, err := domainService.GetPublicKey()
	if err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(string(key)), nil
}

func NameServerDetectionHandler(ctx context.Context, args DomainArgs, domainService service.DomainService) (*mcp.CallToolResult, error) {
	domainName, nameservers, provider, err := domainService.DetectProvider(args.Domain)
	if err != nil {
		return errResult(err.Error()), nil
	}

	return jsonResult(map[string]interface{}{
		"domain":      domainName,
		"nameservers": nameservers,
		"provider":    provider,
	})
}
