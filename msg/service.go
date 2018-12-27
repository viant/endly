package msg

import (
	"fmt"
	"github.com/viant/endly"
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

//TODO UDFs support and example

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
		Action: "create",
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
		Action: "delete",
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
	var duration, _ = toolbox.NewDuration(request.TimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, request.Dest, duration)
	if err != nil {
		return response, err
	}
	defer client.Close()
	dest := expandResource(context, request.Dest)
	var state = context.State()
	for _, message := range request.Messages {
		result, err := client.Push(dest, message.Expand(state))
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
	response.Messages, err = client.PullN(source, request.Count)
	return response, err
}

func (s *service) createResource(context *endly.Context, resource *ResourceSetup) (*Resource, error) {
	var duration, _ = toolbox.NewDuration(defaultTimeoutMs, toolbox.DurationMillisecond)
	client, err := NewPubSubClient(context, &resource.Resource, duration)
	if err != nil {
		return nil, err
	}
	var state = context.State()
	resource.URL = state.ExpandAsText(resource.URL)
	defer client.Close()
	return client.Create(resource)
}

func (s *service) create(context *endly.Context, request *CreateRequest) (interface{}, error) {
	var response = CreateResponse{
		Resources: make([]*Resource, 0),
	}
	for _, resource := range request.Resources {
		responseResource, err := s.createResource(context, resource)
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
	return client.Delete(resource)
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
