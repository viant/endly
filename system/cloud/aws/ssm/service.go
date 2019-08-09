package ssm

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"log"
)

const (
	//ServiceID aws Simple Systems Manager ID.
	ServiceID = "aws/ssm"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &ssm.SSM{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "setParameter",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setParameter", &SetParameterInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &ssm.PutParameterOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetParameterInput{}
		},
		ResponseProvider: func() interface{} {
			return &ssm.PutParameterOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetParameterInput); ok {
				resp, err := s.setParameter(context, req)
				if err == nil {
					if context.IsLoggingEnabled() {
						context.Publish(aws.NewOutputEvent("setParameter", "proxy", resp))
					}
				}
				return resp, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) setParameter(context *endly.Context, input *SetParameterInput) (*ssm.PutParameterOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	withDecryption := true
	getOutput, err := client.GetParameter(&ssm.GetParameterInput{
		Name:           input.Name,
		WithDecryption: &withDecryption,
	})

	found := err == nil && getOutput != nil

	if found && *getOutput.Parameter.Value == *input.Value {
		return &ssm.PutParameterOutput{
			Version: getOutput.Parameter.Version,
		}, nil
	}
	putParameter := ssm.PutParameterInput(*input)
	return client.PutParameter(&putParameter)
}

//New creates a new AWS Simple Systems Manager service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
