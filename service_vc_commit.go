package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//VcCommitRequest represents a commit request
type VcCommitRequest struct {
	Target  *url.Resource `required:"true" description:"location to local source code"`
	Type    string        `description:"version control type: git,svn"`
	Message string        `required:"true"`
}

//VcCommitResponse represents a commit response
type VcCommitResponse struct {
	*VcInfo
}

//Init initializes request
func (r *VcCommitRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *VcCommitRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}
