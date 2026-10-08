package mcp

import (
	"context"

	"domain-connect-backend/internal/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolDoc describes a registered tool for the landing page served on "/".
type ToolDoc struct {
	Name        string
	Description string
	Params      []string
	Auth        bool
}

var toolDocs = []ToolDoc{
	{
		Name:        "domain_provider",
		Description: "Detect the DNS / domain record management provider for a given domain.",
		Params:      []string{"domain"},
	},
	{
		Name:        "domain_connect_settings",
		Description: "Fetch the Domain Connect settings (sync/async URLs, provider info) for a given domain and DNS provider.",
		Params:      []string{"domain", "dns_provider"},
	},
	{
		Name:        "domain_status",
		Description: "Check whether the expected A, CNAME, and TXT records are currently configured on a domain.",
		Params:      []string{"domain", "ip?", "target?", "txt_record?"},
	},
	{
		Name:        "domain_verify",
		Description: "Verify ownership and configuration of a domain previously linked by the authenticated user.",
		Params:      []string{"domain"},
		Auth:        true,
	},
	{
		Name:        "list_user_domains",
		Description: "List all domains linked by the authenticated user.",
		Auth:        true,
	},
	{
		Name:        "get_user_domain",
		Description: "Get a single linked domain record for the authenticated user.",
		Params:      []string{"domain"},
		Auth:        true,
	},
	{
		Name:        "domain_template",
		Description: "Return the Domain Connect template served by this service.",
	},
	{
		Name:        "domain_discovery",
		Description: "Return the Domain Connect discovery document served by this service.",
	},
	{
		Name:        "domain_public_key",
		Description: "Return the public key used to sign Domain Connect template requests.",
	},
}

// ToolDocs returns documentation for every tool registered by RegisterTools.
func ToolDocs() []ToolDoc { return toolDocs }

// describe looks up the description for a tool so the landing page and the
// registered tool metadata cannot drift apart.
func describe(name string) string {
	for _, d := range toolDocs {
		if d.Name == name {
			return d.Description
		}
	}
	return ""
}

func RegisterTools(s *mcp.Server, domainService service.DomainService) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_provider",
		Description: describe("domain_provider"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DomainArgs) (*mcp.CallToolResult, any, error) {
		res, err := NameServerDetectionHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_connect_settings",
		Description: describe("domain_connect_settings"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DomainConnectSettingsArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainConnectSettingsHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_status",
		Description: describe("domain_status"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DomainStatusArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainStatusHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_verify",
		Description: describe("domain_verify"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DomainArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainVerifyHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_user_domains",
		Description: describe("list_user_domains"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args NoArgs) (*mcp.CallToolResult, any, error) {
		res, err := ListUserDomainsHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user_domain",
		Description: describe("get_user_domain"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DomainArgs) (*mcp.CallToolResult, any, error) {
		res, err := GetUserDomainHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_template",
		Description: describe("domain_template"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args NoArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainTemplateHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_discovery",
		Description: describe("domain_discovery"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args NoArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainDiscoveryHandler(ctx, args, domainService)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "domain_public_key",
		Description: describe("domain_public_key"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args NoArgs) (*mcp.CallToolResult, any, error) {
		res, err := DomainPublicKeyHandler(ctx, args, domainService)
		return res, nil, err
	})
}
