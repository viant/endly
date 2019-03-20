package logs

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"log"
	"strings"
)

const (
	//ServiceID aws Cloudwatch service id.
	ServiceID = "aws/logs"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) fetchLogEvents(context *endly.Context, request *FilterLogEventMessagesInput) (response *FilterLogEventMessagesOutput, err error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output, err := client.FilterLogEvents(&request.FilterLogEventsInput)
	if err != nil {
		return nil, err
	}
	response = &FilterLogEventMessagesOutput{
		Messages:           make([]interface{}, 0),
		SearchedLogStreams: output.SearchedLogStreams,
	}
	defer func() {
		normalizeMessages(response.Messages)
		if context.IsLoggingEnabled() {
			context.Publish(aws.NewOutputEvent("filterLogEventMessages", "proxy", response))
		}
	}()

	if len(request.Exclude)+len(request.Include) == 0 {
		for _, event := range output.Events {
			response.Messages = append(response.Messages, event.Message)
		}
		return response, nil
	}
	exclusion := len(request.Exclude) > 0
outer:
	for _, candidate := range output.Events {
		if exclusion {
			for _, fragment := range request.Exclude {
				if strings.Contains(*candidate.Message, fragment) {
					continue outer
				}
			}
		}
		if len(request.Include) > 0 {
			for _, fragment := range request.Include {
				if strings.Contains(*candidate.Message, fragment) {
					response.Messages = append(response.Messages, candidate.Message)
					continue outer
				}
			}
			continue outer
		}
		response.Messages = append(response.Messages, candidate.Message)
	}
	return response, nil
}

func (s *service) setupLogGroup(context *endly.Context, input *SetupLogGroupInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	request := cloudwatchlogs.CreateLogGroupInput(*input)

	logGroup, err := client.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: request.LogGroupName,
	})
	if err != nil {
		return client.CreateLogGroup(&request)
	}
	return logGroup, nil
}

func (s *service) registerRoutes() {
	client := &cloudwatchlogs.CloudWatchLogs{}
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
		Action: "filterLogEventMessages",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "fetchLogEvents", &FilterLogEventMessagesInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &FilterLogEventMessagesOutput{}),
		},
		RequestProvider: func() interface{} {
			return &FilterLogEventMessagesInput{}
		},
		ResponseProvider: func() interface{} {
			return &FilterLogEventMessagesOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*FilterLogEventMessagesInput); ok {
				return s.fetchLogEvents(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "setupLogGroup",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupLogGroup", &SetupLogGroupInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &cloudwatchlogs.CreateLogGroupOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupLogGroupInput{}
		},
		ResponseProvider: func() interface{} {
			return &cloudwatchlogs.CreateLogGroupOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupLogGroupInput); ok {
				return s.setupLogGroup(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new AWS Cloudwatch service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
