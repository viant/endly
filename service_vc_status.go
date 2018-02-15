package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//VcStatusRequest represents version control status
type VcStatusRequest struct {
	Target *url.Resource `required:"true"`
	Type   string
}

// /Init initializes request
func (r *VcStatusRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *VcStatusRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}

//VcStatusResponse represents version control status response
type VcStatusResponse struct {
	*VcInfo
}
