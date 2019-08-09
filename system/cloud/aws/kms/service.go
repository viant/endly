package kms

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"log"
)

const (
	//ServiceID aws KMS ID.
	ServiceID = "aws/kms"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &kms.KMS{}
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
		Action: "setupKey",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "call", &SetupKeyInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &kms.CreateAliasOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupKeyInput{}
		},
		ResponseProvider: func() interface{} {
			return &kms.AliasListEntry{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupKeyInput); ok {
				resp, err := s.setupKey(context, req)
				if err == nil {
					if context.IsLoggingEnabled() {
						context.Publish(aws.NewOutputEvent("setupKey", "proxy", resp))
					}
				}
				return resp, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) getAlias(context *endly.Context, aliasName string) (*kms.AliasListEntry, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var nextMarker *string
	for {
		output, err := client.ListAliases(&kms.ListAliasesInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, err
		}
		if len(output.Aliases) == 0 {
			break
		}
		for _, candidate := range output.Aliases {
			if *candidate.AliasName == aliasName {
				return candidate, nil
			}
		}
		nextMarker = output.NextMarker
		if nextMarker == nil {
			break
		}
	}
	return nil, nil
}

func (s *service) setupKey(context *endly.Context, input *SetupKeyInput) (interface{}, error) {
	alias, err := s.getAlias(context, *input.AliasName)
	if err != nil || alias != nil {
		return alias, err
	}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	createKeyOutput, err := client.CreateKey(&input.CreateKeyInput)
	if err != nil {
		return nil, err
	}
	input.TargetKeyId = createKeyOutput.KeyMetadata.KeyId
	_, err = client.CreateAlias(&input.CreateAliasInput)
	if err != nil {
		return nil, err
	}
	return s.getAlias(context, *input.AliasName)
}

//New creates a new AWS Key Management  service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
