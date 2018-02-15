package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerPullRequest represents a docker pull request
type DockerPullRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true"`
	Tag        string        `required:"true"`
}

//DockerPullRequest represents a docker pull response
type DockerPullResponse struct {
	*DockerImageInfo
}