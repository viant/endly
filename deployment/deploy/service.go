package deploy

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/system/storage/copy"
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
	var extractRequest = deployment.VersionCheck.Clone(target)
	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
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
	sessionID := exec.SessionID(context, target)
	operatingSystem := exec.OperatingSystem(context, sessionID)
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

	dest, storageOpts, err := storage.GetResourceWithOptions(context, deploymentTarget.Deployment.Transfer.Dest)
	if err != nil {
		return false, err
	}
	service, err := storage.StorageService(context, dest)
	if err != nil {
		return false, err
	}
	return service.Exists(context.Background(), dest.URL, storageOpts...)
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

//TODO break it down - too large and messy
func (s *service) discoverTransfer(context *endly.Context, request *Request, meta *Meta, deploymentTarget *TargetMeta) (*copy.Rule, error) {
	var state = context.State()
	transfer := deploymentTarget.Deployment.Transfer
	if meta.Versioning == "" || request.Version == "" {
		return transfer, nil
	}

	transfer = transfer.Clone()
	var source = transfer.Source
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
		if len(requestedVersionFragment) == 1 && len(requestedVersionFragment) < len(versioningFragments) {
			for _, target := range meta.Targets {
				for k := range target.MinReleaseVersion {
					parts := strings.Split(k, ".")
					if parts[0] == request.Version {
						request.Version = k
						versioningFragments = strings.Split(meta.Versioning, ".")
						break
					}
				}
			}
		}

		source, storageOpts, err := storage.GetResourceWithOptions(context, source)
		if err != nil {
			return nil, err
		}
		fs, err := storage.StorageService(context, source)
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
			exists, _ := fs.Exists(context.Background(), sourceURL, storageOpts...)
			if exists {
				source = url.NewResource(sourceURL, source.Credentials)
				break
			}
		}
	}
	transfer.Source = source
	if dest, err := context.ExpandResource(transfer.Dest); err == nil {
		transfer.Dest = dest
	}
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
	sessionID := exec.SessionID(context, target)
	operatingSystem := exec.OperatingSystem(context, sessionID)
	if operatingSystem != nil {
		osMap := data.NewMap()
		osMap.Put("System", operatingSystem.System)
		osMap.Put("Architecture", operatingSystem.Architecture)
		osMap.Put("Arch", operatingSystem.Arch)
		osMap.Put("Version", operatingSystem.Version)
		osMap.Put("Hardware", operatingSystem.Hardware)
		var state = context.State()
		state.Put("os", osMap)
	}
}

func (s *service) updateDeployState(context *endly.Context, target *url.Resource) {

	state := context.State()

	deploySetting := data.NewMap()
	state.Put("deploy", deploySetting)

	targetSettings := data.NewMap()
	targetSettings.Put("host", target.ParsedURL.Host)
	targetSettings.Put("URL", target.URL)
	targetSettings.Put("credentials", target.Credentials)
	deploySetting.Put("target", targetSettings)
}

func (s *service) deploy(context *endly.Context, request *Request) (*Response, error) {
	request = request.Expand(context)
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	s.updateDeployState(context, target)
	state := context.State()

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

	baseLocation := request.BaseLocation
	if baseLocation == "" {
		baseLocation = meta.BaseLocation
	}
	state.SetValue("deploy.baseLocation", baseLocation)
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
		runRequest := deploymentTarget.Deployment.Run.Clone(target)
		if err = endly.Run(context, runRequest, nil); err != nil {
			return nil, fmt.Errorf("failed to init deploy app to %v: %v", target, err)
		}
	}
	err = s.deployAddition(context, target, deploymentTarget.Deployment.Post)
	if err != nil {
		return nil, err
	}
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
