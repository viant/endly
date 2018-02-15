package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
	"sync"
)

//DeploymentServiceID represents a deployment service id.
const DeploymentServiceID = "deployment"

const artifactKey = "artifact"
const versionKey = "Version"

//DeploymentDependency represents deployment dependency
type DeploymentDependency struct {
	Name    string
	Version string
}

type deploymentService struct {
	*AbstractService
	registry map[string]*DeploymentMeta
	mutex    *sync.RWMutex
}

func (s *deploymentService) extractVersion(context *Context, target *url.Resource, deployment *Deployment) (string, error) {
	if deployment.VersionCheck == nil {
		return "", nil
	}
	result, err := context.Execute(target, deployment.VersionCheck)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	if len(result.Extracted) > 0 {
		if version, has := result.Extracted[versionKey]; has {
			return version, nil
		}
	}
	return "", nil
}

func (s *deploymentService) deployAddition(context *Context, target *url.Resource, addition *DeploymentAddition) (err error) {
	if addition != nil {
		if len(addition.Commands) > 0 {
			if addition.SuperUser {
				_, err = context.ExecuteAsSuperUser(target, addition.AsCommandRequest().AsExtractableCommandRequest().ExtractableCommand)
				if err != nil {
					return fmt.Errorf("failed to init deploy app to %v: %v", target, err)
				}

			} else {
				_, err = context.Execute(target, addition.AsCommandRequest())
				if err != nil {
					return fmt.Errorf("failed to init deploy app to %v: %v", target, err)
				}
			}

		}
		if len(addition.Transfers) > 0 {
			_, err = context.Transfer(addition.Transfers...)
			if err != nil {
				return fmt.Errorf("failed to init deploy: %v", err)
			}
		}
	}
	return nil
}

func (s *deploymentService) matchDeployment(context *Context, version string, target *url.Resource, meta *DeploymentMeta) (*DeploymentTargetMeta, error) {
	execService, err := context.Service(ExecServiceID)
	if err != nil {
		return nil, err
	}
	openSessionResponse := execService.Run(context, &OpenSessionRequest{
		Target: target,
	})
	if openSessionResponse.Error != "" {
		return nil, errors.New(openSessionResponse.Error)
	}
	operatingSystem := context.OperatingSystem(target.Host())
	if operatingSystem == nil {
		return nil, fmt.Errorf("failed to detect operating system on %v", target.Host())
	}

	deployment := meta.Match(operatingSystem, version)
	if deployment == nil {
		return nil, fmt.Errorf("failed to match '%v' deployment with operating system %v and version %v", meta.Name, operatingSystem, version)
	}
	return deployment, nil
}

func (s *deploymentService) checkIfDeployedOnSession(context *Context, target *url.Resource, request *DeploymentDeployRequest) bool {
	session, err := context.TerminalSession(target)
	if err != nil {
		return false
	}
	session.Mutex.RLock()
	defer session.Mutex.RUnlock()
	var key = request.AppName + target.ParsedURL.Path
	deployedVersion, has := session.Deployed[key]
	if !has {
		return false
	}
	if deployedVersion == "" && request.Version == "" {
		return true
	}
	return MatchVersion(request.Version, deployedVersion)
}

func (s *deploymentService) checkIfDeployedOnSystem(context *Context, target *url.Resource, deploymentTarget *DeploymentTargetMeta, request *DeploymentDeployRequest) (bool, error) {
	if deploymentTarget.Deployment.VersionCheck != nil {
		actualVersion, err := s.extractVersion(context, target, deploymentTarget.Deployment)
		if err != nil || actualVersion == "" {
			return false, err
		}
		if actualVersion == "" {
			return false, nil
		}
		return MatchVersion(request.Version, actualVersion), nil
	}
	transferTarget, err := context.ExpandResource(deploymentTarget.Deployment.Transfer.Target)
	if err != nil {
		return false, err
	}
	service, err := getStorageService(context, transferTarget)
	if err != nil {
		return false, err
	}
	return service.Exists(transferTarget.URL)
}

func (s *deploymentService) updateSessionDeployment(context *Context, target *url.Resource, app, version string) error {
	session, err := context.TerminalSession(target)
	if err != nil {
		return err
	}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	key := app + target.ParsedURL.Path
	session.Deployed[key] = version
	return nil
}

