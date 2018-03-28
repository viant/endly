package deploy

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/workflow"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
	"sync"
)

//ServiceID represents a deployment service id.
const ServiceID = "deployment"

const artifactKey = "artifact"
const versionKey = "Version"

type service struct {
	*endly.AbstractService
	registry map[string]*Meta
	mutex    *sync.RWMutex
}

func (s *service) extractVersion(context *endly.Context, target *url.Resource, deployment *Deployment) (string, error) {
	if deployment.VersionCheck == nil {
		return "", nil
	}

	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, deployment.VersionCheck.Clone(target), runResponse); err != nil {
		return "", err
	}
	if len(runResponse.Data) > 0 {
		if version, has := runResponse.Data[versionKey]; has {
			return version.(string), nil
		}
	}
	return "", nil
}

func (s *service) deployAddition(context *endly.Context, target *url.Resource, addition *Addition) (err error) {
	if addition != nil {
		if len(addition.Commands) > 0 {
			if err = endly.Run(context, addition.AsRunRequest(target), nil); err != nil {
				return fmt.Errorf("failed to init deploy app to %v: %v", target, err)
			}

		}
		if len(addition.Transfers) > 0 {
			_, err = storage.Copy(context, addition.Transfers...)
			if err != nil {
				return fmt.Errorf("failed to init deploy: %v", err)
			}
		}
	}
	return nil
}

func (s *service) matchDeployment(context *endly.Context, version string, target *url.Resource, meta *Meta) (*TargetMeta, error) {
	execService, err := context.Service(exec.ServiceID)
	if err != nil {
		return nil, err
	}
	openSessionResponse := execService.Run(context, &exec.OpenSessionRequest{
		Target: target,
	})
	if openSessionResponse.Error != "" {
		return nil, errors.New(openSessionResponse.Error)
	}
	operatingSystem := exec.OperatingSystem(context, target.Host())
	if operatingSystem == nil {
		return nil, fmt.Errorf("failed to detect operating system on %v", target.Host())
	}

	deployment := meta.Match(operatingSystem, version)
	if deployment == nil {
		return nil, fmt.Errorf("failed to match '%v' deployment with operating system %v and version %v", meta.Name, operatingSystem, version)
	}
	return deployment, nil
}

