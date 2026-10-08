package utils

// TemplateFile is the Domain Connect template served for providerDomain
// (DC_PROVIDER_DOMAIN): ./templates/<providerDomain>.custom-domain.json.
func TemplateFile(providerDomain string) string {
	return "./templates/" + providerDomain + ".custom-domain.json"
}

const PublicKeyFile = "./keys/public_key.pem"

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)
