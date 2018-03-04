package smtp

import (
	"net/smtp"
	"github.com/viant/toolbox/cred"
	"crypto/tls"
	"fmt"
	"github.com/viant/toolbox/url"
)

//NewClient creates a new SMTP client.
func NewClient(target *url.Resource, credentialsFile string) (*smtp.Client, error) {
	credential, err := cred.NewConfig(credentialsFile)
	if err != nil {
		return nil, err
	}
	var targetURL = target.ParsedURL
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         targetURL.Host,
	}
	auth := smtp.PlainAuth("", credential.Username, credential.Password, targetURL.Host)
	conn, err := tls.Dial("tcp", targetURL.Host, tlsConfig)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, targetURL.Host)
	if err != nil {
		return nil, err
	}

	if err = client.Auth(auth); err != nil {
		return nil, fmt.Errorf("failed to auth with %v, %v", credential.Username, err)
	}
	return client, nil
}
