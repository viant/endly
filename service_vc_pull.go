package endly

import (
	"github.com/viant/toolbox/url"
	"errors"
)

//VcPullRequest represents a pull request
type VcPullRequest struct {
	Target *url.Resource //local code destination
	Origin *url.Resource //version control origin
}


func (r *VcPullRequest) Validate() error {
	if r.Origin == nil {
		return errors.New("Origin was empty")
	}
	if r.Target == nil {
		return errors.New("Target was empty")
	}
	return nil
}