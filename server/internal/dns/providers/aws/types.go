package aws

import "errors"

type AuthConfig struct {
	AccessKey string
	SecretKey string
}

func (c AuthConfig) Validate() error {
	if c.AccessKey == "" {
		return errors.New("AWS Access Key is empty")
	}
	if c.SecretKey == "" {
		return errors.New("AWS Secret Key is empty")
	}
	return nil
}
