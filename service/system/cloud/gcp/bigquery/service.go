package bigquery

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
	"google.golang.org/api/bigquery/v2"
	"log"
)

const (
	//ServiceID Google BigQuery Service ID.
	ServiceID  = "gcp/bigquery"
	doneStatus = "DONE"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &bigquery.Service{}
	routes, err := gcp.BuildRoutes(client, nil, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "query",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "query", &QueryRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &QueryRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*QueryRequest); ok {
				output, err := s.query(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "query", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "load",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "load", &LoadRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &LoadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoadRequest); ok {
				output, err := s.load(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "load", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "copy",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "copy", &CopyRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &CopyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CopyRequest); ok {
				output, err := s.copy(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "copy", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "table",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "table", &TableRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &TableResponse{}),
		},
		RequestProvider: func() interface{} {
			return &TableRequest{}
		},
		ResponseProvider: func() interface{} {
			return &TableResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*TableRequest); ok {
				output, err := s.Table(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "table", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "patch",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "patch", &PatchRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Table{}),
		},
		RequestProvider: func() interface{} {
			return &PatchRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Table{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PatchRequest); ok {
				output, err := s.patch(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "table", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "jobWait",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "jobWait", &JobWaitRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Job{}),
		},
		RequestProvider: func() interface{} {
			return &JobWaitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &bigquery.Job{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*JobWaitRequest); ok {
				output, err := s.jobWait(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "list", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New creates a new BigQuery service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
