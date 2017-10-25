package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerPullRequest represents a docker pull request
type DockerPullRequest struct {
	Target     *url.Resource
	Repository string
	Tag        string
}
