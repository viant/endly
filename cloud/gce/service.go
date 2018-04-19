package gce

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"google.golang.org/api/compute/v1"
)

const (
	//ServiceID represents cce service id.
	ServiceID = "gce"
)

//google comput engine service
type service struct {
	*endly.AbstractService
}

func (s *service) fetchInstanceList(context *endly.Context, request *CallRequest) (CallResponse, error) {
	var response interface{}
	config, err := context.Secrets.GetCredentials(request.Credentials)
	if err != nil {
		return nil, err
	}
	computeClient, ctx, err := NewComputeService(config)
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

func (s *service) call(context *endly.Context, request *CallRequest) (CallResponse, error) {
	if request.Service == "Instances" && request.Method == "List" {
		return s.fetchInstanceList(context, request)
	}
	config, err := context.Secrets.GetCredentials(request.Credentials)
	if err != nil {
		return nil, err
	}
	var response interface{}
	computeClient, ctx, err := NewComputeService(config)
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
  "Credentials": "${env.HOME}/.secret/gce.json",
  "Service": "Instances",
  "Method": "Get",
  "Parameters":["myProject","us-west1-b","instance-1"]	
}
`
	gceGetInstanceStatusResponseExample = `{
  "cpuPlatform": "Intel Skylake",
  "id": "2222222254251138088",
  "kind": "compute#instance",
  "machineType": "https://www.googleapis.com/compute/v1/projects/myproject/zones/us-west1-b/machineTypes/n1-standard-8",
  "metadata": {
    "kind": "compute#metadata"
  },
  "name": "instance-1",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "kind": "compute#accessConfig",
          "name": "External NAT",
          "natIP": "235.147.19.148",
          "type": "ONE_TO_ONE_NAT"
        }
      ],
      "kind": "compute#networkInterface",
      "name": "nic0",
      "network": "https://www.googleapis.com/compute/v1/projects/myproject/global/networks/default",
      "networkIP": "11.128.0.2",
      "subnetwork": "https://www.googleapis.com/compute/v1/projects/myproject/regions/us-west1/subnetworks/default"
    }
  ],
  "scheduling": {
    "automaticRestart": false,
    "onHostMaintenance": "TERMINATE",
    "preemptible": true
  },
  "selfLink": "https://www.googleapis.com/compute/v1/projects/myproject/zones/us-west1-b/instances/instance-1",
  "serviceAccounts": [
    {
      "email": "zzzzzzz-compute@developer.gserviceaccount.com",
      "scopes": [
        "https://www.googleapis.com/auth/cloud-platform"
      ]
    }
  ],
  "status": "RUNNING",
  "tags": {
    "items": [
      "http-server",
      "https-server"
    ]
  },
  "zone": "https://www.googleapis.com/compute/v1/projects/myproject/zones/us-west1-b"
}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "call",
		RequestInfo: &endly.ActionInfo{
			Description: "call proxies request into google.golang.org/api/compute/v1.Service client",
			Examples: []*endly.UseCase{
				{
					Description: "get instance status",
					Data:        gceGetInstanceStatusExample,
				},
				{
					Description: "get instance status response",
					Data:        gceGetInstanceStatusResponseExample,
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
				return s.call(context, req)
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
