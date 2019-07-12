package sdk

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/deployment/deploy"
	"github.com/viant/endly/system/exec"
	"github.com/viant/toolbox/url"
)

var errSdkNotFound = errors.New("SDK NOT FUND")

const (
	//SdkServiceID represents system sdk
	ServiceID = "sdk"
)

type service struct {
	*endly.AbstractService
	jdkService  *jdkService
	goService   *goService
	nodeService *nodeService
}

func (s *service) updateSessionSdk(context *endly.Context, target *url.Resource, sdkInfo *Info) error {
	if sdkInfo == nil {
		return nil
	}
	session, err := exec.TerminalSession(context, target)
	if err != nil {
		return err
	}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	session.Cacheable[sdkInfo.Sdk] = sdkInfo
	return nil
}

func (s *service) deploySdk(context *endly.Context, request *SetRequest) error {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil
	}
	deploymentService, err := context.Service(deploy.ServiceID)
	if err != nil {
		return err
	}

	ctx := context.Clone()
	state := ctx.State()
	state.Put("buildHost", target.ParsedURL.Host)
	state.Put("buildHostCredential", target.Credentials)
	serviceResponse := deploymentService.Run(ctx, &deploy.Request{
		Target:  target,
		AppName: request.Sdk,
		BaseLocation:request.BaseLocation,
		Version: request.Version,
	})
	if serviceResponse.Err != nil {
		return serviceResponse.Err
	}
	return nil
}

func (s *service) checkSdkOnSession(context *endly.Context, target *url.Resource, request *SetRequest, response *SetResponse) bool {
	session, err := exec.TerminalSession(context, target)
	if err != nil {
		return false
	}
	if session == nil {
		return false
	}
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	cached, has := session.Cacheable[request.Sdk]
	if !has {
		return false
	}
	sdkInfo, ok := cached.(*Info)
	if !ok {
		return false
	}

	if sdkInfo.Version == "" && request.Version == "" {
		response.SdkInfo = sdkInfo
		return true
	}

	if deploy.MatchVersion(request.Version, sdkInfo.Version) {
		response.SdkInfo = sdkInfo
		return true
	}
	return false
}

func (s *service) setSdk(context *endly.Context, request *SetRequest) (response *SetResponse, err error) {
	response = &SetResponse{}
	service, err := context.Service(exec.ServiceID)
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

	serviceResponse := service.Run(context, &exec.OpenSessionRequest{
		Target: target,
		Env:    request.Env,
	})

	if serviceResponse.Err != nil {
		return nil, serviceResponse.Err
	}

	switch request.Sdk {
	case "jdk":
		response.SdkInfo, err = s.jdkService.setSdk(context, request)
	case "go":
		response.SdkInfo, err = s.goService.setSdk(context, request)
	case "node":
		response.SdkInfo, err = s.nodeService.setSdk(context, request)

	default:
		return nil, fmt.Errorf("unsupported jdk: %v", request.Sdk)
	}
	if err == nil {
		err = s.updateSessionSdk(context, target, response.SdkInfo)
	}
	return response, err
}

func (s *service) setSdkAndDeployIfNeeded(context *endly.Context, request *SetRequest) (response *SetResponse, err error) {
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
    "Credentials": "${env.HOME}/.secret/localhost.json"
  }
}`

const sdkNodeSetExample = `{
  "Sdk": "node:9.9",
  "Env": {
    "NODE_ENV": "dev"
  },
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  }
}`

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "set",
		RequestInfo: &endly.ActionInfo{
			Description: "set sdk on SSH session, deploy SDK if needed",
			Examples: []*endly.UseCase{
				{
					Description: "set go sdk",
					Data:        sdkSetExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SetRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SetResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetRequest); ok {
				return s.setSdkAndDeployIfNeeded(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//New creates a new  sdk service.
func New() endly.Service {
	var result = &service{
		jdkService:      &jdkService{},
		goService:       &goService{},
		nodeService:     &nodeService{},
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
