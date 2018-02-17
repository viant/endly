package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DockerLoginRequest represents a docker pull request
type DockerLoginRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Credential string        `required:"true" description:"credential path"`
	Repository string        `required:"true" description:"repository url"`
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