func (s *deploymentService) discoverTransfer(context *Context, request *DeploymentDeployRequest, meta *DeploymentMeta, deploymentTarget *DeploymentTargetMeta) (*Transfer, error) {
	var state = context.state
	if meta.Versioning == "" || request.Version == "" {
		return deploymentTarget.Deployment.Transfer, nil
	}
	var transfer = &Transfer{
		Target:   deploymentTarget.Deployment.Transfer.Target,
		Expand:   deploymentTarget.Deployment.Transfer.Expand,
		Replace:  deploymentTarget.Deployment.Transfer.Replace,
		Compress: deploymentTarget.Deployment.Transfer.Compress,
	}

	var source = deploymentTarget.Deployment.Transfer.Source
	var artifact = data.NewMap()
	state.Put(artifactKey, artifact)
	var versioningFragments = strings.Split(meta.Versioning, ".")
	var requestedVersionFragment = strings.Split(request.Version, ".")
	if len(deploymentTarget.MinReleaseVersion) == 0 || len(versioningFragments) == len(requestedVersionFragment) {
		artifact.Put(versionKey, request.Version)
		for i, fragmentKey := range versioningFragments {
			artifact.Put(fragmentKey, requestedVersionFragment[i])
		}

	} else {
		service, err := getStorageService(context, source)
		if err != nil {
			return nil, err
		}
		var releaseFragmentKey = versioningFragments[len(versioningFragments)-1]
		for i, versionFragment := range requestedVersionFragment {
			artifact.Put(versioningFragments[i], versionFragment)
		}
		minReleaseVersion, has := deploymentTarget.MinReleaseVersion[request.Version]
		if !has {
			return nil, fmt.Errorf("failed to discover source - unable to determine minReleaseVersion for %v", request.Version)
		}
		var maxReleaseVersion = strings.Repeat("9", len(minReleaseVersion))
		var min = toolbox.AsInt(minReleaseVersion)
		var max = toolbox.AsInt(maxReleaseVersion)
		for i := min; i <= max; i++ {
			artifact.Put(releaseFragmentKey, toolbox.AsString(i))
			artifact.Put(versionKey, fmt.Sprintf("%v.%v", request.Version, i))
			var sourceURL = context.Expand(source.URL)
			exists, _ := service.Exists(sourceURL)
			if exists {
				source = url.NewResource(sourceURL, source.Credential)
				break
			}
		}
	}
	transfer.Source = source
	return transfer, nil
}

func (s *deploymentService) deployDependenciesIfNeeded(context *Context, target *url.Resource, dependencies []*DeploymentDependency) (err error) {
	if len(dependencies) == 0 {
		return nil
	}
	for _, dependency := range dependencies {
		_, err = s.deploy(context, &DeploymentDeployRequest{
			Target:  target,
			AppName: dependency.Name,
			Version: dependency.Version,
		})
		if err != nil {
			break
		}
	}
	return err
}

func (s *deploymentService) updateOperatingSystem(context *Context, target *url.Resource) {
	operatingSystem := context.OperatingSystem(target.Host())
	if operatingSystem != nil {
		osMap := data.NewMap()
		osMap.Put("System", operatingSystem.System)
		osMap.Put("Architecture", operatingSystem.Architecture)
		osMap.Put("Version", operatingSystem.Version)
		osMap.Put("Hardware", operatingSystem.Hardware)
		context.state.Put("os", osMap)
	}
}

