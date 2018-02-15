package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//VcPullRequest represents a pull request
type VcPullRequest struct {
	Type   string
	Target *url.Resource `required:"true"`
	Origin *url.Resource `required:"true"` //version control origin
}

//Init initializes request
func (r *VcPullRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *VcPullRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}

//VcPullResponse represents a pull response
type VcPullResponse struct {
	*VcInfo
}
