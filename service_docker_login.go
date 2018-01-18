package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DockerLoginRequest represents a docker pull request
type DockerLoginRequest struct {
	SysPath    []string
	Target     *url.Resource
	Credential string
	Repository string
}

//DockerLoginResponse represents a docker pull request
type DockerLoginResponse struct {
	Stdout   string
	Username string
}

//Validate checks if request is valid
func (r *DockerLoginRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Repository == "" {
		return errors.New("repository was empty")
	}
	return nil
}
