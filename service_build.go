package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
)

//BuildServiceID represent build service id
const BuildServiceID = "build"

type buildService struct {
	*AbstractService
	registry BuildMetaRegistry
}

func (s *buildService) loadBuildMeta(context *Context, buildMetaURL string) error {
	if buildMetaURL == "" {
		return fmt.Errorf("buildMeta was empty")
	}
	resource := url.NewResource(buildMetaURL)
	meta := &BuildMeta{}
	err := resource.JSONDecode(meta)
	if err != nil {
		return err
	}
	return s.registry.Register(meta)
}

func (s *buildService) deployAppBuilderIfNeeded(context *Context, meta *BuildMeta, spec *BuildSpec, target *url.Resource) error {
	if len(meta.BuildDeployments) == 0 {
		return nil
	}
	operatingSystem := context.OperatingSystem(target.Host())

	buildDeployment := meta.Match(operatingSystem, spec.Version)
	if buildDeployment == nil {
		return fmt.Errorf("Failed to find a build for provided operating system: %v %v", operatingSystem.Name, operatingSystem.Version)
	}

	deploymentService, err := context.Service(DeploymentServiceID)

	if err != nil {
		return err
	}

	response := deploymentService.Run(context, buildDeployment.Deploy)
	if response.Error != "" {
		return errors.New(response.Error)

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
	buildSpec := request.BuildSpec
	_, hasMeta := s.registry[buildSpec.Name]
	if !hasMeta {
		var buildMetaURL = request.BuildMetaURL
		if buildMetaURL == "" {
			service, err := context.Service(WorkflowServiceID)
			if err != nil {
				return nil, err
			}
			if workflowService, ok := service.(*workflowService); ok {
				workflowResource, err := workflowService.Dao.NewRepoResource(state, fmt.Sprintf("build/meta/%v.json", buildSpec.Name))
				if err != nil {
					return nil, err
				}
				buildMetaURL = workflowResource.URL

			}
		}
		err = s.loadBuildMeta(context, buildMetaURL)
		if err != nil {
			return nil, err
		}
	}

	if buildSpec == nil {
		return nil, fmt.Errorf("BuildSpec was empty")
	}
	buildMeta, has := s.registry[buildSpec.Name]
	if !has {
		return nil, fmt.Errorf("Failed to lookup build: %v", buildSpec.Name)
	}

	goal, has := buildMeta.goalsIndex[buildSpec.Goal]
	if !has {
		return nil, fmt.Errorf("Failed to lookup build %v goal: %v", buildSpec.Name, buildSpec.Goal)
	}

	buildState, err := newBuildState(buildSpec, target, request, context)
	if err != nil {
		return nil, err
	}
	if buildMeta.Sdk != "" {
		state.Put("build", buildState)
		sdkService, err := context.Service(SdkServiceID)
		if err != nil {
			return nil, err
		}
		serviceResponse := sdkService.Run(context, &SystemSdkSetRequest{Target: request.Target,
			Sdk: context.Expand(buildMeta.Sdk),
			Version: context.Expand(buildMeta.SdkVersion),
		})
		if serviceResponse.Error != "" {
			return nil, errors.New(serviceResponse.Error)
		}
		result.SdkResponse, _ = serviceResponse.Response.(*SystemSdkSetResponse)
	}

	execService, err := context.Service(ExecServiceID)
	if err != nil {
		return nil, err
	}
	state.Put("build", buildState)
	response := execService.Run(context, &OpenSessionRequest{
		Target: target,
	})

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	s.deployAppBuilderIfNeeded(context, buildMeta, buildSpec, target)
	commandInfo, err := context.Execute(target, goal.Command)
	if err != nil {
		return nil, err
	}

	if goal.InitTransfers != nil {
		_, err = context.Transfer(goal.InitTransfers.Transfers...)
		if err != nil {
			return nil, err
		}
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
	build.Put("args", buildSepc.Args)
	build.Put("goal", buildSepc.BuildGoal)
	build.Put("target", target.ParsedURL.Path)
	build.Put("host", target.ParsedURL.Host)
	build.Put("credential", target.Credential)
	build.Put("sdk", buildSepc.Sdk)
	build.Put("sdkVersion", buildSepc.SdkVersion)
	return build, nil
}

func (s *buildService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *BuildRequest:
		response.Response, err = s.build(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to build: %v %v", actualRequest.Target.URL, err)
		}
	case *BuildRegisterMetaRequest:
		err = s.registry.Register(actualRequest.Meta)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to register: %v", actualRequest.Meta.Name, err)
		}
		response.Response = &BuildRegisterMetaResponse{
			Name: actualRequest.Meta.Name,
		}

	case *BuildLoadMetaRequest:
		s.load(context, actualRequest)

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *buildService) load(context *Context, request *BuildLoadMetaRequest) (*BuildLoadMetaResponse, error) {
	var result = &BuildLoadMetaResponse{
		Loaded: make(map[string]*BuildMeta),
	}
	target, err := context.ExpandResource(request.Resource)
	if err != nil {
		return nil, err
	}

	service, err := storage.NewServiceForURL(target.URL, "")
	if err != nil {
		return nil, err
	}
	objects, err := service.List(target.URL)
	if err != nil {
		return nil, err
	}

	for _, object := range objects {
		reader, err := service.Download(object)
		if err != nil {
			return nil, err
		}
		var buildMeta = &BuildMeta{}
		err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(buildMeta)
		if err != nil {
			return nil, err
		}
		err = s.registry.Register(buildMeta)
		if err != nil {
			return nil, err
		}
		result.Loaded[object.URL()] = buildMeta
	}
	return result, nil
}

func (s *buildService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "load":
		return &BuildLoadMetaRequest{}, nil
	case "register":
		return &BuildRegisterMetaRequest{}, nil
	case "build":
		return &BuildRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)

}

//NewBuildService creates a new build service
func NewBuildService() Service {
	var result = &buildService{
		registry:        make(map[string]*BuildMeta),
		AbstractService: NewAbstractService(BuildServiceID),
	}
	result.AbstractService.Service = result
	return result
}

//BuildMetaRegistry represents a build meta registry
type BuildMetaRegistry map[string]*BuildMeta

func indexBuildGoals(goals []*BuildGoal, index map[string]*BuildGoal) {
	if len(goals) == 0 {
		return
	}
	for _, goal := range goals {
		index[goal.Name] = goal
	}
}

//Register register build meta into registry, if passes validation
func (r *BuildMetaRegistry) Register(meta *BuildMeta) error {
	err := meta.Validate()
	if err != nil {
		return nil
	}
	meta.goalsIndex = make(map[string]*BuildGoal)
	indexBuildGoals(meta.Goals, meta.goalsIndex)
	(*r)[meta.Name] = meta
	return nil
}
