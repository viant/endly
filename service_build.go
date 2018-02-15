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
		if response.err != nil {
			return err
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
		Env:     request.Env,
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



const (
	buildGoBuildExample = `{
	"BuildSpec": {
		"Name": "go",
		"Goal": "build",
		"BuildGoal": "build",
		"Args": " -o echo",
		"Sdk": "go",
		"SdkVersion": "1.8"
	},
	"Env": {
		"GOOS": "linux"
	},
	"Target": {
		"URL": "scp://127.0.0.1/tmp/app/echo",
		"Credential": "${env.HOME}/.secret/localhost.json"
	}
}
`
	buildJavaBuildExample = `{
  "BuildSpec": {
    "Name": "maven",
    "Version": "3.5",
    "Goal": "build",
    "BuildGoal": "install",
    "Args": " -f pom.xml -am -pl server -DskipTest",
    "Sdk": "jdk",
    "SdkVersion": "1.7"
  },
 "Target": {
    "URL": "scp://127.0.0.1/tmp/app/server/",
    "Credential": "${env.HOME}/.secret/scp.json"
  }
}
`
)




func (s *buildService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "build",
		RequestInfo: &ActionInfo{
			Description: "build app with supplied specification",
			Examples: []*ExampleUseCase{
				{
					UseCase: "go app build",
					Data: buildGoBuildExample,
				},
				{
					UseCase: "java app build",
					Data:buildJavaBuildExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &BuildRequest{}
		},
		ResponseProvider: func() interface{} {
			return &BuildResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*BuildRequest); ok {
				return s.build(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "load",
		RequestInfo: &ActionInfo{
			Description: "load build meta instruction",
		},
		RequestProvider: func() interface{} {
			return &BuildLoadMetaRequest{}
		},
		ResponseProvider: func() interface{} {
			return &BuildLoadMetaResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*BuildLoadMetaRequest); ok {
				return s.loadMeta(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewBuildService creates a new build service
func NewBuildService() Service {
	var result = &buildService{
		registry:        make(map[string]*BuildMeta),
		mutex:           &sync.RWMutex{},
		AbstractService: NewAbstractService(BuildServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
