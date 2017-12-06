package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"errors"
)

const versionControlCredentailKey = "***vc***"

//VersionControlServiceID version control service id
var VersionControlServiceID = "version/control"

type versionControlService struct {
	*AbstractService
	*gitService
	*svnService
}

//checkInfo returns version control info
func (s *versionControlService) checkInfo(context *Context, request *VcStatusRequest) (*VcInfo, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	switch target.Type {
	case "git":
		return s.gitService.checkInfo(context, request)
	case "svn":
		return s.svnService.checkInfo(context, request)
	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", target.Type, target.URL)
}

//commit commits local changes to the version control
func (s *versionControlService) commit(context *Context, request *VcCommitRequest) (interface{}, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd  %v", target.DirectoryPath()),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	switch target.Type {
	case "git":
		return s.gitService.commit(context, request)
	case "svn":
		return s.svnService.commit(context, request)

	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", target.Type, target.URL)
}

//pull retrieves the latest changes from the origin
func (s *versionControlService) pull(context *Context, request *VcPullRequest) (*VcInfo, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd  %v", target.DirectoryPath()),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	switch target.Type {
	case "git":
		return s.gitService.pull(context, request)
	case "svn":
		return s.svnService.pull(context, request)
	}
	return nil, fmt.Errorf("unsupported type: %v for URL %v", target.Type, target.URL)
}

//checkout If target directory exist and already contains matching origin URL, only taking the latest changes without overriding local if performed, otherwise full checkout
func (s *versionControlService) checkout(context *Context, request *VcCheckoutRequest) (*VcCheckoutResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
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
		info, err := s.checkoutArtifact(context, moduleOrigin, targetModule, request.RemoveLocalChanges)
		if err != nil {
			return nil, err
		}
		response.Checkouts[moduleOrigin.URL] = info
	}
	return response, nil
}

func (s *versionControlService) checkoutArtifact(context *Context, origin, target *url.Resource, removeLocalChanges bool) (*VcInfo, error) {
	var directoryPath = target.DirectoryPath()
	var parent, _ = path.Split(directoryPath)

	_, err := context.Execute(target, fmt.Sprintf("mkdir -p %v", parent))
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, fmt.Sprintf("cd %v", parent))
	if err != nil {
		return nil, err
	}

	storageService, err := storage.NewServiceForURL(target.URL, target.Credential)
	if err != nil {
		return nil, err
	}
	exists, err := storageService.Exists(target.URL)
	if err != nil {
		return nil, err
	}

	if exists {
		response, err := s.checkInfo(context, &VcStatusRequest{Target: target})
		if err != nil {
			return nil, err
		}

		var originURLResource = url.NewResource(origin.URL)
		var actualURLResource = url.NewResource(response.Origin)
		originPath := originURLResource.ParsedURL.Hostname() + originURLResource.DirectoryPath()
		actualPath := actualURLResource.ParsedURL.Hostname() + originURLResource.DirectoryPath()
		if originPath == actualPath {
			s.pull(context, &VcPullRequest{
				Origin: origin,
				Target: target,
			})
			return response, nil
		}

		if removeLocalChanges {
			_, err = context.Execute(target, &ManagedCommand{
				Executions: []*Execution{
					{
						Command: fmt.Sprintf("rm -rf %v",directoryPath),
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


	_, err = context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("mkdir -p %v", parent),
			},
			{
				Command: fmt.Sprintf("cd %v", parent),
			},
		},
	})
	if err != nil {
		return nil, err
	}


	switch origin.Type {
	case "git":
		return s.gitService.checkout(context, &VcCheckoutRequest{
			Origin: origin,
			Target: target,
		})
	case "svn":
		return s.svnService.checkout(context, &VcCheckoutRequest{
			Origin: origin,
			Target: target,
		})

	default:
		return nil, fmt.Errorf("unsupported version control type: '%v'", target.Type)
	}
}

func (s *versionControlService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))

	var err error
	switch actualRequest := request.(type) {
	case *VcStatusRequest:
		response.Response, err = s.checkInfo(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to check version: %v(%v), %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *VcCheckoutRequest:
		response.Response, err = s.checkout(context, actualRequest)

		if err != nil {
			response.Error = fmt.Sprintf("failed to checkout version: %v -> %v, %v", actualRequest.Origin.URL, actualRequest.Target.URL, err)
		}

	case *VcCommitRequest:
		response.Response, err = s.commit(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to commit version: %v(%v), %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}
	case *VcPullRequest:
		response.Response, err = s.pull(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to commit version: %v -> %v, %v", actualRequest.Origin.URL, actualRequest.Target.URL, err)
		}
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)

	}

	if response.Error != "" {
		response.Status = "err"
	}
	return response
}


var errorRewrites = map[string]func(*url.Resource) string {
	"authentication failed":  func(resource *url.Resource) string{
		username, _, _ := resource.LoadCredential(false)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret", username, resource.Credential)
	},
	"error validating server certificate":func(resource *url.Resource) string{
		return fmt.Sprintf("failed to validate svn certificate: %v", resource.URL)
	},
	"username":func(resource *url.Resource) string{
		username, _, _ := resource.LoadCredential(false)
		return fmt.Sprintf("failed to authenticate username: %v with %v secret",  username, resource.Credential)
	},
}

func checkVersionControlAuthErrors(err error, resource *url.Resource) error {
	if err != nil {
		errorMessage := strings.ToLower(err.Error())
		for candidate, callback := range errorRewrites {
			if strings.Contains(errorMessage, candidate) {
				return errors.New(callback(resource))
			}
		}
	}
	return err
}


func (s *versionControlService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "status":
		return &VcStatusRequest{}, nil
	case "checkout":
		return &VcCheckoutRequest{}, nil
	case "commit":
		return &VcCommitRequest{}, nil
	case "pull":
		return &VcPullRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewVersionControlService creates a new version control
func NewVersionControlService() Service {
	var result = &versionControlService{
		AbstractService: NewAbstractService(VersionControlServiceID),
		gitService:      &gitService{},
		svnService:      &svnService{},
	}
	result.AbstractService.Service = result
	return result
}
