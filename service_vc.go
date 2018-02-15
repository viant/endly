package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"path"
)

const (
	//VersionControlServiceID version control service id
	VersionControlServiceID = "version/control"
	//versionControlCredentialKey represents credential key
	versionControlCredentialKey = "***vc***"
)

type versionControlService struct {
	*AbstractService
	*gitService
	*svnService
}

//checkInfo returns version control info
func (s *versionControlService) checkInfo(context *Context, request *VcStatusRequest) (*VcStatusResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.gitService.checkInfo(context, request)
	case "svn":
		return s.svnService.checkInfo(context, request)
	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", request.Type, target.URL)
}

//commit commits local changes to the version control
func (s *versionControlService) commit(context *Context, request *VcCommitRequest) (*VcCommitResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd %v", target.DirectoryPath()),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.gitService.commit(context, request)
	case "svn":
		return s.svnService.commit(context, request)

	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", request.Type, target.URL)
}

//pull retrieves the latest changes from the origin
func (s *versionControlService) pull(context *Context, request *VcPullRequest) (*VcPullResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd %v", target.DirectoryPath()),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.gitService.pull(context, request)
	case "svn":
		return s.svnService.pull(context, request)
	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", request.Type, target.URL)
}

//checkout If target directory exist and already contains matching origin URL, only taking the latest changes without overriding local if performed, otherwise full checkout
func (s *versionControlService) checkout(context *Context, request *VcCheckoutRequest) (*VcCheckoutResponse, error) {
	var response = &VcCheckoutResponse{
		Checkouts: make(map[string]*VcInfo),
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var modules = request.Modules
	if len(modules) == 0 {
		modules = append(modules, "")
		parent, _ := path.Split(target.DirectoryPath())
		_, err = context.Execute(target, &ExtractableCommand{
			Executions: []*Execution{
				{
					Command: fmt.Sprintf("mkdir -p %v", parent),
				},
			},
		})

	} else {
		_, err = context.Execute(target, &ExtractableCommand{
			Executions: []*Execution{
				{
					Command: fmt.Sprintf("mkdir -p %v", target.DirectoryPath()),
				},
			},
		})
		if err != nil {
			return nil, err
		}
	}

	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}

	for _, module := range modules {
		var moduleOrigin = origin.Clone()
		var targetModule = target.Clone()
		if module != "" {
			moduleOrigin.URL = toolbox.URLPathJoin(origin.URL, module)
			targetModule.URL = toolbox.URLPathJoin(target.URL, module)
		}
		info, err := s.checkoutArtifact(context, request.Type, moduleOrigin, targetModule, request.RemoveLocalChanges)
		if err != nil {
			return nil, err
		}
		response.Checkouts[moduleOrigin.URL] = info
	}
	return response, nil
}

