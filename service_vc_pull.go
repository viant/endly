package endly

import (
	"github.com/viant/toolbox/url"
)
//VcPullRequest represents a pull request
type VcPullRequest struct {
	Target *url.Resource//local code destination
	Origin *url.Resource//version control origin
}

