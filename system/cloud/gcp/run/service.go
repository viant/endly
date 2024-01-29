package run

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/toolbox"
	"google.golang.org/api/run/v1"
	"log"
	"time"
)

const (
	//ServiceID Google run Service ID.
	ServiceID      = "gcp/run"
	waitTimeoutMs  = 30000
	runInvokerRole = "roles/run.invoker"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &run.APIService{}
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
	s.Register(&endly.Route{
		Action: "getService",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getService", &GetServiceRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetServiceResponse{}),
		},
		RequestProvider: func() interface{} {
			return &GetServiceRequest{}
		},
		ResponseProvider: func() interface{} {
			return &GetServiceResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetServiceRequest); ok {
				output, err := s.getService(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "getService", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getConfiguration",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getConfiguration", &GetConfigurationRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetServiceResponse{}),
		},
		RequestProvider: func() interface{} {
			return &GetConfigurationRequest{}
		},
		ResponseProvider: func() interface{} {
			return &GetServiceResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetConfigurationRequest); ok {
				output, err := s.getConfiguration(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "getService", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) getService(context *endly.Context, request *GetServiceRequest) (*GetServiceResponse, error) {
	response := &GetServiceResponse{}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	service := run.NewNamespacesServicesService(client.service)
	URI := gcp.ExpandMeta(context, request.uri)
	getCall := service.Get(URI)
	getCall.Context(client.Context())
	response.Service, err = getCall.Do()
	err = toolbox.ReclassifyNotFoundIfMatched(err, URI)
	if err != nil && !toolbox.IsNotFoundError(err) {
		return nil, err
	}
	return response, nil
}

func (s *service) getConfiguration(context *endly.Context, request *GetConfigurationRequest) (*GetConfigurationResponse, error) {
	client, err := GetClient(context)
	if err == nil {
		err = request.Init()
	}
	if err != nil {
		return nil, err
	}
	service := run.NewNamespacesConfigurationsService(client.service)
	URI := gcp.ExpandMeta(context, request.uri)
	getCall := service.Get(URI)
	getCall.Context(client.Context())
	response, err := getCall.Do()
	if err != nil {
		return nil, err
	}
	return &GetConfigurationResponse{response}, nil
}

func (s *service) deploy(context *endly.Context, request *DeployRequest) (*DeployResponse, error) {
	var err error
	response := &DeployResponse{}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	service := run.NewNamespacesServicesService(client.service)
	getRequest := &GetServiceRequest{Name: request.Name}
	getResponse := &GetServiceResponse{}
	err = endly.Run(context, getRequest, getResponse)
	if err != nil {
		return nil, err
	}

	srv, err := request.Service(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract service from request")
	}

	if getResponse.Service == nil || getResponse.Service.Spec == nil {
		parent := gcp.ExpandMeta(context, request.parent)
		createCall := service.Create(parent, srv)
		createCall.Context(client.Context())
		if getResponse.Service, err = createCall.Do(); err != nil {
			return nil, err
		}
	} else if request.Replace {
		URI := gcp.ExpandMeta(context, getRequest.uri)
		replaceCall := service.ReplaceService(URI, srv)
		replaceCall.Context(client.Context())
		if getResponse.Service, err = replaceCall.Do(); err != nil {
			return nil, err
		}
	}

	if err = s.RunInBackground(context, func() error {
		getResponse, err = s.waitForServiceDeployment(context, getRequest, waitTimeoutMs)
		return err
	}); err != nil {
		return nil, err
	}

	configResponse, err := s.getConfiguration(context, &GetConfigurationRequest{
		Name: request.Name,
	})
	if err == nil {
		response.Configuration = configResponse.Configuration
	}
	response.Endpoint = getResponse.Status.Url
	response.Header = nil

	if len(request.Members) > 0 {
		resource := gcp.ExpandMeta(context, request.resource)
		err = s.updateInvokers(context, resource, request.Members...)
	}
	return response, err
}

func (s *service) updateInvokers(context *endly.Context, resource string, members ...string) error {
	ctxClient, err := GetClient(context)
	if err != nil {
		return err
	}
	projectService := run.NewProjectsLocationsServicesService(ctxClient.service)
	call := projectService.GetIamPolicy(resource)
	call.Context(ctxClient.Context())
	policy, err := call.Do()
	if err != nil || policy == nil {
		return nil
	}

	if len(policy.Bindings) == 0 {
		policy.Bindings = make([]*run.Binding, 0)
	}
	updated := false
	for _, binding := range policy.Bindings {
		if binding.Role == runInvokerRole {
			binding.Members = members
			updated = true
			break
		}
	}
	if !updated {
		policy.Bindings = append(policy.Bindings, &run.Binding{
			Members: members,
			Role:    runInvokerRole,
		})
	}
	policyRequest := &run.SetIamPolicyRequest{
		Policy: policy,
	}
	updatePolicyCall := projectService.SetIamPolicy(resource, policyRequest)
	updatePolicyCall.Context(ctxClient.Context())
	_, err = updatePolicyCall.Do()
	return err
}

func (s *service) waitForServiceDeployment(context *endly.Context, request *GetServiceRequest, timeoutMs int) (*GetServiceResponse, error) {
	var response *GetServiceResponse
	var err error
	sleepTime := 3 * time.Second
	timeout := time.Millisecond*time.Duration(timeoutMs) - sleepTime
	startTime := time.Now()
	for !context.IsClosed() {
		response, err = s.getService(context, request)
		if err != nil {
			return nil, err
		}
		if response == nil {
			return &GetServiceResponse{}, nil
		}
		if response.Service == nil {
			return response, nil
		}
		if isServiceReady(response.Status) {
			return response, nil
		}
		if time.Now().Sub(startTime) >= timeout {
			break
		}
		time.Sleep(3 * time.Second)
	}
	return response, nil
}

// New creates a new run service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
