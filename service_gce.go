package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"golang.org/x/net/context"
	netcontext "golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
	"reflect"
)

const (
	//GCEServiceID represents cce service id.
	GCEServiceID = "gce"
)

//google comput engine service
type gCEService struct {
	*AbstractService
}

//NewComputeService creates a new compute service.
func NewComputeService(credentialsFile string) (*compute.Service, netcontext.Context, error) {
	resource := url.NewResource(credentialsFile)
	config := &cred.Config{}
	err := resource.JSONDecode(config)
	if err != nil {
		return nil, nil, err
	}
	jwtConfig, err := config.NewJWTConfig(compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, jwtConfig.TokenSource(ctx))
	cilent, err := compute.New(httpClient)
	return cilent, ctx, err
}

//GetComputeService returns specialised compute service for provided service name.
func GetComputeService(client *compute.Service, service string) (interface{}, error) {
	_, found := reflect.TypeOf(*client).FieldByName(service)
	if !found {
		return nil, fmt.Errorf("failed to lookup service %v on google compute service", service)
	}
	return reflect.ValueOf(*client).FieldByName(service).Interface(), nil
}

func (s *gCEService) fetchInstanceList(request *GCECallRequest) (GCECallResponse, error) {
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

func (s *gCEService) call(request *GCECallRequest) (GCECallResponse, error) {
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

func (s *gCEService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "call",
		RequestInfo: &ActionInfo{
			Description: "call proxies request into google.golang.org/api/compute/v1.Service client",
			Examples: []*ExampleUseCase{
				{
					UseCase: "get instance status",
					Data:    gceGetInstanceStatusExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &GCECallRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*GCECallRequest); ok {
				return s.call(handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewGceService creates a new NoOperation service.
func NewGceService() Service {
	var result = &gCEService{
		AbstractService: NewAbstractService(GCEServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
