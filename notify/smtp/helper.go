package smtp

import (
	"crypto/tls"
	"fmt"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"net/smtp"
)

//NewClient creates a new SMTP client.
func NewClient(target *url.Resource, credConfig *cred.Config) (*smtp.Client, error) {
	var targetURL = target.ParsedURL
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         targetURL.Host,
	}
	auth := smtp.PlainAuth("", credConfig.Username, credConfig.Password, targetURL.Host)

	conn, err := tls.Dial("tcp", targetURL.Host, tlsConfig)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, targetURL.Host)
	if err != nil {
		return nil, err
	}
	if err = client.Auth(auth); err != nil {
		return nil, fmt.Errorf("failed to auth with %v, %v", credConfig.Username, err)
	}
	return client, nil
}
