package cloudscheduler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/cloudscheduler/v1beta1"
	"log"
)

const (
	//ServiceID Google cloudscheduler Service ID.
	ServiceID = "gcp/cloudscheduler"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) getJob(context *endly.Context, parent, name string) (*cloudscheduler.Job, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	service := cloudscheduler.NewProjectsLocationsJobsService(client.service)
	id := parent + "/jobs/" + name
	call := service.Get(id)
	call.Context(client.Context())
	return call.Do()
}

func (s *service) deploy(context *endly.Context, request *DeployRequest) (*DeployResponse, error) {
	var err error
	response := &DeployResponse{}
	parent := request.Parent(context)
	job, _ := s.getJob(context, parent, request.jobName)

	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	service := cloudscheduler.NewProjectsLocationsJobsService(client.service)
	id := parent + "/jobs/" + request.jobName
	request.Name = id

	if request.HttpTarget != nil && request.Body != "" {
		body := context.Expand(request.Body)
		request.HttpTarget.Body = base64.StdEncoding.EncodeToString([]byte(body))
	}
	if job == nil {
		call := service.Create(parent, &request.Job)
		call.Context(client.Context())
		response.Job, err = call.Do()
	} else {
		call := service.Patch(id, &request.Job)
		call.Context(client.Context())
		response.Job, err = call.Do()
	}
	if err != nil {
		JSON, _ := json.Marshal(request.Job)
		err = errors.Wrapf(err, "failed to deploy: %v, %s", id, JSON)
	}
	return response, err
}

func (s *service) registerRoutes() {
	client := &cloudscheduler.Service{}
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
		Action: "deploy",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deploy", &DeployRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeployResponse{}),
		},
		RequestProvider: func() interface{} {
			return &DeployRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeployResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeployRequest); ok {
				output, err := s.deploy(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "deploy", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New creates a new cloudscheduler service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
