package endly

import (
	"errors"
	"fmt"
	"net/url"
)

const DeploymentServiceId = "deployment"

type DeploymentAddition struct {
	SuperUser bool
	Commands  []string
	Transfers []*Transfer
}

func (a *DeploymentAddition) AsCommandRequest() *CommandRequest {
	return &CommandRequest{
		Commands:  a.Commands,
		SuperUser: a.SuperUser,
	}
}

type DeploymentDeployRequest struct {
	Sdk          string
	SdkVersion   string
	Pre          *DeploymentAddition
	Transfer     *Transfer
	Command      *ManagedCommand
	VersionCheck *ManagedCommand
	Post         *DeploymentAddition
	AppName      string
	Force        bool
}

//TODO add global path in the target
type deploymentService struct {
	*AbstractService
}

func (r *DeploymentDeployRequest) Validate() error {
	if r.Transfer == nil {
		return fmt.Errorf("Failed to deploy app, transfer was nil")
	}
	if r.Transfer.Target == nil {
		return fmt.Errorf("Failed to deploy app, target was not specified")
	}
	if r.Transfer.Target.URL == "" {
		return fmt.Errorf("Failed to deploy app, target URL was empty")
	}
	if r.Transfer.Source.URL == "" {
		return fmt.Errorf("Failed to deploy app, Source URL was empty")
	}
	return nil
}

func (s *deploymentService) extractVersion(context *Context, request *DeploymentDeployRequest, exec Service, parsedURL *url.URL) (string, error) {
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

func (s *deploymentService) deployAddition(context *Context, target *Resource, addition *DeploymentAddition) (err error) {
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

func (s *deploymentService) deploy(context *Context, request *DeploymentDeployRequest) (interface{}, error) {
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
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	if request.Sdk != "" {
		sdkService, err := context.Service(SdkServiceId)
		if err != nil {
			return nil, err
		}
		response = sdkService.Run(context, &SdkSetRequest{
			Sdk:     request.Sdk,
			Version: request.SdkVersion,
			Target:  request.Transfer.Target,
		})
		if response.Error != "" {
			return nil, errors.New(response.Error)
		}
	}

	defer execService.Run(context, CloseSession{Name: target.Session()})
	parsedURL, err := url.Parse(target.URL)
	if err != nil {
		return nil, err
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

	err = s.deployAddition(context, target, request.Pre)
	if err != nil {
		return nil, err
	}
	_, err = context.Transfer(request.Transfer)
	if err != nil {
		return nil, fmt.Errorf("Failed to deploy: %v", err)
	}

	if request.Command != nil {
		_, err = context.Execute(target, request.Command)
		if err != nil {
			return nil, fmt.Errorf("Failed to init deploy app to %v: %v", target, err)
		}
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
	err = s.deployAddition(context, target, request.Post)
	return nil, err
}

func (s *deploymentService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{
		Status: "ok",
	}
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

func NewDeploymentService() Service {
	var result = &deploymentService{
		AbstractService: NewAbstractService(DeploymentServiceId),
	}
	result.AbstractService.Service = result
	return result
}
