package gce

import (
	"fmt"
	"google.golang.org/api/compute/v1"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

const (
	//ServiceID represents cce service id.
	ServiceID = "gce"
)

//google comput engine service
type service struct {
	*endly.AbstractService
}


func (s *service) fetchInstanceList(request *CallRequest) (CallResponse, error) {
	var response interface{}
	computeClient, ctx, err := NewComputeService(request.Credential)
	if err != nil {
		return nil, err
	}
	project := toolbox.AsString(request.Parameters[0])
	zone := toolbox.AsString(request.Parameters[1])
	req := computeClient.Instances.List(project, zone)
	var instances = make([]*compute.Instance, 0)
	err = req.Pages(ctx, func(list *compute.InstanceList) error {
		instances = append(instances, list.Items...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	response = instances
	return response, nil
}

func (s *service) call(request *CallRequest) (CallResponse, error) {
	if request.Service == "Instances" && request.Method == "List" {
		return s.fetchInstanceList(request)
	}
	var response interface{}
	computeClient, ctx, err := NewComputeService(request.Credential)
	if err != nil {
		return nil, err
	}
	computeService, err := GetComputeService(computeClient, request.Service)

	method, err := toolbox.GetFunction(computeService, request.Method)
	if err != nil {
		return nil, err
	}
	parameters, err := toolbox.AsCompatibleFunctionParameters(method, request.Parameters)
	if err != nil {
		return nil, err
	}
	functionResult := toolbox.CallFunction(method, parameters...)
	if len(functionResult) != 1 {
		return nil, fmt.Errorf("unsupported operation: %v.%v", request.Service, request.Method)
	}
	callRequest := functionResult[0]
	contextFunction, err := toolbox.GetFunction(callRequest, "Context")
	if err != nil {
		return nil, err
	}
	contextResult := toolbox.CallFunction(contextFunction, ctx)
	if len(contextResult) != 1 {
		return nil, fmt.Errorf("unsupported operation: %v.%v", request.Service, request.Method)
	}

	if doFunction, err := toolbox.GetFunction(callRequest, "Do"); err == nil {
		doResult := toolbox.CallFunction(doFunction)
		if len(doResult) > 0 {
			if err, ok := doResult[len(doResult)-1].(error); ok {
				return nil, err
			}
			response = doResult[0]
			return response, nil
		}
	}
	return nil, fmt.Errorf("unsupported operation: %v.%v", request.Service, request.Method)
}

const (
	gceGetInstanceStatusExample = `{
  "Credential": "${env.HOME}/.secret/gce.json",
  "Service": "Instances",
  "Method": "Get",
  "Parameters":["myProject","us-west1-b","instance-1"]	
}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.ServiceActionRoute{
		Action: "call",
		RequestInfo: &endly.ActionInfo{
			Description: "call proxies request into google.golang.org/api/compute/v1.Service client",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "get instance status",
					Data:    gceGetInstanceStatusExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CallRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CallRequest); ok {
				return s.call(req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
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
