package s3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/viant/endly/system/cloud/aws/lambda"

	"log"
)

const (
	//ServiceID aws s3 service id.
	ServiceID = "aws/s3"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) setupBucketNotification(context *endly.Context, request *SetupBucketNotificationInput) (*SetupBucketNotificationOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}

	currentConfig, err := client.GetBucketNotificationConfiguration(&s3.GetBucketNotificationConfigurationRequest{
		Bucket: request.Bucket,
	})

	if err != nil {
		currentConfig = &s3.NotificationConfiguration{
			LambdaFunctionConfigurations: make([]*s3.LambdaFunctionConfiguration, 0),
		}
	}
	response, err := s.updateBucketNotification(context, currentConfig, request)
	if err != nil {
		return nil, err
	}
	currentConfig, err = client.GetBucketNotificationConfiguration(&s3.GetBucketNotificationConfigurationRequest{
		Bucket: request.Bucket,
	})
	response.NotificationConfiguration = currentConfig
	return response, err
}

func (s *service) updateBucketNotification(ctx *endly.Context, currentConfig *s3.NotificationConfiguration, request *SetupBucketNotificationInput) (*SetupBucketNotificationOutput, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, err
	}
	input := &s3.PutBucketNotificationConfigurationInput{
		Bucket: request.Bucket,
		NotificationConfiguration: &s3.NotificationConfiguration{},
	}

	response := &SetupBucketNotificationOutput{
		Bucket: request.Bucket,
		NotificationConfiguration: input.NotificationConfiguration,
		LambdaPermissions:         make([]*lambda.SetupPermissionInput, 0),
	}

	if len(request.LambdaFunctionConfigurations) > 0 {
		if input.NotificationConfiguration.LambdaFunctionConfigurations, err = s.setupLambdaNotification(ctx, currentConfig, request, response); err != nil {
			return nil, err
		}
	}
	if len(request.QueueConfigurations) > 0 {
		if input.NotificationConfiguration.QueueConfigurations, err = s.setupQueueNotification(ctx, currentConfig, request, response); err != nil {
			return nil, err
		}
	}
	if len(request.TopicConfigurations) > 0 {
		if input.NotificationConfiguration.TopicConfigurations, err = s.setupTopicNotification(ctx, currentConfig, request, response); err != nil {
			return nil, err
		}
	}

	_, err = client.PutBucketNotificationConfiguration(input)
	if err != nil {
		return nil, errors.Wrapf(err, "unable put bucket notification: %v", input)
	}

	return response, nil
}

func (s *service) setupLambdaNotification(ctx *endly.Context, currentConfig *s3.NotificationConfiguration, request *SetupBucketNotificationInput, response *SetupBucketNotificationOutput) ([]*s3.LambdaFunctionConfiguration, error) {
	var result = make([]*s3.LambdaFunctionConfiguration, 0)
	configuredLambdaFunctions := currentConfig.LambdaFunctionConfigurations
	existingFunction := indexLambdaFunction(configuredLambdaFunctions)
	var state = ctx.State()
	state = state.Clone()

	for _, config := range request.NotificationConfiguration.LambdaFunctionConfigurations {
		funcName := *config.FunctionName
		lambdaConfig, ok := existingFunction[funcName]
		if ok {
			lambdaConfig.Events = config.Events
			lambdaConfig.Filter = config.Filter.ToNotificationConfigurationFilter()
		} else {
			function, err := aws.GetFunctionConfiguration(ctx, funcName)
			if err != nil {
				return nil, err
			}
			aws.SetFunctionInfo("function", function, state)
			lambdaConfig = &config.LambdaFunctionConfiguration
			lambdaConfig.LambdaFunctionArn = function.FunctionArn
		}
		lambdaConfig.Filter = config.Filter.ToNotificationConfigurationFilter()
		permissionInput := &config.SetupPermissionInput
		*permissionInput.StatementId = state.ExpandAsText(*permissionInput.StatementId)
		*permissionInput.SourceArn = state.ExpandAsText(*permissionInput.SourceArn)

		if config.Id != nil {
			*config.Id = state.ExpandAsText(*config.Id)
		}

		result = append(result, lambdaConfig)
		if err := endly.Run(ctx, permissionInput, nil); err != nil {
			return nil, err
		}
		response.LambdaPermissions = append(response.LambdaPermissions, &config.SetupPermissionInput)
	}
	return result, nil
}

func (s *service) setupQueueNotification(context *endly.Context, configuration *s3.NotificationConfiguration, input *SetupBucketNotificationInput, output *SetupBucketNotificationOutput) ([]*s3.QueueConfiguration, error) {
	var err error
	state := context.State()
	var result = make([]*s3.QueueConfiguration, 0)

	//getBucketPolicy

	for i := range input.QueueConfigurations {
		config := input.QueueConfigurations[i]
		if config.QueueArn == nil {
			config.Queue = context.Expand(config.Queue)
			if config.QueueArn, err = aws.GetQueueARN(context, config.Queue); err != nil {
				return nil, err
			}
			config.QueueConfiguration.Filter = config.Filter.ToNotificationConfigurationFilter()
			if config.Id != nil {
				*config.Id = state.ExpandAsText(*config.Id)
			}
			config.SetupPermissionInput.Queue = config.Queue
			config.SetupPermissionInput.Everybody = true
			config.SetupPermissionInput.SourceArn = fmt.Sprintf("arn:aws:s3:::%s", *input.Bucket)
			if err := endly.Run(context, &config.SetupPermissionInput, nil); err != nil {
				return nil, err
			}
		}
		result = append(result, &config.QueueConfiguration)
	}
	return result, nil
}

func (s *service) setupTopicNotification(context *endly.Context, configuration *s3.NotificationConfiguration,
	input *SetupBucketNotificationInput, output *SetupBucketNotificationOutput) ([]*s3.TopicConfiguration, error) {
	var err error
	var result = make([]*s3.TopicConfiguration, 0)
	for i := range input.TopicConfigurations {
		config := input.TopicConfigurations[i]
		if config.TopicArn == nil {
			config.Topic = context.Expand(config.Topic)
			if config.TopicArn, err = aws.GetTopicARN(context, config.Topic); err != nil {
				return nil, err
			}
			config.TopicConfiguration.Filter = config.Filter.ToNotificationConfigurationFilter()
			if config.Id != nil {
				*config.Id = context.Expand(*config.Id)
			}

			config.SetupPermissionInput.Topic = config.Topic
			config.SetupPermissionInput.Everybody = true
			config.SetupPermissionInput.SourceArn = fmt.Sprintf("arn:aws:s3:::%s", *input.Bucket)
			if err := endly.Run(context, &config.SetupPermissionInput, nil); err != nil {
				return nil, err
			}
		}
		result = append(result, &config.TopicConfiguration)
	}
	return result, nil
}

func (s *service) registerRoutes() {
	client := &s3.S3{}
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
		Action: "setupBucketNotification",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupBucketNotification", &SetupBucketNotificationInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &SetupBucketNotificationOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupBucketNotificationInput{}
		},
		ResponseProvider: func() interface{} {
			return &SetupBucketNotificationOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupBucketNotificationInput); ok {
				response, err := s.setupBucketNotification(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(aws.NewOutputEvent("notification", "s3", response))
				return response, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new AWS S3 service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
