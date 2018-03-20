package smtp

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

const (
	//ServiceID represents smtp service id.
	ServiceID = "smtp"
)

//service represent SMTP service
type service struct {
	*endly.AbstractService
}

func (s *service) send(context *endly.Context, request *SendRequest) (*SendResponse, error) {
	var response = &SendResponse{}
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(target, target.Credentials)
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

	if request.UDF != "" {
		if transformed, err := endly.TransformWithUDF(context, request.UDF, "mail.Body", mainMessage.Body); err == nil {
			mainMessage.Body = toolbox.AsString(transformed)
		}
	}
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
    "Credentials": "${env.HOME}/.secret/smtp.json"
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

func (s *service) registerRoutes() {
	s.Register(&endly.ServiceActionRoute{
		Action: "send",
		RequestInfo: &endly.ActionInfo{
			Description: "send an email",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "email send",
					Data:    sMTPSendExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SendRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SendRequest{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SendRequest); ok {
				return s.send(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//New creates a new NoOperation service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