func (s *service) checkIfDeployedOnSession(context *endly.Context, target *url.Resource, request *Request) bool {
	session, err := exec.TerminalSession(context, target)
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

func (s *service) checkIfDeployedOnSystem(context *endly.Context, target *url.Resource, deploymentTarget *TargetMeta, request *Request) (bool, error) {
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
	transferTarget, err := context.ExpandResource(deploymentTarget.Deployment.Transfer.Dest)
	if err != nil {
		return false, err
	}
	service, err := storage.GetStorageService(context, transferTarget)
	if err != nil {
		return false, err
	}
	return service.Exists(transferTarget.URL)
}

func (s *service) updateSessionDeployment(context *endly.Context, target *url.Resource, app, version string) error {
	session, err := exec.TerminalSession(context, target)
	if err != nil {
		return err
	}
	session.Mutex.Lock()
	defer session.Mutex.Unlock()
	key := app + target.ParsedURL.Path
	session.Deployed[key] = version
	return nil
}

func (s *service) discoverTransfer(context *endly.Context, request *Request, meta *Meta, deploymentTarget *TargetMeta) (*storage.Transfer, error) {
	var state = context.State()
	if meta.Versioning == "" || request.Version == "" {
		return deploymentTarget.Deployment.Transfer, nil
	}
	var transfer = &storage.Transfer{
		Dest:     deploymentTarget.Deployment.Transfer.Dest,
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
		service, err := storage.GetStorageService(context, source)
		if err != nil {
			return nil, err
		}
		var releaseFragmentKey = versioningFragments[len(versioningFragments)-1]
		for i, versionFragment := range requestedVersionFragment {
			artifact.Put(versioningFragments[i], versionFragment)
		}
		minReleaseVersion, has := deploymentTarget.MinReleaseVersion[request.Version]
		if !has {
			minReleaseVersion = ""
		}
		var repeatCount = len(minReleaseVersion)
		if repeatCount == 0 {
			repeatCount = 1
		}
		var maxReleaseVersion = strings.Repeat("9", repeatCount)
		var min = toolbox.AsInt(minReleaseVersion)
		var max = toolbox.AsInt(maxReleaseVersion)

		for i := min; i <= max; i++ {
			artifact.Put(releaseFragmentKey, toolbox.AsString(i))
			if minReleaseVersion == "" && i == 0 {
				artifact.Put(versionKey, "%v")
			} else {
				artifact.Put(versionKey, fmt.Sprintf("%v.%v", request.Version, i))
			}
			var sourceURL = context.Expand(source.URL)
			exists, _ := service.Exists(sourceURL)
			if exists {
				source = url.NewResource(sourceURL, source.Credentials)
				break
			}
		}
	}
	transfer.Source = source
	return transfer, nil
}

func (s *service) deployDependenciesIfNeeded(context *endly.Context, target *url.Resource, dependencies []*Dependency) (err error) {
	if len(dependencies) == 0 {
		return nil
	}
	for _, dependency := range dependencies {
		_, err = s.deploy(context, &Request{
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

func (s *service) updateOperatingSystem(context *endly.Context, target *url.Resource) {
	operatingSystem := exec.OperatingSystem(context, target.Host())
	if operatingSystem != nil {
		osMap := data.NewMap()
		osMap.Put("System", operatingSystem.System)
		osMap.Put("Architecture", operatingSystem.Architecture)
		osMap.Put("Version", operatingSystem.Version)
		osMap.Put("Hardware", operatingSystem.Hardware)
		var state = context.State()
		state.Put("os", osMap)
	}
}

func (s *service) deploy(context *endly.Context, request *Request) (*Response, error) {
	request = &Request{
		AppName: context.Expand(request.AppName),
		Version: context.Expand(request.Version),
		Target:  request.Target,
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	state := context.State()
	if !state.Has("targetHost") {
		state.Put("targetHost", target.ParsedURL.Host)
		state.Put("targetHostCredential", target.Credentials)
	}

	var response = &Response{}
	if s.checkIfDeployedOnSession(context, target, request) {
		response.Version = request.Version
		return response, nil
	}

	if target.ParsedURL.Path != "" {
		if err := endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.ParsedURL.Path)), nil); err != nil {
			if err := endly.Run(context, exec.NewRunRequest(target, false, "cd /"), nil); err != nil {
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
	_, err = storage.Copy(context, transfer)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %v", err)
	}
	if deploymentTarget.Deployment.Run != nil {
		if err = endly.Run(context, deploymentTarget.Deployment.Run.Clone(target), nil); err != nil {
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

func (s *service) loadMeta(context *endly.Context, request *LoadMetaRequest) (*LoadMetaResponse, error) {
	source, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	meta := &Meta{}
	err = source.Decode(meta)
	if err != nil {
		return nil, fmt.Errorf("unable to decode: %v, %v", source.URL, err)
	}
	if err = meta.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate deployment meta: %v %v", source.URL, err)
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.registry[meta.Name] = meta
	return &LoadMetaResponse{
		Meta: meta,
	}, nil
}

func (s *service) getMeta(context *endly.Context, request *Request) (*Meta, error) {
	s.mutex.RLock()
	result, hasMeta := s.registry[request.AppName]
	s.mutex.RUnlock()
	var state = context.State()
	if !hasMeta {
		var metaURL = request.MetaURL
		if metaURL == "" {
			service, err := context.Service(workflow.ServiceID)
			if err != nil {
				return nil, err
			}
			if workflowService, ok := service.(*workflow.Service); ok {
				workflowResource, err := workflowService.Dao.NewRepoResource(state, fmt.Sprintf("meta/deployment/%v.json", request.AppName))
				if err != nil {
					return nil, err
				}
				metaURL = workflowResource.URL

			}
		}
		var credentials = ""
		mainWorkflow := workflow.LastWorkflow(context)
		if mainWorkflow != nil {
			credentials = mainWorkflow.Source.Credentials
		}
		response, err := s.loadMeta(context, &LoadMetaRequest{
			Source: url.NewResource(metaURL, credentials),
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
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "AppName": "tomcat",
  "Version": "7.0",
  "Force": true
}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "deploy",
		RequestInfo: &endly.ActionInfo{
			Description: "deploy specific app version on target host, if existing app version matches requested version, deployment is skipped",
			Examples: []*endly.UseCase{
				{
					Description: "tomcat deploy",
					Data:        deploymentTomcatDeployExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &Request{}
		},
		ResponseProvider: func() interface{} {
			return &Response{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*Request); ok {
				return s.deploy(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "load",
		RequestInfo: &endly.ActionInfo{
			Description: "load deployment meta instruction into registry",
		},
		RequestProvider: func() interface{} {
			return &LoadMetaRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LoadMetaResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoadMetaRequest); ok {
				return s.loadMeta(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new deployment service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		mutex:           &sync.RWMutex{},
		registry:        make(map[string]*Meta),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
