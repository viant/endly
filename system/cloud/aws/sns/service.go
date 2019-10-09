package sns

import (
	"encoding/json"
	"fmt"
	aaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/viant/endly/system/cloud/aws/iam"
	"github.com/viant/endly/system/cloud/aws/lambda"
	"github.com/viant/toolbox"
	"log"
	"strings"
)

const (
	//ServiceID aws Simple Notification Service ID.
	ServiceID = "aws/sns"

	policyAttribute = "Policy"
	snsPrincipal    = "sns.amazonaws.com"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) matchTopic(client *sns.SNS, name string) (*sns.Topic, error) {
	var nextToken *string
	for {
		list, err := client.ListTopics(&sns.ListTopicsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, candidate := range list.Topics {
			ARN, _ := arn.Parse(*candidate.TopicArn)
			topicName := strings.Replace(ARN.Resource, "topic:", "", 1)
			if topicName == name {
				return candidate, nil
			}
		}
		nextToken = list.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil, nil
}

func (s *service) setupTopic(context *endly.Context, request *SetupTopicInput) (*sns.Topic, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	topic, err := s.matchTopic(client, *request.Name)
	if err != nil || topic != nil {
		return topic, err
	}
	createTopicInput := sns.CreateTopicInput(*request)
	output, err := client.CreateTopic(&createTopicInput)
	if err != nil {
		return nil, err
	}
	return &sns.Topic{
		TopicArn: output.TopicArn,
	}, nil
}

func (s *service) matchSubscription(client *sns.SNS, request *SetupSubscriptionInput) (*sns.Subscription, error) {
	var nextToken *string
	for {
		list, err := client.ListSubscriptions(&sns.ListSubscriptionsInput{NextToken: nextToken})
		if err != nil {
			return nil, err
		}
		for _, candidate := range list.Subscriptions {
			if *candidate.TopicArn != *request.TopicArn {
				continue
			}
			if *candidate.Protocol != *request.Protocol {
				continue
			}
			return candidate, nil
		}
		nextToken = list.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil, nil
}

func (s *service) updateSubscriptionEndpointIfNeeded(context *endly.Context, request *SetupSubscriptionInput) error {
	switch *request.Protocol {
	case "lambda", "sqs":
	default:
		return nil
	}
	_, err := arn.Parse(*request.Endpoint)
	if err == nil {
		return nil
	}
	switch *request.Protocol {
	case "lambda":
		config, err := aws.GetFunctionConfiguration(context, *request.Endpoint)
		if err != nil {
			return err
		}
		request.Endpoint = config.FunctionArn
	case "sqs":
		ARN, err := aws.GetQueueARN(context, *request.Endpoint)
		if err != nil {
			return err
		}
		request.Endpoint = ARN
	}
	return nil
}

func (s *service) setupSubscription(context *endly.Context, request *SetupSubscriptionInput) (*sns.Subscription, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	if request.TopicArn == nil {
		topic, err := s.setupTopic(context, &SetupTopicInput{
			Name: request.Topic,
		})
		if err != nil {
			return nil, err
		}
		request.TopicArn = topic.TopicArn
	}

	if err = s.updateSubscriptionEndpointIfNeeded(context, request); err != nil {
		return nil, err
	}
	subscription, err := s.matchSubscription(client, request)
	if err != nil {
		return subscription, err
	}
	if subscription == nil {
		output, err := client.Subscribe(request.SubscribeInput)
		if err != nil {
			return nil, err
		}
		subscription = &sns.Subscription{
			SubscriptionArn: output.SubscriptionArn,
			Endpoint:        request.Endpoint,
			Protocol:        request.Protocol,
			TopicArn:        request.TopicArn,
		}
	}
	state := context.State()
	if *subscription.Protocol == "lambda" {
		permissionInput := &lambda.SetupPermissionInput{}
		functionARN, _ := arn.Parse(*request.Endpoint)
		functionName := strings.Replace(functionARN.Resource, "function:", "", 1)
		permissionInput.FunctionName = &functionName
		permissionInput.SourceArn = subscription.TopicArn
		permissionInput.Action = &aws.LambdaInvoke
		permissionInput.Principal = aaws.String(snsPrincipal)
		statementID := state.ExpandAsText(fmt.Sprintf("%v_%v_${uuid.next}", request.Topic, functionName))
		permissionInput.StatementId = &statementID
		if err := endly.Run(context, permissionInput, nil); err != nil {
			return nil, err
		}
	}
	return subscription, nil
}

func (s *service) registerRoutes() {
	client := &sns.SNS{}
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
		Action: "setupTopic",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupTopic", &SetupTopicInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &sns.Topic{}),
		},
		RequestProvider: func() interface{} {
			return &SetupTopicInput{}
		},
		ResponseProvider: func() interface{} {
			return &sns.Topic{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupTopicInput); ok {
				return s.setupTopic(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "setupSubscription",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupSubscription", &SetupSubscriptionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &sns.Subscription{}),
		},
		RequestProvider: func() interface{} {
			return &SetupSubscriptionInput{}
		},
		ResponseProvider: func() interface{} {
			return &sns.Subscription{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupSubscriptionInput); ok {
				output, err := s.setupSubscription(context, req)
				if err != nil {
					return nil, err
				}
				ARN, _ := arn.Parse(*output.SubscriptionArn)
				topic := string(ARN.Resource[:strings.Index(ARN.Resource, ":")])
				context.Publish(aws.NewOutputEvent(fmt.Sprintf("%v", topic), "subscription", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "setupPermission",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupPermission", &SetupPermissionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &sns.AddPermissionOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupPermissionInput{}
		},
		ResponseProvider: func() interface{} {
			return &sns.AddPermissionOutput{}
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

func (s *service) setupPermission(context *endly.Context, request *SetupPermissionInput) (*sns.AddPermissionOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	if request.TopicArn == nil {
		if request.TopicArn, err = aws.GetTopicARN(context, request.Topic); err != nil {
			return nil, err
		}
	}

	*request.Label = context.Expand(*request.Label)

	_, _ = client.RemovePermission(&sns.RemovePermissionInput{
		TopicArn: request.TopicArn,
		Label:    request.Label,
	})

	if len(request.AWSAccountId) > 0 {
		for i, acount := range request.AWSAccountId {
			if acount != nil {
				*request.AWSAccountId[i] = context.Expand(*acount)
			}
		}
	}

	output, err := client.AddPermission(&request.AddPermissionInput)
	if err != nil {
		err = errors.Wrapf(err, "failed to add permission: %v", request.AddPermissionInput)
		return nil, err
	}
	if request.Everybody || request.SourceArn != "" {
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
	topicAttributes, err := client.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: request.TopicArn,
	})
	if err == nil {
		policyPayload, ok := topicAttributes.Attributes[policyAttribute]
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
			_, err = client.SetTopicAttributes(&sns.SetTopicAttributesInput{
				TopicArn:       request.TopicArn,
				AttributeName:  aaws.String(policyAttribute),
				AttributeValue: aaws.String(string(updated)),
			})
		}
	}
	return err
}

//New creates a new AWS SNS service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
