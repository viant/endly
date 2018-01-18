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

	//SMTPServiceSendAction represents smtp action
	SMTPServiceSendAction = "send"
)

//no operation service
type smtpService struct {
	*AbstractService
}

func (s *smtpService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: request}
	var err error
	switch actualRequest := request.(type) {
	case *SMTPSendRequest:
		response.Response, err = s.send(context, actualRequest)

	default:
		err = fmt.Errorf("unsupported request type: %T", request)
	}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	}
	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *smtpService) NewRequest(action string) (interface{}, error) {

	switch action {
	case SMTPServiceSendAction:
		return &SMTPSendRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
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

	client, err := NewSMTPClient(target, request.Credential)
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

//NewSMTPService creates a new NoOperation service.
func NewSMTPService() Service {
	var result = &smtpService{
		AbstractService: NewAbstractService(SMTPServiceID,
			SMTPServiceSendAction),
	}
	result.AbstractService.Service = result
	return result
}
