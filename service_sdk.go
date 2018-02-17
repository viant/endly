package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

var errSdkNotFound = errors.New("SDK NOT FUND")

const (
	//SdkServiceID represents system sdk
	SdkServiceID = "sdk"
)

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

func (s *systemSdkService) deploySdk(context *Context, request *SdkSetRequest) error {
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
	if serviceResponse.err != nil {
		return serviceResponse.err
	}
	return nil
}

func (s *systemSdkService) checkSdkOnSession(context *Context, target *url.Resource, request *SdkSetRequest, response *SdkSetResponse) bool {
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

func (s *systemSdkService) setSdk(context *Context, request *SdkSetRequest) (response *SdkSetResponse, err error) {
	response = &SdkSetResponse{}
	service, err := context.Service(ExecServiceID)
	if err != nil {
		return nil, err
	}

	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	if s.checkSdkOnSession(context, target, request, response) {
		return response, nil
	}

	serviceResponse := service.Run(context, &OpenSessionRequest{
		Target: target,
		Env:    request.Env,
	})

	if serviceResponse.err != nil {
		return nil, serviceResponse.err
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

func (s *systemSdkService) setSdkAndDeployIfNeeded(context *Context, request *SdkSetRequest) (response *SdkSetResponse, err error) {
	response, err = s.setSdk(context, request)
	if err == errSdkNotFound {
		err = s.deploySdk(context, request)
		if err == nil {
			response, err = s.setSdk(context, request)
		}
	}
	return response, err
}

const sdkSetExample = `{
  "Sdk": "go",
  "Version": "1.8",
  "Env": {
    "GOOS": "linux",
    "GOPATH": "/Projects/go/workspace/"
  },
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  }
}`

func (s *systemSdkService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "set",
		RequestInfo: &ActionInfo{
			Description: "set sdk on SSH session, deploy SDK if needed",
			Examples: []*ExampleUseCase{
				{
					UseCase: "set go sdk",
					Data:    sdkSetExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SdkSetRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SdkSetResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SdkSetRequest); ok {
				return s.setSdkAndDeployIfNeeded(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewSdkService creates a new system sdk service.
func NewSdkService() Service {
	var result = &systemSdkService{
		jdkService:      &systemJdkService{},
		goService:       &systemGoService{},
		AbstractService: NewAbstractService(SdkServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
