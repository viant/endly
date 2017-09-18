package endly

import (
	"fmt"
	"github.com/viant/toolbox/storage"
	"path"
)

var VersionControlServiceId = "VcServiceId"

type versionControlService struct {
	*AbstractService
	*gitService
	*svnService
}

type VcCheckoutRequest struct {
	Origin             *Resource
	Target             *Resource
	RemoveLocalChanges bool
}

func (r *VcCheckoutRequest) Validate() error {
	return nil
}

type VcPullRequest struct {
	Target *Resource
	Origin *Resource
}

type VcCommitRequest struct {
	Target  *Resource
	Message string
}

type VcStatusRequest struct {
	Target *Resource
}

type VcInfoResponse struct {
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

func (r *VcInfoResponse) HasPendingChanges() bool {
	return len(r.New) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}

func (s *versionControlService) checkInfo(context *Context, request *VcStatusRequest) (*VcInfoResponse, error) {
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

func (s *versionControlService) pull(context *Context, request *VcPullRequest) (*VcInfoResponse, error) {
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

func (s *versionControlService) checkOut(context *Context, request *VcCheckoutRequest) (*VcInfoResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	storageService, err := storage.NewServiceForURL(target.URL, target.CredentialFile)
	if err != nil {
		return nil, err
	}

	exists, err := storageService.Exists(target.URL)
	if err != nil {
		return nil, err
	}

	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}

	if exists {
		response, err := s.checkInfo(context, &VcStatusRequest{Target: request.Target})
		if err != nil {
			return nil, err
		}
		if origin.URL == response.Origin {
			if response.IsUptoDate {
				return response, nil
			}

			return s.pull(context, &VcPullRequest{
				Target: request.Target,
				Origin: request.Origin,
			})
		}

		if request.RemoveLocalChanges {
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

	parent, _ := path.Split(target.ParsedURL.Path)
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

	switch target.Type {
	case "git":
		return s.gitService.checkout(context, request)
	case "svn":
		return s.svnService.checkout(context, request)
	}
	return nil, nil
}

func (s *versionControlService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}

	var err error
	switch actualRequest := request.(type) {
	case *VcStatusRequest:
		response.Response, err = s.checkInfo(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to check version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *VcCheckoutRequest:
		response.Response, err = s.checkOut(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to checkout version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *VcCommitRequest:
		response.Response, err = s.commit(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to commit version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}
	case *VcPullRequest:
		response.Response, err = s.pull(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to commit version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
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
	case "commit":
		return &VcCommitRequest{}, nil
	case "pull":
		return &VcPullRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewVersionControlService() Service {
	var result = &versionControlService{
		AbstractService: NewAbstractService(VersionControlServiceId),
		gitService:      &gitService{},
		svnService:      &svnService{},
	}
	result.AbstractService.Service = result
	return result
}
