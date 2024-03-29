package smtp

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-smtp"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/toolbox/data"
	"log"
	"path"
)

const ServiceID = "smtp/endpoint"

type service struct {
	*endly.AbstractService
	messages *Messages
}

func (s *service) listen(context *endly.Context, request *ListenRequest) (*ListenResponse, error) {
	var response = &ListenResponse{}
	be := newBackend(s.messages, request.Users)
	server := smtp.NewServer(be)
	err := s.initServer(server, request)
	if err != nil {
		return nil, err
	}
	s.messages.debug = request.Debug
	go startServer(server, request)
	return response, err
}

func startServer(server *smtp.Server, request *ListenRequest) {
	if request.EnableTLS {
		if err := server.ListenAndServeTLS(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (s *service) initServer(server *smtp.Server, request *ListenRequest) error {
	server.Addr = fmt.Sprintf(":%v", request.Port)
	server.Domain = request.ServerName
	server.MaxMessageBytes = request.MaxBodySize
	server.AllowInsecureAuth = true
	if request.EnableTLS {
		server.TLSConfig = &tls.Config{MinVersion: tls.VersionSSL30}
		location := location.NewResource(request.CertLocation).Path()
		certFile := path.Join(location, "cert.pem")
		keyFile := path.Join(location, "key.pem")
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load certifiacte: %v, %v, %v", err, certFile, keyFile)
		}
		server.TLSConfig.Certificates = []tls.Certificate{
			certificate,
		}
	}
	return nil
}

func (s *service) assert(context *endly.Context, request *AssertRequest) (*AssertResponse, error) {
	var response = &AssertResponse{
		Validations: make([]*assertly.Validation, 0),
	}
	if len(request.Expect) == 0 {
		return response, nil
	}

	var messageCount = map[string]int{}
	for _, userMessage := range request.Expect {
		var aMap = data.NewMap()
		aMap.Put("user", userMessage.User)
		aMap.Put("TagID", userMessage.TagID)

		var validation = &assertly.Validation{
			TagID:       userMessage.TagID,
			Description: aMap.ExpandAsText(request.DescriptionTemplate),
		}
		response.Validations = append(response.Validations, validation)
		messageCount[userMessage.User]++
		actualMessage := s.messages.Shift(userMessage.User)
		if actualMessage == nil {
			validation.AddFailure(assertly.NewFailure("", fmt.Sprintf("[%v]", userMessage.TagID), fmt.Sprintf("missing mail,  user %v ", userMessage.User), userMessage.Message, nil))
			break
		}
		taggedAssert := &validator.TaggedAssert{
			TagID:    userMessage.TagID,
			Expected: userMessage.Message,
			Actual:   actualMessage,
		}
		messageValidation, err := criteria.Assert(context, fmt.Sprintf("mail(%v[%v])", userMessage.User, messageCount[userMessage.User]-1), taggedAssert.Expected, taggedAssert.Actual)
		if err != nil {
			return nil, err
		}
		context.Publish(taggedAssert)
		context.Publish(messageValidation)
		validation.MergeFrom(messageValidation)
	}
	return response, nil
}

func (s *service) registerRoutes() {
	//listen action route
	s.Register(&endly.Route{
		Action: "listen",
		RequestInfo: &endly.ActionInfo{
			Description: "Listen A SMTP service.",
		},
		RequestProvider: func() interface{} {
			return &ListenRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListenResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListenRequest); ok {
				return s.listen(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	//assert action route
	s.Register(&endly.Route{
		Action: "assert",
		RequestInfo: &endly.ActionInfo{
			Description: "assert received messages",
		},
		RequestProvider: func() interface{} {
			return &AssertRequest{}
		},
		ResponseProvider: func() interface{} {
			return &AssertResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*AssertRequest); ok {
				return s.assert(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New create a new SMTP endpoint service
func New() endly.Service {
	var result = &service{
		messages:        NewMessages(),
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
