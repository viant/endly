package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

//DeploymentServiceID represents a deployment service id.
const DeploymentServiceID = "deployment"

type deploymentService struct {
	*AbstractService
}

func (s *deploymentService) extractVersion(context *Context, request *DeploymentDeployRequest, exec Service) (string, error) {
	result, err := context.Execute(request.Transfer.Target, request.VersionCheck)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	if len(result.Extracted) > 0 {
		if version, has := result.Extracted["Version"]; has {
			return version, nil
		}
	}
	return "", nil
}

func (s *deploymentService) deployAddition(context *Context, target *url.Resource, addition *DeploymentAddition) (err error) {
	if addition != nil {
		if len(addition.Commands) > 0 {
			if addition.SuperUser {
				_, err = context.ExecuteAsSuperUser(target, addition.AsCommandRequest().AsManagedCommandRequest().ManagedCommand)
				if err != nil {
					return fmt.Errorf("Failed to init deploy app to %v: %v", target, err)
				}

			} else {
				_, err = context.Execute(target, addition.AsCommandRequest())
				if err != nil {
					return fmt.Errorf("Failed to init deploy app to %v: %v", target, err)
				}
			}

		}
		if len(addition.Transfers) > 0 {
			_, err = context.Transfer(addition.Transfers...)
			if err != nil {
				return fmt.Errorf("Failed to init deploy: %v", err)
			}
		}
	}
	return nil
}

func (s *deploymentService) deploy(context *Context, request *DeploymentDeployRequest) (*DeploymentDeployResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	execService, err := context.Service(SystemExecServiceID)
	if err != nil {
		return nil, err
	}
	target, err := context.ExpandResource(request.Transfer.Target)
	if err != nil {
		return nil, err
	}
	openSessionResponse := execService.Run(context, &OpenSessionRequest{
		Target: target,
	})
	if openSessionResponse.Error != "" {
		return nil, errors.New(openSessionResponse.Error)
	}

	if request.Sdk != "" {
		sdkService, err := context.Service(SdkServiceID)
		if err != nil {
			return nil, err
		}
		openSessionResponse = sdkService.Run(context, &SystemSdkSetRequest{
			Sdk:     request.Sdk,
			Version: request.SdkVersion,
			Target:  request.Transfer.Target,
		})
		if openSessionResponse.Error != "" {
			return nil, errors.New(openSessionResponse.Error)
		}
	}

	defer execService.Run(context, CloseSessionRequest{SessionID: target.Host()})
	if !request.Force && request.VersionCheck != nil {
		version, err := s.extractVersion(context, request, execService)
		if err != nil {
			return nil, fmt.Errorf("Failed to check version: %v", err)
		}
		if version == target.Version {
			return nil, nil
		}
	}

	err = s.deployAddition(context, target, request.Pre)
	if err != nil {
		return nil, err
	}
	_, err = context.Transfer(request.Transfer)
	if err != nil {
		return nil, fmt.Errorf("Failed to deploy: %v", err)
	}

	if request.Command != nil {
		_, err = context.Execute(target, &superUserCommandRequest{
			MangedCommand: request.Command,
		})
		if err != nil {
			return nil, fmt.Errorf("Failed to init deploy app to %v: %v", target, err)
		}
	}
	var version string
	if request.VersionCheck != nil {
		version, err = s.extractVersion(context, request, execService)
		if err != nil {
			return nil, fmt.Errorf("Failed to check version: %v", err)
		}
		if version != target.Version {
			return nil, fmt.Errorf("Failed to deploy %v: invalid version expected: %v, but had: %v ", target.Host(), target.Version, version)
		}
	}
	err = s.deployAddition(context, target, request.Post)
	var response = &DeploymentDeployResponse{
		Version: version,
	}
	return response, err
}

func (s *deploymentService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	switch castedRequest := request.(type) {
	case *DeploymentDeployRequest:
		var err error
		response.Response, err = s.deploy(context, castedRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *deploymentService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "deploy":
		return &DeploymentDeployRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewDeploymentService returns new deployment service
func NewDeploymentService() Service {
	var result = &deploymentService{
		AbstractService: NewAbstractService(DeploymentServiceID),
	}
	result.AbstractService.Service = result
	return result
}
