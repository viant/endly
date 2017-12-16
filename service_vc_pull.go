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

//Validate checks if request is valud
func (r *VcPullRequest) Validate() error {
	if r.Origin == nil {
		return errors.New("origin was empty")
	}
	if r.Target == nil {
		return errors.New("target was empty")
	}
	return nil
}