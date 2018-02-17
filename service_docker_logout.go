package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerLogoutRequest represents a docker pull request
type DockerLogoutRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true" description:"repository URL"`
}

//DockerLogoutResponse represents a docker pull request
type DockerLogoutResponse struct {
	Stdout string
}
