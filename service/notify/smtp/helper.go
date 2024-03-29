package smtp

import (
	"crypto/tls"
	"fmt"
	"github.com/viant/endly/model/location"
	"github.com/viant/scy/cred"
	"net/smtp"
)

// NewClient creates a new SMTP client.
func NewClient(target *location.Resource, credConfig *cred.Generic) (*smtp.Client, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         target.Host(),
	}
	auth := smtp.PlainAuth("", credConfig.Username, credConfig.Password, target.Hostname())

	conn, err := tls.Dial("tcp", target.Hostname(), tlsConfig)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, target.Hostname())
	if err != nil {
		return nil, err
	}
	if err = client.Auth(auth); err != nil {
		return nil, fmt.Errorf("failed to auth with %v, %v", credConfig.Username, err)
	}
	return client, nil
}
