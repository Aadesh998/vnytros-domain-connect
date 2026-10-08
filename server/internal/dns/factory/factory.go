package factory

import (
	"domain-connect-backend/internal/dns"
	"domain-connect-backend/internal/dns/providers/aws"
	"domain-connect-backend/internal/dns/providers/hostinger"
	"encoding/json"
	"errors"
	"fmt"
)

func GetProviderFromJSON(providerName string, rawConfig []byte) (dns.DNSProvider, error) {
	switch providerName {
	case "hostinger":
		var cfg hostinger.AuthConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid hostinger config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return hostinger.NewHostinger(cfg), nil

	case "aws":
		var cfg aws.AuthConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid aws config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return aws.NewAWS(cfg), nil

	default:
		return nil, errors.New("unsupported DNS provider")
	}
}
