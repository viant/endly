package endly

import (
	"fmt"
	"net/url"
)

const DeploymentServiceId = "deployment"

type DeploymentConfig struct {
	Before       *ManagedCommand
	Transfer     *TransferRequest
	After        *ManagedCommand
	VersionCheck *ManagedCommand
	AppName      string
	Force        bool
}

//TODO add global path in the target
type deploymentService struct {
	*AbstractService
}

func (r *DeploymentConfig) Validate() error {
	if r.Transfer.Target == nil {
		return fmt.Errorf("Failed to deploy app, target was not specified")
	}
	if r.Transfer.Target.URL == "" {
		return fmt.Errorf("Failed to deploy app, target URL was empty")
	}
	if r.Transfer.Source.URL == "" {
		return fmt.Errorf("Failed to deploy app, source URL was empty")
	}
	return nil
}

func (s *deploymentService) extractVersion(context *Context, request *DeploymentConfig, exec Service, parsedURL *url.URL) (string, error) {
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

func (s *deploymentService) deploy(context *Context, request *DeploymentConfig) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	execService, err := context.Service(ExecServiceId)
	if err != nil {
		return nil, err
	}

	target, err := context.ExpandResource(request.Transfer.Target)
	if err != nil {
		return nil, err
	}
	response := execService.Run(context, &OpenSession{
		Target: target,
	})
	if response.Error != nil {
		return nil, response.Error
	}
	defer execService.Run(context, CloseSession{Name: target.Session()})
	parsedURL, err := url.Parse(target.URL)
	if err != nil {
		return nil, err
	}

	_, err = context.Execute(target, request.Before)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return "", fmt.Errorf("Failed to deploy app to %v: %v", target, err)
	}
	if !request.Force && request.VersionCheck != nil {
		version, err := s.extractVersion(context, request, execService, parsedURL)
		if err != nil {
			return nil, fmt.Errorf("Failed to check version: %v", err)
		}
		if version == target.Version {
			return nil, nil
		}
	}
	_, err = context.Transfer(request.Transfer)
	if err != nil {
		return nil, fmt.Errorf("Failed to deploy: %v", err)
	}
	_, err = context.Execute(target, request.After)
	if err != nil {
		return nil, err
	}

	if request.VersionCheck != nil {
		version, err := s.extractVersion(context, request, execService, parsedURL)
		if err != nil {
			return nil, fmt.Errorf("Failed to check version: %v", err)
		}
		if version != target.Version {
			return nil, fmt.Errorf("Failed to deploy %v: invalud version expected: %v, but had: %v ", target.Session(), target.Version, version)
		}
	}
	return nil, err
}

func (s *deploymentService) Run(context *Context, request interface{}) *Response {
	var response = &Response{
		Status: "ok",
	}
	switch castedRequest := request.(type) {
	case *DeploymentConfig:
		response.Response, response.Error = s.deploy(context, castedRequest)
	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}
	if response.Error != nil {
		response.Status = "error"
	}
	return response
}

func (s *deploymentService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "deploy":
		return &DeploymentConfig{}, nil
	}
	return nil, fmt.Errorf("Unsupported request: %v", name)
}

func NewDeploymentService() Service {
	var result = &deploymentService{
		AbstractService: NewAbstractService(DeploymentServiceId),
	}
	result.AbstractService.Service = result
	return result
}
