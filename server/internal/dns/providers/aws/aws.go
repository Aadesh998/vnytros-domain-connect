package aws

import (
	"context"
	"domain-connect-backend/internal/dns"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type AWSProvider struct {
	config AuthConfig
}

type AWSClient struct {
	r53 *route53.Client
}

func NewAWS(config AuthConfig) dns.DNSProvider {
	return &AWSProvider{
		config: config,
	}
}

func New(ctx context.Context, acessKey string, secretKey string) (*AWSClient, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				acessKey,
				secretKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &AWSClient{
		r53: route53.NewFromConfig(cfg),
	}, nil
}

func (c *AWSClient) ValidateCredential(ctx context.Context) error {
	_, err := c.r53.ListHostedZones(
		ctx,
		&route53.ListHostedZonesInput{},
	)
	if err != nil {
		return fmt.Errorf("invalid credential: %w", err)
	}
	return nil
}

func (c *AWSClient) GetDomainZoneID(ctx context.Context, domain string) (string, error) {
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}

	results, err := c.r53.ListHostedZonesByName(
		ctx,
		&route53.ListHostedZonesByNameInput{
			DNSName: aws.String(domain),
		},
	)

	if err != nil {
		return "", err
	}

	for _, zone := range results.HostedZones {
		if aws.ToString(zone.Name) == domain {
			id := aws.ToString(zone.Id)
			parts := strings.Split(id, "/")
			return parts[len(parts)-1], nil
		}
	}
	return "", fmt.Errorf("zone not found for domain: %s", domain)
}

func (p *AWSProvider) AddRecord(ctx context.Context, domain string, records []dns.DNSRecord) error {
	c, err := New(ctx, p.config.AccessKey, p.config.SecretKey)
	if err != nil {
		return err
	}

	err = c.ValidateCredential(ctx)
	if err != nil {
		return err
	}

	zoneID, err := c.GetDomainZoneID(ctx, domain)
	if err != nil {
		return err
	}

	var changes []types.Change
	for _, record := range records {
		name := record.Name
		if name == "@" || name == "" {
			name = domain
		} else {
			name = record.Name + "." + domain
		}

		// Route53 requires TXT values to be enclosed in double quotes.
		value := record.Value
		if record.Type == dns.TypeTXT {
			value = `"` + record.Value + `"`
		}

		changes = append(changes, types.Change{
			Action: types.ChangeActionUpsert,
			ResourceRecordSet: &types.ResourceRecordSet{
				Name: aws.String(name),
				Type: types.RRType(record.Type),
				TTL:  aws.Int64(record.TTL),
				ResourceRecords: []types.ResourceRecord{
					{Value: aws.String(value)},
				},
			},
		})
	}

	_, err = c.r53.ChangeResourceRecordSets(
		ctx,
		&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneID),
			ChangeBatch: &types.ChangeBatch{
				Comment: aws.String("Applied by Vnytros"),
				Changes: changes,
			},
		},
	)
	return err
}

// DeleteRecord is not implemented; it is a no-op that returns nil.
func (p *AWSProvider) DeleteRecord(ctx context.Context, domain string, record dns.DNSRecord) error {
	return nil
}

// ListRecords is not implemented; it returns no records and no error.
func (p *AWSProvider) ListRecords(ctx context.Context, domain string) ([]dns.DNSRecord, error) {
	return nil, nil
}

// TestConnection is not implemented; it always returns nil.
func (p *AWSProvider) TestConnection(ctx context.Context) error {
	return nil
}
