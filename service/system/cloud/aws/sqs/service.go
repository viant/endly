package sqs

import (
	"encoding/json"
	"fmt"
	aaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
	"github.com/viant/endly/service/system/cloud/aws/iam"
	"github.com/viant/toolbox"
	"log"
)

const (
	//ServiceID aws Simple Queue Service ID.
	ServiceID = "aws/sqs"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &sqs.SQS{}
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
		Action: "setupPermission",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupPermission", &SetupPermissionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &sqs.AddPermissionOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupPermissionInput{}
		},
		ResponseProvider: func() interface{} {
			return &sqs.AddPermissionOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupPermissionInput); ok {
				return s.setupPermission(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) setupPermission(context *endly.Context, request *SetupPermissionInput) (*sqs.AddPermissionOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	if request.QueueUrl == nil {
		queueURLOutput, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: &request.Queue,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get queue URL for %v", request.Queue)
		}
		request.QueueUrl = queueURLOutput.QueueUrl
	}

	*request.Label = context.Expand(*request.Label)
	_, _ = client.RemovePermission(&sqs.RemovePermissionInput{
		QueueUrl: request.QueueUrl,
		Label:    request.Label,
	})

	if len(request.AWSAccountIds) > 0 {
		for i, action := range request.AWSAccountIds {
			if action != nil {
				*request.AWSAccountIds[i] = context.Expand(*action)
			}
		}
	}
	output, err := client.AddPermission(&request.AddPermissionInput)
	if err != nil {
		err = errors.Wrapf(err, "failed to add permission: %v", request.AddPermissionInput)
	}
	if request.Everybody {
		if err = s.adjustPermissionPrincipal(context, request); err != nil {
			return nil, err
		}
	}
	return output, err
}

func (s *service) adjustPermissionPrincipal(context *endly.Context, request *SetupPermissionInput) error {
	client, err := GetClient(context)
	if err != nil {
		return err
	}
	queueAttributes, err := client.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl: request.QueueUrl,
		AttributeNames: []*string{
			aaws.String(string(sqs.QueueAttributeNamePolicy)),
		},
	})
	if err == nil {
		policyPayload, ok := queueAttributes.Attributes[sqs.QueueAttributeNamePolicy]
		if ok {
			policy := &iam.PolicyDocument{}
			if err = json.Unmarshal([]byte(*policyPayload), &policy); err != nil {
				return errors.Wrapf(err, "failed to decode policy: %s", *policyPayload)
			}
			for i := range policy.Statement {
				if *policy.Statement[i].Sid == *request.Label {
					if request.Everybody {
						policy.Statement[i].Principal = toolbox.AnyJSONType(`{"AWS":"*"}`)
					}
					if request.SourceArn != "" {
						policy.Statement[i].Condition = toolbox.AnyJSONType(fmt.Sprintf(`{"StringEquals": {"aws:SourceArn": "%s"}}`, request.SourceArn))
					}
				}
			}
			updated, err := json.Marshal(policy)
			if err != nil {
				return errors.Wrapf(err, "failed to decode policy: %s", *policyPayload)
			}

			_, err = client.SetQueueAttributes(&sqs.SetQueueAttributesInput{
				QueueUrl: request.QueueUrl,
				Attributes: map[string]*string{
					sqs.QueueAttributeNamePolicy: aaws.String(string(updated)),
				},
			})
		}
	}
	return err
}

// New creates a new AWS SQS service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
