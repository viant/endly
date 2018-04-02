package vc

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox"

	"github.com/viant/toolbox/url"
	"path"
)

const (
	//ServiceID version control service id
	ServiceID = "version/control"
	//CredentialKey represents credentials key
	CredentialKey = "***vc***"
)

type service struct {
	*endly.AbstractService
	*git
	*svnService
}

//checkInfo returns version control info
func (s *service) checkInfo(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	source, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.git.checkInfo(context, request)
	case "svn":
		return s.svnService.checkInfo(context, request)
	case "local":
		return &StatusResponse{
			Info: &Info{
				Origin: request.Source.URL,
			},
		}, nil
	}
	return nil, fmt.Errorf("unsupported vc type: %v for URL %v", request.Type, source.URL)
}

//commit commits local changes to the version control
func (s *service) commit(context *endly.Context, request *CommitRequest) (*CommitResponse, error) {
	target, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}

	if err = endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.DirectoryPath())), nil); err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.git.commit(context, request)
	case "svn":
		return s.svnService.commit(context, request)

	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", request.Type, target.URL)
}

//pull retrieves the latest changes from the origin
func (s *service) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	target, err := context.ExpandResource(request.Dest)
	if err != nil {
		return nil, err
	}
	if err = endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.DirectoryPath())), nil); err != nil {
		return nil, err
	}
	switch request.Type {
	case "git":
		return s.git.pull(context, request)
	case "svn":
		return s.svnService.pull(context, request)
	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", request.Type, target.URL)
}

