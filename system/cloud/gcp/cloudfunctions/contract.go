package cloudfunctions

import (
	"errors"
	"github.com/viant/toolbox/url"
	"google.golang.org/api/cloudfunctions/v1"
	"strings"
)

// CallRequest represents a call request
type CallRequest struct {
	Name   string
	Region string
	Data   interface{}
}

// DeployRequest represents deploy request
type DeployRequest struct {
	cloudfunctions.CloudFunction `yaml:",inline" json:",inline"`
	Source                       *url.Resource
	Public                       bool     `description:"set this flag to make function public"`
	Members                      []string `description:"members with roles/cloudfunctions.invoker role"`
	Region                       string
	Retry                        bool
}

// DeployResponse represents deploy response
type DeployResponse struct {
	Operation string
	Meta      interface{}
	Function  *cloudfunctions.CloudFunction
}

// GetRequest represents get function requests
type GetRequest struct {
	Name   string
	Region string
}

// GetRequest represents list function requests
type ListRequest struct {
	Region string
}

// GetRequest represents list function response
type ListResponse struct {
	Function []*cloudfunctions.CloudFunction
}

// DeleteRequest represents delete function requests
type DeleteRequest struct {
	Name   string
	Region string
}

// DeleteResponse represents delete response
type DeleteResponse struct {
	Operation string
	Meta      interface{}
}

// Init initializes request
func (r *ListRequest) Init() error {
	r.Region = initRegion(r.Region)
	return nil
}

// Init initializes request
func (r *CallRequest) Init() error {
	r.Name = initFullyQualifiedName(r.Name)
	r.Region = initRegion(r.Region)
	return nil
}

// Validate checks if request was valid
func (r *CallRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name was empty")
	}
	return nil
}

// Validate checks if request was valid
func (r *DeployRequest) Validate() error {
	if r.CloudFunction.Name == "" {
		return errors.New("name was empty")
	}
	if r.Runtime == "" {
		return errors.New("runtime was empty")
	}
	return nil
}

// Init initializes request
func (r *DeployRequest) Init() error {
	if r.Region == "" {
		r.Region = defaultRegion
	}
	if r.CloudFunction.Name == "" {
		return nil
	}
	r.Name = initFullyQualifiedName(r.Name)
	r.Region = initRegion(r.Region)
	if len(r.Labels) == 0 {
		r.Labels = make(map[string]string)
	}
	if r.HttpsTrigger == nil && r.EventTrigger == nil {
		r.HttpsTrigger = &cloudfunctions.HttpsTrigger{}
	}

	if r.EntryPoint == "" {
		fragments := strings.Split(r.Name, "/")
		if len(fragments) > 0 {
			r.EntryPoint = fragments[len(fragments)-1]
		}
	}
	if r.Public {
		r.Members = []string{"allUsers"}
	}
	r.Labels["deployment-tool"] = "endly"

	if r.Retry && r.EventTrigger != nil {
		r.EventTrigger.FailurePolicy = &cloudfunctions.FailurePolicy{
			Retry: &cloudfunctions.Retry{},
		}
	}
	return r.Source.Init()
}

// Init initializes request
func (r *GetRequest) Init() error {
	r.Name = initFullyQualifiedName(r.Name)
	r.Region = initRegion(r.Region)
	return nil
}

// Validate checks if request was valid
func (r *GetRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name was empty")
	}
	return nil
}

// Init initializes request
func (r *DeleteRequest) Init() error {
	r.Name = initFullyQualifiedName(r.Name)
	r.Region = initRegion(r.Region)
	return nil
}

// Validate checks if request was valid
func (r *DeleteRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name was empty")
	}
	return nil
}
