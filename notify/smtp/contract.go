package smtp

import (
	"errors"
	"github.com/viant/toolbox/url"
)

// SendRequest represents send request.
type SendRequest struct {
	Target *url.Resource `required:"true" description:"SMTP endpoint"`
	Mail   *Message      `required:"true"`
	UDF    string        `description:"body UDF"`
}

// SendResponse represents send response.
type SendResponse struct {
	SendPayloadSize int
}

// Validate validates send request.
func (r *SendRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Target.Credentials == "" {
		return errors.New("credentials was empty")
	}
	if r.Mail == nil {
		return errors.New("mail was nil")
	}
	return r.Mail.Validate()
}
