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

	//GceServiceGceAction represents cce action
	GCEServiceCallAction = "call"
)

//google comput engine service
type gCEService struct {
	*AbstractService
}

func (s *gCEService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: request}
	var err error
	switch actualRequest := request.(type) {
	case *GCECallRequest:
		response.Response, err = s.call(actualRequest)
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *gCEService) NewRequest(action string) (interface{}, error) {
	switch action {
	case GCEServiceCallAction:
		return &GCECallRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

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

func GetComputeService(client *compute.Service, service string) (interface{}, error) {
	_, found := reflect.TypeOf(*client).FieldByName(service)
	if !found {
		return nil, fmt.Errorf("failed to lookup service %v on google compute service", service)
	}
	return reflect.ValueOf(*client).FieldByName(service).Interface(), nil
}

func (s *gCEService) fetchInstanceList(request *GCECallRequest) (*GCECallResponse, error) {
	var response = &GCECallResponse{}
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
		response.Error = err.Error()
	}
	response.Response = instances
	return response, nil
}

func (s *gCEService) call(request *GCECallRequest) (*GCECallResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if request.Service == "Instances" && request.Method == "List" {
		return s.fetchInstanceList(request)
	}

	var response = &GCECallResponse{}
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
				response.Error = err.Error()
			}
			response.Response = doResult[0]
			return response, nil
		}
	}
	return nil, fmt.Errorf("unsupported operation: %v.%v", request.Service, request.Method)
}

//NewGceService creates a new NoOperation service.
func NewGceService() Service {
	var result = &gCEService{
		AbstractService: NewAbstractService(GCEServiceID,
			GCEServiceCallAction),
	}
	result.AbstractService.Service = result
	return result
}