func (s *versionControlService) checkoutArtifact(context *Context, versionControlType string, origin, target *url.Resource, removeLocalChanges bool) (info *VcInfo, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to checkout %v, %v", origin.URL, err)
		}
	}()
	var directoryPath = target.DirectoryPath()
	storageService, err := storage.NewServiceForURL(target.URL, target.Credential)
	if err != nil {
		return nil, err
	}
	exists, err := storageService.Exists(target.URL)
	if err != nil {
		return nil, err
	}
	if exists {
		var response *VcStatusResponse
		response, err = s.checkInfo(context, &VcStatusRequest{Target: target, Type: versionControlType})
		if err != nil {
			return nil, err
		}
		var originURLResource = url.NewResource(origin.URL)
		var actualURLResource = url.NewResource(response.Origin)
		originPath := originURLResource.ParsedURL.Hostname() + originURLResource.DirectoryPath()
		actualPath := actualURLResource.ParsedURL.Hostname() + actualURLResource.DirectoryPath()
		if originPath == actualPath {
			_, err = s.pull(context, &VcPullRequest{
				Type:   versionControlType,
				Origin: origin,
				Target: target,
			})
			if err != nil {
				return nil, err
			}
			return response.VcInfo, nil
		}

		if removeLocalChanges {
			_, err = context.Execute(target, &ExtractableCommand{
				Executions: []*Execution{
					{
						Command: fmt.Sprintf("rm -rf %v", directoryPath),
					},
				},
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("directory contains incompatible repo: %v %v", response.Origin, origin.URL)
		}
	}
	if err != nil {
		return nil, err
	}
	switch versionControlType {
	case "git":
		info, err = s.gitService.checkout(context, &VcCheckoutRequest{
			Origin: origin,
			Target: target,
		})
	case "svn":
		info, err = s.svnService.checkout(context, &VcCheckoutRequest{
			Origin: origin,
			Target: target,
		})

	default:
		err = fmt.Errorf("unsupported version control type: '%v'", versionControlType)
	}
	return info, err
}

const (
	vcExplicitVersionCheckExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/myproject/trunk",
						"Credential":"${env.HOME}/.secret/localhost.json"
					},
					"Type":"svn"
}`
	vcImplicitVersionCheckExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/git/myproject/trunk",
						"Credential":"${env.HOME}/.secret/localhost.json"
					}

}`
	vcSingleProjectCheckoutExample = `{
  "Target":{
    "URL":"ssh://127.0.0.1/Projects/go/",
    "Credential":"${env.HOME}/.secret/localhost.json"
  },
  "Origin": {
      "URL":"https://github.com/viant/endly/"
  }
}
`
	vcMultiProjectCheckoutExample = `{
  "Target":{
    "URL":"ssh://127.0.0.1/Projects/go/",
    "Credential":"${env.HOME}/.secret/localhost.json"
  },
  "Origin": {
    "URL":"https://github.com/viant/"
  },
  "Modules":["toolbox", "assertly", "endly"]
}`

	vcMultiProjectCheckoutResponseExample = `{
			"Checkouts": {
			"https://github.com/adrianwit/echo": {
				"IsVersionControlManaged": true,
				"Origin": "https://github.com:443/adrianwit/echo",
				"Revision": "7f98e433333bc1961135d4ec9023aa95134198fd",
				"Branch": "master",
				"IsUptoDate": true,
				"New": [],
				"Untracked": [],
				"Modified": [],
				"Deleted": []
			},
			"https://github.com/adrianwit/neatly-introduction": {
				"IsVersionControlManaged": true,
				"Origin": "https://github.com:443/adrianwit/neatly-introduction",
				"Revision": "f194db0d9f7574b424e9820b423d2357da4775f8",
				"Branch": "master",
				"IsUptoDate": true,
				"New": [],
				"Untracked": [],
				"Modified": [],
				"Deleted": []
			}
		}
	}`
	vcCommitExample = `{
  "Target":{
    "URL":"ssh://127.0.0.1/Projects/myproject/trunk",
    "Credential":"${env.HOME}/.secret/localhost.json"
  },
  "Type":"svn",
  "Message":"my comments"
}`
	vcPullExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/go/",
						"Credential":"${env.HOME}/.secret/localhost.json"
					},
					"Origin": {
						"URL":"https://github.com/viant/endly/"
						"Credential":"${env.HOME}/.secret/git.json"
					}
				}`
)

func (s *versionControlService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "status",
		RequestInfo: &ActionInfo{
			Description: "check status of version control on supplied target URL host and path",
			Examples: []*ExampleUseCase{
				{
					UseCase: "Explicit version control type",
					Data:    vcExplicitVersionCheckExample,
				},
				{
					UseCase: "Implicit version control type derived from URL",
					Data:    vcImplicitVersionCheckExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &VcStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &VcStatusResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*VcStatusRequest); ok {
				return s.checkInfo(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "checkout",
		RequestInfo: &ActionInfo{
			Description: `pull orign code to destination defined by target resource. 
If target directory exist and contains matching origin URL, only latest changes without overriding local are sync, otherwise full checkout`,
			Examples: []*ExampleUseCase{
				{
					UseCase: "single project checkout",
					Data:    vcSingleProjectCheckoutExample,
				},
				{
					UseCase: "multi projects checkout",
					Data:    vcMultiProjectCheckoutExample,
				},
			},
		},
		ResponseInfo: &ActionInfo{
			Description: "returns key value pairs of origin url with corresponding info ",
			Examples: []*ExampleUseCase{
				{
					UseCase: "multi project checkout",
					Data:    vcMultiProjectCheckoutResponseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &VcCheckoutRequest{}
		},
		ResponseProvider: func() interface{} {
			return &VcCheckoutResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*VcCheckoutRequest); ok {
				return s.checkout(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "commit",
		RequestInfo: &ActionInfo{
			Description: "submit code changes to version control origin",
			Examples: []*ExampleUseCase{
				{
					UseCase: "",
					Data:    vcCommitExample,
				}},
		},
		RequestProvider: func() interface{} {
			return &VcCommitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &VcCommitResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*VcCommitRequest); ok {
				return s.commit(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&ServiceActionRoute{
		Action: "pull",
		RequestInfo: &ActionInfo{
			Description: "",
			Examples: []*ExampleUseCase{
				{
					UseCase: "",
					Data:    vcPullExample,
				}},
		},
		RequestProvider: func() interface{} {
			return &VcPullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &VcPullResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*VcPullRequest); ok {
				return s.pull(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewVersionControlService creates a new version control
func NewVersionControlService() Service {
	var service = &versionControlService{
		AbstractService: NewAbstractService(VersionControlServiceID),
		gitService:      &gitService{},
		svnService:      &svnService{},
	}
	service.AbstractService.Service = service
	service.registerRoutes()
	return service
}
