package endly

import (
	"errors"
	"fmt"
	"strings"
)

var sdkNotFound = errors.New("SDK NOT FUND")

//SdkServiceID represents system sdk
const SdkServiceID = "sdk"

type systemSdkService struct {
	*AbstractService
	jdkService *systemJdkService
	goService  *systemGoService
}

func (s *systemSdkService) loadSdkMeta(context *Context, request *SystemSdkSetRequest) error {
	service, err := context.Service(WorkflowServiceID)
	if err != nil {
		return err
	}
	if workflowService, ok := service.(*workflowService); ok {
		var version = strings.Replace(request.Version, ".", "_", len(request.Version))
		sdkResource, err := workflowService.Dao.NewRepoResource(context.state, fmt.Sprintf("sdk/%v%v.json", request.Sdk, version))
		if err != nil {
			return err
		}
		request := &SystemSdkRegisterMetaDeploymentRequest{}
		err = sdkResource.JSONDecode(request)
		if err != nil {
			return err
		}
		_, err = s.registerMeta(context, request)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *systemSdkService) deploySdk(context *Context, request *SystemSdkSetRequest) error {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil
	}
	meta, err := sdkDeploymentMetaRegistry.Get(request.Sdk, request.Version)
	if err != nil {
		s.loadSdkMeta(context, request)
		meta, err = sdkDeploymentMetaRegistry.Get(request.Sdk, request.Version)
		if err != nil {
			return err
		}
	}
	operatingSystem := context.OperatingSystem(target.Host())
	deploymentRequest, err := meta.Match(operatingSystem)
	if err != nil {
		return err
	}
	deploymentService, err := context.Service(DeploymentServiceID)
	if err != nil {
		return err
	}
	response := deploymentService.Run(context, deploymentRequest)
	if response.Error != "" {
		return fmt.Errorf("Failed to deploy sdk %v", response.Error)
	}
	return nil
}

func (s *systemSdkService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))

	var err error
	switch actualRequest := request.(type) {
	case *SystemSdkSetRequest:
		response.Response, err = s.setSdk(context, actualRequest)
		if err == sdkNotFound {
			err = s.deploySdk(context, actualRequest)
			if err == nil {
				response.Response, err = s.setSdk(context, actualRequest)
			}
		}
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run sdk: %v, %v", actualRequest.Sdk, err)
		}
	case *SystemSdkRegisterMetaDeploymentRequest:
		response.Response, err = s.registerMeta(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run sdk: %v, %v", actualRequest.Meta, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}
func (service *systemSdkService) registerMeta(context *Context, request *SystemSdkRegisterMetaDeploymentRequest) (*SystemSdkRegisterMetaDeploymentResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	sdkDeploymentMetaRegistry.Register(request.Meta)
	var response = &SystemSdkRegisterMetaDeploymentResponse{
		Sdk:        request.Meta.Sdk,
		SdkVersion: request.Meta.SdkVersion,
	}
	return response, nil
}

func (s *systemSdkService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "set":
		return &SystemSdkSetRequest{}, nil
	case "register":
		return &SystemSdkRegisterMetaDeploymentRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)

}

func (s *systemSdkService) setSdk(context *Context, request *SystemSdkSetRequest) (*SystemSdkSetResponse, error) {
	switch request.Sdk {
	case "jdk":
		return s.jdkService.setSdk(context, request)
	case "go":
		return s.goService.setSdk(context, request)
	}
	return nil, fmt.Errorf("Unsupported jdk: %v", request.Sdk)
}

//NewSystemJdkService creates a new system jdk service.
func NewSystemJdkService() Service {
	var result = &systemSdkService{
		jdkService:      &systemJdkService{},
		goService:       &systemGoService{},
		AbstractService: NewAbstractService(SdkServiceID),
	}
	result.AbstractService.Service = result
	return result
}
