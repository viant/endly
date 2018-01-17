package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"sync"
)

const (
	//BuildServiceID represent build service id
	BuildServiceID = "build"

	//BuildServiceBuildAction represent build action
	BuildServiceBuildAction = "build"

	//BuildServiceLoadAction represent load build instruction action
	BuildServiceLoadAction = "load"
)

type buildService struct {
	*AbstractService
	mutex    *sync.RWMutex
	registry map[string]*BuildMeta
}

func (s *buildService) getMeta(context *Context, request *BuildRequest) (*BuildMeta, error) {
	s.mutex.RLock()
	result, hasMeta := s.registry[request.BuildSpec.Name]
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
				workflowResource, err := workflowService.Dao.NewRepoResource(state, fmt.Sprintf("meta/build/%v.json", request.BuildSpec.Name))
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
		response, err := s.loadMeta(context, &BuildLoadMetaRequest{
			Source: url.NewResource(metaURL, credential),
		})
		if err != nil {
			return nil, err
		}
		result = response.Meta
	}
	return result, nil
}

func (s *buildService) loadMeta(context *Context, request *BuildLoadMetaRequest) (*BuildLoadMetaResponse, error) {
	source, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	meta := &BuildMeta{}
	err = source.JSONDecode(meta)
	if err != nil {
		return nil, fmt.Errorf("unable to decode: %v, %v", source.URL, err)
	}

	meta.goalsIndex = make(map[string]*BuildGoal)
	indexBuildGoals(meta.Goals, meta.goalsIndex)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.registry[meta.Name] = meta
	return &BuildLoadMetaResponse{
		Meta: meta,
	}, nil
}

func (s *buildService) deployDependencyIfNeeded(context *Context, meta *BuildMeta, spec *BuildSpec, target *url.Resource) error {
	if len(meta.Dependencies) == 0 {
		return nil
	}
	deploymentService, err := context.Service(DeploymentServiceID)
	if err != nil {
		return err
	}
	for _, dependency := range meta.Dependencies {
		var app = context.Expand(dependency.Name)
		var version = context.Expand(dependency.Version)
		response := deploymentService.Run(context, &DeploymentDeployRequest{
			AppName: app,
			Version: version,
			Target:  target,
		})
		if response.Error != "" {
			return fmt.Errorf("failed to build %v, %v", spec.Name, response.Error)
		}
	}
	return nil
}

func indexBuildGoals(goals []*BuildGoal, index map[string]*BuildGoal) {
	if len(goals) == 0 {
		return
	}
	for _, goal := range goals {
		index[goal.Name] = goal
	}
}

func (s *buildService) setSdkIfNeeded(context *Context, request *BuildRequest) error {
	if request.BuildSpec.Sdk == "" {
		return nil
	}
	sdkService, err := context.Service(SdkServiceID)
	if err != nil {
		return err
	}
	serviceResponse := sdkService.Run(context, &SystemSdkSetRequest{
		Target:  request.Target,
		Sdk:     request.BuildSpec.Sdk,
		Version: request.BuildSpec.SdkVersion,
	})
	if serviceResponse.Error != "" {
		return errors.New(serviceResponse.Error)
	}
	return nil
}

func (s *buildService) build(context *Context, request *BuildRequest) (*BuildResponse, error) {
	var result = &BuildResponse{}
	state := context.State()
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	meta, err := s.getMeta(context, request)
	if err != nil {
		return nil, err
	}
	buildSpec := request.BuildSpec
	goal, has := meta.goalsIndex[buildSpec.Goal]
	if !has {
		return nil, fmt.Errorf("failed to lookup build %v goal: %v", buildSpec.Name, buildSpec.Goal)
	}

	buildState, err := newBuildState(buildSpec, target, request, context)
	if err != nil {
		return nil, err
	}
	state.Put("buildSpec", buildState)
	err = s.setSdkIfNeeded(context, request)
	if err != nil {
		return nil, err
	}

	err = s.deployDependencyIfNeeded(context, meta, buildSpec, target)
	if err != nil {
		return nil, err
	}
	if goal.InitTransfers != nil {
		_, err = context.Transfer(goal.InitTransfers.Transfers...)
		if err != nil {
			return nil, err
		}
	}

	if len(request.Credentials) > 0 {
		for _, execution := range goal.Command.Executions {
			if execution.MatchOutput != "" {
				execution.Credentials = request.Credentials
			}
		}
	}
	commandInfo, err := context.Execute(target, goal.Command)
	if err != nil {
		return nil, err
	}
	result.CommandInfo = commandInfo

	if goal.PostTransfers != nil {
		_, err = context.Transfer(goal.PostTransfers.Transfers...)
		if err != nil {
			return nil, err
		}
	}

	if goal.VerificationCommand != nil {
		_, err = context.Execute(target, goal.VerificationCommand)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func newBuildState(buildSepc *BuildSpec, target *url.Resource, request *BuildRequest, context *Context) (data.Map, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	build := data.NewMap()
	build.Put("name", buildSepc.Name)
	build.Put("version", buildSepc.Version)
	build.Put("args", buildSepc.Args)
	build.Put("goal", buildSepc.BuildGoal)
	build.Put("path", target.ParsedURL.Path)
	build.Put("host", target.ParsedURL.Host)
	build.Put("credential", target.Credential)
	build.Put("target", target)
	build.Put("sdk", buildSepc.Sdk)
	build.Put("sdkVersion", buildSepc.SdkVersion)
	return build, nil
}

func (s *buildService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err = s.Validate(request, response)
	if err != nil {
		return response
	}
	switch actualRequest := request.(type) {
	case *BuildRequest:
		response.Response, err = s.build(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to build: %v %v", actualRequest.Target.URL, err)
		}
	case *BuildLoadMetaRequest:
		response.Response, err = s.loadMeta(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to load build meta: %v %v", actualRequest.Source, err)
		}
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *buildService) NewRequest(action string) (interface{}, error) {
	switch action {
	case BuildServiceLoadAction:
		return &BuildLoadMetaRequest{}, nil
	case BuildServiceBuildAction:
		return &BuildRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)

}

//NewBuildService creates a new build service
func NewBuildService() Service {
	var result = &buildService{
		registry: make(map[string]*BuildMeta),
		mutex:    &sync.RWMutex{},
		AbstractService: NewAbstractService(BuildServiceID,
			BuildServiceBuildAction,
			BuildServiceLoadAction),
	}
	result.AbstractService.Service = result
	return result
}
