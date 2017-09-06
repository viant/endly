package vc

import (
	"fmt"
	"github.com/viant/toolbox/storage"
	"github.com/viant/endly"
	"path"
)

var VersionControlServiceId = "VcServiceId"

type service struct {
	*endly.AbstractService
	*gitService
	*svnService
}

type CheckoutRequest struct {
	Origin *endly.Resource
	Target *endly.Resource
}

func (r *CheckoutRequest) Validate() error {
	return nil
}

type CommitRequest struct {
	Target  *endly.Resource
	Message string
}

type StatusRequest struct {
	Target *endly.Resource
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

func (s *service) checkInfo(context *endly.Context, request *StatusRequest) (*InfoResponse, error) {
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

func (s *service) commit(context *endly.Context, request *CommitRequest) (interface{}, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	_, err = context.Execute(target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
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


func (s *service) checkOut(context *endly.Context, request *CheckoutRequest) (interface{}, error) {
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
		if origin.URL == response.Origin && response.IsUptoDate && ! response.HasPendingChanges() {
			return response, nil
		}

		_, err = context.Execute(target, &endly.ManagedCommand{
			Executions: []*endly.Execution{
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
	_, err = context.Execute(target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("mkdir -p %v", parent),
			},
			{
				Command: fmt.Sprintf("cd  %v", parent),
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

func (s *service) Run(context *endly.Context, request interface{}) *endly.Response {
	var response = &endly.Response{Status: "ok"}

	switch actualRequest := request.(type) {
	case *StatusRequest:
		response.Response, response.Error = s.checkInfo(context, actualRequest)

	case *CheckoutRequest:
		response.Response, response.Error = s.checkOut(context, actualRequest)

	case *CommitRequest:
		response.Response, response.Error = s.commit(context, actualRequest)
	}

	if response.Error != nil {
		response.Status = "err"
	}
	return response
}

func (s *service) NewRequest(name string) (interface{}, error) {
	switch name {
	case "command":
		return &endly.ScriptCommand{}, nil
	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
}

func NewVersionControlService() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(VersionControlServiceId),
		gitService:      &gitService{},
		svnService:      &svnService{},
	}
	result.AbstractService.Service = result
	return result
}

func init() {
	endly.NewServiceManager().Register(NewVersionControlService())
}
