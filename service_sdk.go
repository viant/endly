package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

var sdkNotFound = errors.New("SDK NOT FUND")

//SdkServiceID represents system sdk
const SdkServiceID = "sdk"

type systemSdkService struct {
	*AbstractService
	jdkService *systemJdkService
	goService  *systemGoService
}

func (s *systemSdkService) updateSessionSdk(context *Context, target *url.Resource, sdkInfo *SystemSdkInfo) error {
	if sdkInfo == nil {
		return nil
	}
	session, err := context.TerminalSession(target)
	if err != nil {
		return err
	}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	session.Sdk[sdkInfo.Sdk] = sdkInfo
	return nil
}

func (s *systemSdkService) deploySdk(context *Context, request *SystemSdkSetRequest) error {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil
	}
	deploymentService, err := context.Service(DeploymentServiceID)
	if err != nil {
		return err
	}
	serviceResponse := deploymentService.Run(context, &DeploymentDeployRequest{
		Target:  target,
		AppName: request.Sdk,
		Version: request.Version,
	})
	if serviceResponse.Error != "" {
		return fmt.Errorf("failed to deploy sdk: %v %v, %v", request.Sdk, request.Version, serviceResponse.Error)
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
			response.Error = fmt.Sprintf("failed to run sdk: %v, %v", actualRequest.Sdk, err)
		}
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *systemSdkService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "set":
		return &SystemSdkSetRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)

}

func (s *systemSdkService) checkSdkOnSession(context *Context, target *url.Resource, request *SystemSdkSetRequest, response *SystemSdkSetResponse) bool {
	session, err := context.TerminalSession(target)
	if err != nil {
		return false
	}
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	sdkInfo, has := session.Sdk[request.Sdk]
	if !has {
		return false
	}
	if sdkInfo.Version == "" && request.Version == "" {
		response.SdkInfo = sdkInfo
		return true
	}
	if MatchVersion(request.Version, sdkInfo.Version) {
		response.SdkInfo = sdkInfo
		return true
	}
	return false
}

func (s *systemSdkService) setSdk(context *Context, request *SystemSdkSetRequest) (response *SystemSdkSetResponse, err error) {
	response = &SystemSdkSetResponse{}
	service, err := context.Service(ExecServiceID)
	if err != nil {
		return nil, err
	}

	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	if s.checkSdkOnSession(context, target, request, response) {

	}

	serviceResponse := service.Run(context, &OpenSessionRequest{
		Target: target,
		Env:    request.Env,
	})

	if serviceResponse.Error != "" {
		return nil, fmt.Errorf("failed to set sdk %v", serviceResponse.Error)
	}

	switch request.Sdk {
	case "jdk":
		response.SdkInfo, err = s.jdkService.setSdk(context, request)
	case "go":
		response.SdkInfo, err = s.goService.setSdk(context, request)

	default:
		return nil, fmt.Errorf("unsupported jdk: %v", request.Sdk)
	}

	s.updateSessionSdk(context, target, response.SdkInfo)
	return response, err
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
