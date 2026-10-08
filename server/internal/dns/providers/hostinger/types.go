package hostinger

import "errors"

type AuthConfig struct {
	APIToken string
}

func (c AuthConfig) Validate() error {
	if c.APIToken == "" {
		return errors.New("API Token is empty")
	}
	return nil
}
