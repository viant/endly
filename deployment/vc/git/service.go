package git

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"gopkg.in/src-d/go-git.v4"
	"os"
)

const ServiceID = "vc/git"

type service struct {
	*endly.AbstractService
}

func (s *service) clone(context *endly.Context, request *CheckoutRequest) (*git.Repository, error) {
	options := &git.CloneOptions{
		URL:      request.Origin.URL,
		Progress: os.Stdout,
		Depth:    request.Depth,
	}
	var err error
	if request.Origin.Credentials != "" {
		if options.Auth, err = getAuth(context, request.Origin.Credentials); err != nil {
			return nil, err
		}
	}
	return git.PlainClone(request.Dest.ParsedURL.Path, false, options)

}

func (s *service) checkout(context *endly.Context, request *CheckoutRequest) (*CheckoutResponse, error) {

	var err error
	destFile := request.Dest.ParsedURL.Path
	repository, err := git.PlainOpen(destFile)
	freshCheckout := err != nil

	if !freshCheckout && !matchesOrigin(repository, request.Origin) {
		return nil, fmt.Errorf("local not match remote repository: %v", request.Dest.URL)
	}
	if freshCheckout {
		if repository, err = s.clone(context, request); err != nil {
			return nil, err
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree, %v", err)
	}

	pullOptions := &git.PullOptions{RemoteName: "origin", Progress: os.Stdout}
	if request.Origin.Credentials != "" {
		if pullOptions.Auth, err = getAuth(context, request.Origin.Credentials); err != nil {
			return nil, err
		}
	}
	if err = worktree.Pull(pullOptions); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return nil, err
		}
	}
	statusResponse, err := s.status(context, &StatusRequest{Source: request.Dest})
	if err != nil {
		return nil, err
	}
	response := CheckoutResponse(*statusResponse)
	return &response, err
}

func (s *service) status(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	response := &StatusResponse{NewInfo()}
	destFile := request.Source.ParsedURL.Path
	repository, err := git.PlainOpen(destFile)
	if err != nil {
		response.IsVersionControlManaged = false
		return response, nil
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree, %v", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status, %v", err)
	}

	config, err := repository.Config()
	if err != nil {
		return nil, err
	}
	if origin, has := config.Remotes["origin"]; has {
		if len(origin.URLs) > 0 {
			response.Origin = origin.URLs[0]
		}
	}
	if head, err := repository.Head(); err == nil {
		response.Revision = head.Hash().String()
	}
	response.IsUptoDate = status.IsClean()
	for k, v := range status {
		switch v.Staging {
		case git.Untracked:
			response.Untracked = append(response.Untracked, k)
		case git.Added:
			response.Added = append(response.Added, k)
		case git.Modified, git.Renamed:
			response.Modified = append(response.Modified, k)
		case git.Deleted:
			response.Deleted = append(response.Deleted, k)
		}
	}
	return response, nil
}

func (s *service) registerRoutes() {

	//xx action route
	s.Register(&endly.Route{
		Action: "checkout",
		RequestInfo: &endly.ActionInfo{
			Description: "checkout checkout origin into dest",
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
		Action: "status",
		RequestInfo: &endly.ActionInfo{
			Description: "status get repo status",
		},
		RequestProvider: func() interface{} {
			return &StatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StatusRequest); ok {
				response, err := s.status(context, req)
				if err == nil {
					context.Publish(&OutputEvent{msg.NewOutputEvent("", "git", response)})

				}
				return response, err

			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
