package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
)


const BuildServiceId = "build"

type OperatingSystemDeployment struct {
	OsTarget *OperatingSystemTarget
	Deploy   *DeploymentDeployRequest
}

type BuildGoal struct {
	Name                string
	InitTransfers       *TransferCopyRequest
	Command             *ManagedCommand
	PostTransfers       *TransferCopyRequest
	VerificationCommand *ManagedCommand
}

type BuildMeta struct {
	Sdk              string
	SdkVersion       string
	Name             string
	Goals            []*BuildGoal
	goalsIndex       map[string]*BuildGoal
	BuildDeployments []*OperatingSystemDeployment //defines deployment of the build app itself, i.e how to get maven installed
}

func (m *BuildMeta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("MetaBuild.Names %v", m.Name)

	}
	if len(m.Goals) == 0 {
		return fmt.Errorf("MetaBuild.Goals were empty %v", m.Name)
	}
	return nil
}

func (m *BuildMeta) Match(operatingSystem *OperatingSystem, version string) *OperatingSystemDeployment {
	for _, candidate := range m.BuildDeployments {
		osTarget := candidate.OsTarget
		if version != "" {
			if candidate.Deploy.Transfer.Target.Version != version {
				continue
			}
		}
		if operatingSystem.Matches(osTarget) {
			return candidate
		}
	}
	return nil
}

type BuildSpec struct {
	Name       string //build name  like go, mvn, node, yarn
	Version    string
	Goal       string //lookup for BuildMeta goal
	BuildGoal  string //actual build target, like clean, test
	Args       string // additional build arguments , that can be expanded with $build.args
	Sdk        string
	SdkVersion string
}

type BuildRequest struct {
	BuildMetaURL string
	BuildSpec    *BuildSpec //build specification
	Target       *url.Resource  //path to application to be build, Note that command may use $build.target variable. that expands to Target URL path
}

type BuildResponse struct {
	SdkResponse *SdkSetResponse
	CommandInfo *CommandInfo
}

type BuildRegisterMetaRequest struct {
	Meta *BuildMeta
}

type BuildLoadMetaRequest struct {
	Resource *url.Resource
}

type BuildLoadMetaResponse struct {
	Loaded map[string]*BuildMeta //url to size
}

type BuildService struct {
	*AbstractService
	registry BuildMetaRegistry
}

func (s *BuildService) loadBuildMeta(context *Context, buildMetaURL string) error {
	if buildMetaURL == "" {
		return fmt.Errorf("buildMeta was empty")
	}
	resource := url.NewResource(buildMetaURL)
	meta := &BuildMeta{}
	err := resource.JsonDecode(meta)
	if err != nil {
		return err
	}
	return s.registry.Register(meta)
}

func (s *BuildService) build(context *Context, request *BuildRequest) (interface{}, error) {
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
			service, err := context.Service(WorkflowServiceId)
			if err != nil {
				return nil, err
			}
			if workflowService, ok := service.(*WorkflowService); ok {
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
		sdkService, err := context.Service(SdkServiceId)
		if err != nil {
			return nil, err
		}
		serviceResponse := sdkService.Run(context, &SdkSetRequest{Target: request.Target,
			Sdk:     context.Expand(buildMeta.Sdk),
			Version: context.Expand(buildMeta.SdkVersion),
		})
		if serviceResponse.Error != "" {
			return nil, errors.New(serviceResponse.Error)
		}
		result.SdkResponse, _ = serviceResponse.Response.(*SdkSetResponse)
	}

	execService, err := context.Service(ExecServiceId)
	if err != nil {
		return nil, err
	}
	state.Put("build", buildState)
	response := execService.Run(context, &OpenSession{
		Target: target,
	})

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	operatingSystem := context.OperatingSystem(target.Host())
	buildDeployment := buildMeta.Match(operatingSystem, buildSpec.Version)
	if buildDeployment == nil {
		return nil, fmt.Errorf("Failed to find a build for provided operating system: %v %v", operatingSystem.Name, operatingSystem.Version)
	}

	deploymentService, err := context.Service(DeploymentServiceId)

	if err != nil {
		return nil, err
	}

	response = deploymentService.Run(context, buildDeployment.Deploy)
	if response.Error != "" {
		return nil, errors.New(response.Error)

	}

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

func (s *BuildService) Run(context *Context, request interface{}) *ServiceResponse {
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

func (s *BuildService) load(context *Context, request *BuildLoadMetaRequest) (*BuildLoadMetaResponse, error) {
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

func (s *BuildService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "build":
		return &BuildRequest{}, nil
	case "load":
		return &BuildLoadMetaRequest{}, nil
	case "register":
		return &BuildRegisterMetaRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)

}

func NewBuildService() Service {
	var result = &BuildService{
		registry:        make(map[string]*BuildMeta),
		AbstractService: NewAbstractService(BuildServiceId),
	}
	result.AbstractService.Service = result
	return result
}

type BuildMetaRegistry map[string]*BuildMeta

func indexBuildGoals(goals []*BuildGoal, index map[string]*BuildGoal) {
	if len(goals) == 0 {
		return
	}
	for _, goal := range goals {
		index[goal.Name] = goal
	}
}

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