//checkout If target directory exist and already contains matching origin URL, only taking the latest changes without overriding local if performed, otherwise full checkout
func (s *service) checkout(context *endly.Context, request *CheckoutRequest) (*CheckoutResponse, error) {
	var response = &CheckoutResponse{
		Checkouts: make(map[string]*Info),
	}
	target, err := context.ExpandResource(request.Dest)
	if err != nil {
		return nil, err
	}

	var modules = request.Modules
	var directory = target.DirectoryPath()
	if len(modules) == 0 {
		modules = append(modules, "")
		parent, _ := path.Split(target.DirectoryPath())
		directory = parent
	}
	if directory != "/" {
		if err = endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("mkdir -p %v", directory)), nil); err != nil {
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

func (s *service) checkoutArtifact(context *endly.Context, versionControlType string, origin, dest *url.Resource, removeLocalChanges bool) (info *Info, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to checkout %v, %v", origin.URL, err)
		}
	}()

	var directoryPath = dest.DirectoryPath()
	storageService, err := storage.GetStorageService(context, dest)
	if err != nil {
		return nil, err
	}
	exists, err := storageService.Exists(dest.URL)
	if err != nil {
		return nil, err
	}

	if exists {
		var response *StatusResponse
		response, err = s.checkInfo(context, &StatusRequest{Source: dest, Type: versionControlType})
		if err != nil {
			return nil, err
		}
		if response.IsVersionControlManaged {
			var originURLResource = url.NewResource(origin.URL)
			var actualURLResource = url.NewResource(response.Origin)
			originPath := originURLResource.ParsedURL.Hostname() + originURLResource.DirectoryPath()
			actualPath := actualURLResource.ParsedURL.Hostname() + actualURLResource.DirectoryPath()
			if originPath == actualPath {
				_, err = s.pull(context, &PullRequest{
					Type:   versionControlType,
					Origin: origin,
					Dest:   dest,
				})
				if err != nil {
					return nil, err
				}
				return response.Info, nil
			}

			if removeLocalChanges {
				if err = endly.Run(context, exec.NewRunRequest(dest, false, fmt.Sprintf("rm -rf %v", directoryPath)), nil); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("directory contains incompatible repo: %v %v", response.Origin, origin.URL)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	switch versionControlType {
	case "git":
		info, err = s.git.checkout(context, &CheckoutRequest{
			Origin: origin,
			Dest:   dest,
		})
	case "svn":
		info, err = s.svnService.checkout(context, &CheckoutRequest{
			Origin: origin,
			Dest:   dest,
		})
	case "local":
		err = endly.Run(context, storage.NewCopyRequest(nil, storage.NewTransfer(origin, dest, true, false, nil)), nil)
		info = &Info{Origin: origin.URL}
	default:
		err = fmt.Errorf("unsupported version control type: '%v'", versionControlType)
	}
	return info, err
}

const (
	vcExplicitVersionCheckExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/myproject/trunk",
						"Credentials":"${env.HOME}/.secret/localhost.json"
					},
					"Type":"svn"
}`
	vcImplicitVersionCheckExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/git/myproject/trunk",
						"Credentials":"${env.HOME}/.secret/localhost.json"
					}

}`
	vcSingleProjectCheckoutExample = `{
  "Target":{
    "URL":"ssh://127.0.0.1/Projects/go/",
    "Credentials":"${env.HOME}/.secret/localhost.json"
  },
  "Origin": {
      "URL":"https://github.com/viant/endly/"
  }
}
`
	vcMultiProjectCheckoutExample = `{
  "Target":{
    "URL":"ssh://127.0.0.1/Projects/go/",
    "Credentials":"${env.HOME}/.secret/localhost.json"
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
    "Credentials":"${env.HOME}/.secret/localhost.json"
  },
  "Type":"svn",
  "Styled":"my comments"
}`
	vcPullExample = `{
					"Target":{
						"URL":"ssh://127.0.0.1/Projects/go/",
						"Credentials":"${env.HOME}/.secret/localhost.json"
					},
					"Origin": {
						"URL":"https://github.com/viant/endly/"
						"Credentials":"${env.HOME}/.secret/git.json"
					}
				}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "status",
		RequestInfo: &endly.ActionInfo{
			Description: "check status of version control on supplied target URL host and path",
			Examples: []*endly.UseCase{
				{
					Description: "Explicit version control type",
					Data:        vcExplicitVersionCheckExample,
				},
				{
					Description: "Implicit version control type derived from URL",
					Data:        vcImplicitVersionCheckExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &StatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StatusRequest); ok {
				return s.checkInfo(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "checkout",
		RequestInfo: &endly.ActionInfo{
			Description: `pull orign code to destination defined by target resource. 
If target directory exist and contains matching origin URL, only latest changes without overriding local are sync, otherwise full checkout`,
			Examples: []*endly.UseCase{
				{
					Description: "single project checkout",
					Data:        vcSingleProjectCheckoutExample,
				},
				{
					Description: "multi projects checkout",
					Data:        vcMultiProjectCheckoutExample,
				},
			},
		},
		ResponseInfo: &endly.ActionInfo{
			Description: "returns key value pairs of origin url with corresponding info ",
			Examples: []*endly.UseCase{
				{
					Description: "multi project checkout",
					Data:        vcMultiProjectCheckoutResponseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CheckoutRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CheckoutResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CheckoutRequest); ok {
				return s.checkout(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "commit",
		RequestInfo: &endly.ActionInfo{
			Description: "submit code changes to version control origin",
			Examples: []*endly.UseCase{
				{
					Description: "",
					Data:        vcCommitExample,
				}},
		},
		RequestProvider: func() interface{} {
			return &CommitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CommitResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CommitRequest); ok {
				return s.commit(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "pull",
		RequestInfo: &endly.ActionInfo{
			Description: "",
			Examples: []*endly.UseCase{
				{
					Description: "",
					Data:        vcPullExample,
				}},
		},
		RequestProvider: func() interface{} {
			return &PullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PullRequest); ok {
				return s.pull(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new version control service (git,svn)
func New() endly.Service {
	var service = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		git:             &git{},
		svnService:      &svnService{},
	}
	service.AbstractService.Service = service
	service.registerRoutes()
	return service
}
