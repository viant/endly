package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerLogoutRequest represents a docker pull request
type DockerLogoutRequest struct {
	SysPath    []string
	Target     *url.Resource
	Repository string
}

//DockerLogoutResponse represents a docker pull request
type DockerLogoutResponse struct {
	Stdout string
}