func (s *deploymentService) deploy(context *Context, request *DeploymentDeployRequest) (*DeploymentDeployResponse, error) {
	request = &DeploymentDeployRequest{
		AppName: context.Expand(request.AppName),
		Version: context.Expand(request.Version),
		Target:  request.Target,
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	state := context.state
	if !state.Has("targetHost") {
		state.Put("targetHost", target.ParsedURL.Host)
		state.Put("targetHostCredential", target.Credential)
	}

	var response = &DeploymentDeployResponse{}
	if s.checkIfDeployedOnSession(context, target, request) {
		response.Version = request.Version
		return response, nil
	}

	if target.ParsedURL.Path != "" {

		if _, err := context.Execute(target, fmt.Sprintf("cd %v", target.ParsedURL.Path)); err != nil {
			if _, err = context.Execute(target, "cd /"); err != nil {
				return nil, err
			}
		}

	}

	meta, err := s.getMeta(context, request)
	if err != nil {
		return nil, err
	}

	var expectedVersion = context.Expand(request.Version)
	deploymentTarget, err := s.matchDeployment(context, expectedVersion, target, meta)
	if err != nil {
		return nil, err
	}

	s.updateOperatingSystem(context, target)
	err = s.deployDependenciesIfNeeded(context, target, deploymentTarget.Dependencies)
	if err != nil {
		return nil, err
	}
	if !request.Force {
		if deployed, _ := s.checkIfDeployedOnSystem(context, target, deploymentTarget, request); deployed {
			return response, err
		}
	}
	transfer, err := s.discoverTransfer(context, request, meta, deploymentTarget)
	if err != nil {
		return nil, err
	}
	var artifact = state.GetMap(artifactKey)
	if artifact != nil {
		response.Version = artifact.GetString("")
	}
	defer state.Delete(artifactKey)
	err = s.deployAddition(context, target, deploymentTarget.Deployment.Pre)
	if err != nil {
		return nil, err
	}
	_, err = context.Transfer(transfer)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %v", err)
	}
	if deploymentTarget.Deployment.Command != nil {
		_, err = context.Execute(target,
			deploymentTarget.Deployment.Command,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to init deploy app to %v: %v", target, err)
		}
	}

	err = s.deployAddition(context, target, deploymentTarget.Deployment.Post)
	if deployed, _ := s.checkIfDeployedOnSystem(context, target, deploymentTarget, request); deployed {
		var version = request.Version
		if version == "" {
			version = response.Version
		}
		err = s.updateSessionDeployment(context, target, request.AppName, version)
		return response, err
	}

	return nil, fmt.Errorf("failed to deploy %v, unable to verify deployments", request.AppName)
}




func (s *deploymentService) loadMeta(context *Context, request *DeploymentLoadMetaRequest) (*DeploymentLoadMetaResponse, error) {
	source, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	meta := &DeploymentMeta{}
	err = source.JSONDecode(meta)
	if err != nil {
		return nil, fmt.Errorf("unable to decode: %v, %v", source.URL, err)
	}
	if err = meta.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate deployment meta: %v %v", source.URL, err)
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.registry[meta.Name] = meta
	return &DeploymentLoadMetaResponse{
		Meta: meta,
	}, nil
}

func (s *deploymentService) getMeta(context *Context, request *DeploymentDeployRequest) (*DeploymentMeta, error) {
	s.mutex.RLock()
	result, hasMeta := s.registry[request.AppName]
	s.mutex.RUnlock()
	var state = context.state
	if !hasMeta {
		var metaURL = request.MetaURL
		if metaURL == "" {
			service, err := context.Service(WorkflowServiceID)
			if err != nil {
				return nil, err
			}
			if workflowService, ok := service.(*workflowService); ok {
				workflowResource, err := workflowService.Dao.NewRepoResource(state, fmt.Sprintf("meta/deployment/%v.json", request.AppName))
				if err != nil {
					return nil, err
				}
				metaURL = workflowResource.URL

			}
		}
		var credential = ""
		mainWorkflow := context.Workflow()
		if mainWorkflow != nil {
			credential = mainWorkflow.Source.Credential
		}
		response, err := s.loadMeta(context, &DeploymentLoadMetaRequest{
			Source: url.NewResource(metaURL, credential),
		})
		if err != nil {
			return nil, err
		}
		result = response.Meta
	}
	return result, nil
}



const (
	deploymentTomcatDeployExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/opt/server/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "AppName": "tomcat",
  "Version": "7.0",
  "Force": true
}`
)



func (s *deploymentService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "deploy",
		RequestInfo: &ActionInfo{
			Description: "deploy specific app version on target host, if existing app version matches requested version, deployment is skipped",
			Examples: []*ExampleUseCase{
				{
					UseCase: "tomcat deploy",
					Data:deploymentTomcatDeployExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DeploymentDeployRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeploymentDeployResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DeploymentDeployRequest); ok {
				return s.deploy(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "load",
		RequestInfo: &ActionInfo{
			Description: "load deployment meta instruction into registry",
		},
		RequestProvider: func() interface{} {
			return &DeploymentLoadMetaRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeploymentLoadMetaResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DeploymentLoadMetaRequest); ok {
 				return s.loadMeta(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}



//NewDeploymentService returns new deployment service
func NewDeploymentService() Service {
	var result = &deploymentService{
		AbstractService: NewAbstractService(DeploymentServiceID),
		mutex:    &sync.RWMutex{},
		registry: make(map[string]*DeploymentMeta),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
