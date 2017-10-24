package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"path"
	"strings"
	"github.com/viant/toolbox/url"
)

//VersionControlServiceID version control service id
var VersionControlServiceID = "versionControl"

type versionControlService struct {
	*AbstractService
	*gitService
	*svnService
}

//VcCheckoutRequest represents checkout request. If target directory exist and contains matching origin URL,
// only taking the latest changes without overriding local if performed, otherwise full checkout
type VcCheckoutRequest struct {
	Origin             *url.Resource//version control origin
	Target             *url.Resource//local code destination
	Modules            []string //vc path to project
	RemoveLocalChanges bool//flag to remove local changes
}

//VcCheckoutResponse represents checkout response
type VcCheckoutResponse struct {
	Checkouts map[string]*VcInfo
}

//Validate validates request
func (r *VcCheckoutRequest) Validate() error {

	if r.Origin == nil {
		return fmt.Errorf("Origin type was empty")
	}
	if r.Target == nil {
		return fmt.Errorf("Target type was empty")
	}

	if r.Origin.Type == "" {
		if strings.Contains(r.Origin.URL, "/svn/") {
			r.Origin.Type = "svn"
		} else if strings.Contains(r.Origin.URL, "git") {
			r.Origin.Type = "git"
		} else {
			return fmt.Errorf("Origin type was empty for %v", r.Origin.URL)
		}
	}
	if r.Target.Type == "" {
		r.Target.Type = r.Origin.Type
	}
	return nil
}

//VcPullRequest represents a pull request
type VcPullRequest struct {
	Target *url.Resource//local code destination
	Origin *url.Resource//version control origin
}

//VcCommitRequest represents a commit request
type VcCommitRequest struct {
	Target  *url.Resource //local code source repo
	Message string//commit message
}

//VcStatusRequest represents version control status
type VcStatusRequest struct {
	Target *url.Resource//local code source repo
}

//VcInfo represents version control info
type VcInfo struct {
	IsVersionControlManaged bool
	Origin                  string
	Revision                string
	Branch                  string
	IsUptoDate              bool
	New                     []string
	Untracked               []string
	Modified                []string
	Deleted                 []string
}

//HasPendingChanges returns true if there are any untracked, new, modified, deleted files.
func (r *VcInfo) HasPendingChanges() bool {
	return len(r.New) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}

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
	return nil, fmt.Errorf("Unsupported type: %v -> ", target.Type, target.URL)
}

func (s *versionControlService) commit(context *Context, request *VcCommitRequest) (interface{}, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd  %v", target.ParsedURL.Path),
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
	return nil, fmt.Errorf("Unsupported type: %v -> %v", target.Type, target.URL)
}

func (s *versionControlService) pull(context *Context, request *VcPullRequest) (*VcInfo, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd  %v", target.ParsedURL.Path),
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
	return nil, fmt.Errorf("Unsupported type: %v -> %v", target.Type, target.URL)
}

func (s *versionControlService) checkOut(context *Context, request *VcCheckoutRequest) (*VcCheckoutResponse, error) {
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
	var parent, _ = path.Split(target.ParsedURL.Path)
	context.Execute(target, fmt.Sprintf("cd %v", parent))
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
		if origin.URL == response.Origin {
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
						Command: fmt.Sprintf("rm -rf %v", target.ParsedURL.Path),
					},
				},
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("Directory containst different version: %v at rev: %v", response.Origin, response.Revision)
		}
	}

	parent, _ = path.Split(target.ParsedURL.Path)
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
		return nil, fmt.Errorf("Unsupproted version control type: '%v'", target.Type)
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
			response.Error = fmt.Sprintf("Failed to check version: %v(%v), %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *VcCheckoutRequest:
		response.Response, err = s.checkOut(context, actualRequest)

		if err != nil {
			response.Error = fmt.Sprintf("Failed to checkout version: %v -> %v, %v", actualRequest.Origin.URL, actualRequest.Target.URL, err)
		}

	case *VcCommitRequest:
		response.Response, err = s.commit(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to commit version: %v(%v), %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}
	case *VcPullRequest:
		response.Response, err = s.pull(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to commit version: %v -> %v, %v", actualRequest.Origin.URL, actualRequest.Target.URL, err)
		}
	}

	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *versionControlService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "status":
		return &VcStatusRequest{}, nil
	case "checkout":
		return &VcCheckoutRequest{}, nil
	case "module_checkout":
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
