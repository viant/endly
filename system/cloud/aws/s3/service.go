package s3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
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
	return s.updateBucketNotification(context, currentConfig, request)
}


func (s *service) updateBucketNotification(ctx *endly.Context, currentConfig *s3.NotificationConfiguration, request *SetupBucketNotificationInput)  (*SetupBucketNotificationOutput, error)  {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, err
	}


	input := &s3.PutBucketNotificationConfigurationInput{
		Bucket: request.Bucket,
		NotificationConfiguration:&s3.NotificationConfiguration{
			LambdaFunctionConfigurations: make([]*s3.LambdaFunctionConfiguration, 0),
			QueueConfigurations:          request.NotificationConfiguration.QueueConfigurations,
			TopicConfigurations:          request.NotificationConfiguration.TopicConfigurations,
		},
	}
	response := &SetupBucketNotificationOutput{
		Bucket:                    request.Bucket,
		NotificationConfiguration: input.NotificationConfiguration,
		Permissions:               make([]*lambda.SetupPermissionInput, 0),
	}
	configuredLambdaFunctions := currentConfig.LambdaFunctionConfigurations
	existingFunction := indexLambdaFunction(configuredLambdaFunctions)
	var state = ctx.State()
	state = state.Clone()

	for _, configuration := range request.NotificationConfiguration.LambdaFunctionConfigurations {
		funcName:= *configuration.FunctionName
		lambdaConfig, ok := existingFunction[funcName]
		if ok {
			lambdaConfig.Events = configuration.Events
			lambdaConfig.Filter = configuration.Filter.ToNotificationConfigurationFilter()
		} else {
			function, err := lambda.GetFunctionConfiguration(ctx, funcName)
			if err != nil {
				return nil, err
			}
			lambda.SetFunctionInfo(function, state)
			lambdaConfig = &configuration.LambdaFunctionConfiguration
			lambdaConfig.LambdaFunctionArn = function.FunctionArn
		}
		lambdaConfig.Filter = configuration.Filter.ToNotificationConfigurationFilter()
		permissionInput := &configuration.SetupPermissionInput
		*permissionInput.StatementId = state.ExpandAsText(*permissionInput.StatementId)
		*permissionInput.SourceArn = state.ExpandAsText(*permissionInput.SourceArn)
		*configuration.Id = state.ExpandAsText(*configuration.Id)
		input.NotificationConfiguration.LambdaFunctionConfigurations = append(input.NotificationConfiguration.LambdaFunctionConfigurations, lambdaConfig)
		if err  := endly.Run(ctx, permissionInput, nil);err != nil {
			return nil, err
		}
		response.Permissions = append(response.Permissions, &configuration.SetupPermissionInput)
	}
	if _, err = client.PutBucketNotificationConfiguration(input);err != nil {
		return nil, fmt.Errorf("unable put bucket notification: %v", err)
	}
	return response, nil
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
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupFunction", &SetupBucketNotificationInput{}),
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
				return s.setupBucketNotification(context, req)
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
