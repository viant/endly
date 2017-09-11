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

type CheckoutRequest struct {
	Origin *Resource
	Target *Resource
}

func (r *CheckoutRequest) Validate() error {
	return nil
}

type CommitRequest struct {
	Target  *Resource
	Message string
}

type StatusRequest struct {
	Target *Resource
}

type InfoResponse struct {
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

func (r *InfoResponse) HasPendingChanges() bool {
	return len(r.New) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}

func (s *versionControlService) checkInfo(context *Context, request *StatusRequest) (*InfoResponse, error) {
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

func (s *versionControlService) commit(context *Context, request *CommitRequest) (interface{}, error) {
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

func (s *versionControlService) checkOut(context *Context, request *CheckoutRequest) (interface{}, error) {
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
		response, err := s.checkInfo(context, &StatusRequest{Target: request.Target})
		if err != nil {
			return nil, err
		}
		if origin.URL == response.Origin && response.IsUptoDate && !response.HasPendingChanges() {
			return response, nil
		}

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
	case *StatusRequest:
		response.Response, err = s.checkInfo(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to check version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *CheckoutRequest:
		response.Response, err = s.checkOut(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to checkout version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	case *CommitRequest:
		response.Response, err = s.commit(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to commit version: %vL%v, %v", actualRequest.Target.URL, actualRequest.Target.Type, err)
		}

	}

	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *versionControlService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "command":
		return &ScriptCommand{}, nil
	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
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
