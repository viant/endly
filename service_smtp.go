package endly

import (
	"crypto/tls"
	"fmt"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"net/smtp"
)

const (
	//SMTPServiceID represents smtp service id.
	SMTPServiceID = "smtp"
)

//no operation service
type smtpService struct {
	*AbstractService
}

//NewSMTPClient creates a new SMTP client.
func NewSMTPClient(target *url.Resource, credentialsFile string) (*smtp.Client, error) {
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

func (s *smtpService) send(context *Context, request *SMTPSendRequest) (*SMTPSendResponse, error) {
	var response = &SMTPSendResponse{}
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	client, err := NewSMTPClient(target, target.Credential)
	if err != nil {
		return nil, err
	}
	defer client.Quit()
	mainMessage := request.Mail
	if err = client.Mail(mainMessage.From); err != nil {
		return nil, fmt.Errorf("failed to mainMessage: sender: %v, %v", mainMessage.From, err)
	}
	for _, receiver := range mainMessage.Receivers() {
		if err = client.Rcpt(receiver); err != nil {
			return nil, err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	var payload = mainMessage.Payload()
	payload = []byte(context.Expand(string(payload)))
	response.SendPayloadSize, err = writer.Write(payload)
	if err != nil {
		return nil, err
	}
	return response, nil
}

const sMTPSendExample = `{
  "Target": {
    "URL": "smtp://smtp.gmail.com:465",
    "Credential": "${env.HOME}/.secret/smtp.json"
  },
  "Mail": {
    "From": "sender@gmail.com",
    "To": [
      "abc@gmail.com"
    ],
    "Subject": "Subject",
    "Body": "<h1>Header</h1><p>message</p>",
    "ContentType": "text/html"
  }
}`

func (s *smtpService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "send",
		RequestInfo: &ActionInfo{
			Description: "send an email",
			Examples: []*ExampleUseCase{
				{
					UseCase: "email send",
					Data:    sMTPSendExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SMTPSendRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SMTPSendRequest{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SMTPSendRequest); ok {
				return s.send(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewSMTPService creates a new NoOperation service.
func NewSMTPService() Service {
	var result = &smtpService{
		AbstractService: NewAbstractService(SMTPServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
