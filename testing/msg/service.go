package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox"
)

const (
	//ServiceID represents gloud msg  pubsub service id.
	ServiceID = "msg"
)

//service represent SMTP service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "push",
		RequestInfo: &endly.ActionInfo{
			Description: "push message",
		},
		RequestProvider: func() interface{} {
			return &PushRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PushResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PushRequest); ok {
				return s.push(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "pull",
		RequestInfo: &endly.ActionInfo{
			Description: "pull message",
		},
		RequestProvider: func() interface{} {
			return &PullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PullRequest); ok {
				return s.pull(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "setupResource",
		RequestInfo: &endly.ActionInfo{
			Description: "create pubsub resources",
		},
		RequestProvider: func() interface{} {
			return &CreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CreateResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CreateRequest); ok {
				return s.create(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "deleteResource",
		RequestInfo: &endly.ActionInfo{
			Description: "delete pubsub resources",
		},
		RequestProvider: func() interface{} {
			return &DeleteRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeleteResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeleteRequest); ok {
				return s.delete(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) push(context *endly.Context, request *PushRequest) (interface{}, error) {
	response := PushResponse{
		Results: make([]Result, 0),
	}

	if request.Source != nil {
		download := &storage.DownloadResponse{}
		err := endly.Run(context, &storage.DownloadRequest{Source: request.Source}, download)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download push source: %v", request.Source)
		}
		request.Messages = loadMessages([]byte(download.Payload))
	}

	var duration, _ = toolbox.NewDuration(request.TimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, request.Dest, duration)
	if err != nil {
		return response, err
	}
	defer client.Close()

	dest := expandResource(context, request.Dest)
	var state = context.State()
	for _, message := range request.Messages {
		expanded := message.Expand(state)
		if request.UDF != "" {
			expanded.Data, err = udf.TransformWithUDF(context, request.UDF, fmt.Sprintf("%v/%v", request.Dest.Type, request.Dest.Name), expanded.Data)
			if err != nil {
				return nil, err
			}
		}
		result, err := client.Push(context.Background(), dest, expanded)
		if err != nil {
			return response, err
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

func (s *service) pull(context *endly.Context, request *PullRequest) (interface{}, error) {
	response := PullResponse{}
	var duration, _ = toolbox.NewDuration(request.TimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, request.Source, duration)
	if err != nil {
		return response, err
	}
	source := expandResource(context, request.Source)
	defer client.Close()
	response.Messages, err = client.PullN(context.Background(), source, request.Count, request.Nack)
	if err == nil {
		for _, message := range response.Messages {
			if request.UDF != "" {
				message.Transformed, err = udf.TransformWithUDF(context, request.UDF, fmt.Sprintf("%v/%v", request.Source.Type, request.Source.Name), message.Data)
				if err != nil {
					return nil, err
				}
			}
		}
		if request.Expect != nil {
			response.Assert, err = validator.Assert(context, request, request.Expect, response.Messages, "msg.response", "assert msg response")
		}
	}

	return response, err
}

func (s *service) setupResource(context *endly.Context, resource *ResourceSetup) (*Resource, error) {
	var duration, _ = toolbox.NewDuration(defaultTimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, &resource.Resource, duration)
	if err != nil {
		return nil, err
	}
	var state = context.State()
	resource.URL = state.ExpandAsText(resource.URL)
	resource.projectID = state.ExpandAsText(resource.projectID)
	if resource.Config != nil {
		if resource.Config.Topic != nil {
			resource.Config.Topic.URL = state.ExpandAsText(resource.Config.Topic.URL)
			resource.Config.Topic.projectID = state.ExpandAsText(resource.Config.Topic.projectID)
		}
	}
	defer client.Close()
	return client.SetupResource(resource)
}

func (s *service) create(context *endly.Context, request *CreateRequest) (interface{}, error) {
	var response = CreateResponse{
		Resources: make([]*Resource, 0),
	}
	for _, resource := range request.Resources {
		responseResource, err := s.setupResource(context, resource)
		if err != nil {
			return nil, err
		}
		response.Resources = append(response.Resources, responseResource)
	}
	return response, nil
}

func (s *service) delete(context *endly.Context, request *DeleteRequest) (interface{}, error) {
	response := &DeleteRequest{}
	for _, resource := range request.Resources {
		if err := s.deleteResource(context, resource); err != nil {
			return response, err
		}
	}
	return response, nil
}
func (s *service) deleteResource(context *endly.Context, resource *Resource) error {
	var duration, _ = toolbox.NewDuration(defaultTimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, resource, duration)
	if err != nil {
		return err
	}
	defer client.Close()
	var state = context.State()
	resource.URL = state.ExpandAsText(resource.URL)
	return client.DeleteResource(resource)
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
