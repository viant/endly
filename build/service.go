package build

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"net/url"
)

//Add global request with map context parameters

const BuildServiceId = "build"

type OperatingSystemDeployment struct {
	OsTarget *endly.OperatingSystemTarget
	Config   *endly.DeploymentConfig
}

type Goal struct {
	Name                string
	Command             *endly.ManagedCommand
	Transfers           *endly.TransfersRequest
	VerificationCommand *endly.ManagedCommand
}

type Meta struct {
	Name             string
	Goals            []*Goal
	goalsIndex       map[string]*Goal
	BuildDeployments []*OperatingSystemDeployment //defines deployment of the build app itself, i.e how to get maven installed
}

func (m *Meta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("MetaBuild.Name %v", m.Name)

	}
	if len(m.Goals) == 0 {
		return fmt.Errorf("MetaBuild.Goals were empty %v", m.Name)
	}
	return nil
}

func (m *Meta) Match(operatingSystem *endly.OperatingSystem, version string) *OperatingSystemDeployment {
	for _, candidate := range m.BuildDeployments {
		osTarget := candidate.OsTarget
		if version != "" {
			if candidate.Config.Transfer.Target.Version != version {
				continue
			}
		}
		if operatingSystem.Matches(osTarget) {
			return candidate
		}
	}
	return nil
}

type BuildMetaRegistry map[string]*Meta

func indexBuildGoals(goals []*Goal, index map[string]*Goal) {
	if len(goals) == 0 {
		return
	}
	for _, goal := range goals {
		index[goal.Name] = goal
		fmt.Printf("!!%v\n", goal.Name)
	}
}

func (r *BuildMetaRegistry) Register(meta *Meta) {
	meta.goalsIndex = make(map[string]*Goal)
	indexBuildGoals(meta.Goals, meta.goalsIndex)
	(*r)[meta.Name] = meta
}

type BuildSpec struct {
	Name    string //build name  like go, mvn, node, yarn
	Version string
	Goal    string //actual build target, like clean, test
	Args    string // additional build arguments , that can be expanded with $build.args
}

type BuildRequest struct {
	BuildSpec *BuildSpec      //build specification
	Target    *endly.Resource //path to application to be build, Note that command may use $build.target variable. that expands to Target URL path
}

type buildService struct {
	*endly.AbstractService
	registry BuildMetaRegistry
}

func (s *buildService) build(context *endly.Context, request *BuildRequest) (interface{}, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	buildSepc := request.BuildSpec

	if buildSepc == nil {
		return nil, fmt.Errorf("BuildSepc was empty")
	}
	buildMeta, has := s.registry[buildSepc.Name]
	if !has {
		return nil, fmt.Errorf("Failed to lookup build: %v", buildSepc.Name)
	}

	goal, has := buildMeta.goalsIndex[buildSepc.Goal]
	if !has {
		return nil, fmt.Errorf("Failed to lookup build %v goal: %v", buildSepc.Name, buildSepc.Goal)
	}

	parsedUrl, err := url.Parse(target.URL)
	if err != nil {
		return nil, err
	}

	err = setBuildState(buildSepc, parsedUrl, request, context)
	if err != nil {
		return nil, err
	}
	execService, err := context.Service(endly.ExecServiceId)
	if err != nil {
		return nil, err
	}
	response := execService.Run(context, &endly.OpenSession{
		Target: target,
	})

	if response.Error != nil {
		return nil, response.Error
	}

	operatingSystem := context.OperatingSystem(target.Session())
	buildDeployment := buildMeta.Match(operatingSystem, buildSepc.Version)
	if buildDeployment == nil {
		return nil, fmt.Errorf("Failed to find a build for provided operating system: %v %v", operatingSystem.Name, operatingSystem.Version)
	}

	deploymentService, err := context.Service(endly.DeploymentServiceId)

	if err != nil {
		return nil, err
	}

	fmt.Printf("buildDeployment: %v\n", buildDeployment)

	response = deploymentService.Run(context, buildDeployment.Config)
	if response.Error != nil {
		return nil, response.Error

	}

	_, err = context.Execute(target, goal.Command)
	if err != nil {
		return nil, err
	}

	if goal.Transfers != nil {
		_, err = context.Transfer(goal.Transfers.Transfers...)
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
	return nil, nil
}
func setBuildState(buildSepc *BuildSpec, parsedUrl *url.URL, request *BuildRequest, context *endly.Context) error {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return err
	}
	build := common.NewMap()
	build.Put("args", buildSepc.Args)
	build.Put("target", parsedUrl.Path)
	build.Put("host", parsedUrl.Host)
	build.Put("credential", target.Credential)
	var state = context.State()
	state.Put("build", build)
	return nil
}

func (s *buildService) Run(context *endly.Context, request interface{}) *endly.Response {
	var response = &endly.Response{
		Status: "ok",
	}
	switch castedRequest := request.(type) {
	case *BuildRequest:
		response.Response, response.Error = s.build(context, castedRequest)
	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}
	if response.Error != nil {
		response.Status = "error"
	}
	return response
}

func (t *buildService) NewRequest(name string) (interface{}, error) {
	return &BuildRequest{}, nil
}

func loadRegistry(config *Config) (BuildMetaRegistry, error) {
	var result BuildMetaRegistry = make(map[string]*Meta)
	if len(config.URL) > 0 {
		for _, url := range config.URL {
			service, err := storage.NewServiceForURL(url, "")
			if err != nil {
				return nil, err
			}
			objects, err := service.List(url)
			if err != nil {
				return nil, err
			}
			for _, object := range objects {
				reader, err := service.Download(object)
				if err != nil {
					return nil, err
				}
				var buildMeta = &Meta{}

				err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(buildMeta)
				if err != nil {
					return nil, err
				}
				err = buildMeta.Validate()
				if err != nil {
					return nil, err
				}
				result.Register(buildMeta)
			}
		}
	}
	return result, nil
}

func NewBuildService(config *Config) (endly.Service, error) {
	registry, err := loadRegistry(config)
	if err != nil {
		return nil, err
	}
	var result = &buildService{
		registry:        registry,
		AbstractService: endly.NewAbstractService(BuildServiceId),
	}
	result.AbstractService.Service = result
	return result, nil
}
