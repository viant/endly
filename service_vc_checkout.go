package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
	"strings"
)

//VcCheckoutRequest represents checkout request. If target directory exist and contains matching origin URL,
// only taking the latest changes without overriding local if performed, otherwise full checkout
type VcCheckoutRequest struct {
	Origin             *url.Resource //version control origin
	Target             *url.Resource //local code destination
	Modules            []string      //vc path to project
	RemoveLocalChanges bool          //flag to remove local changes
}

//VcCheckoutResponse represents checkout response
type VcCheckoutResponse struct {
	Checkouts map[string]*VcInfo
}

//Validate validates request
func (r *VcCheckoutRequest) Validate() error {

	if r.Origin == nil {
		return fmt.Errorf("Origin type was empty")
	}
	if r.Target == nil {
		return fmt.Errorf("Target type was empty")
	}

	if r.Origin.Type == "" {
		if strings.Contains(r.Origin.URL, "/svn/") {
			r.Origin.Type = "svn"
		} else if strings.Contains(r.Origin.URL, "git") {
			r.Origin.Type = "git"
		} else {
			return fmt.Errorf("Origin type was empty for %v", r.Origin.URL)
		}
	}
	if r.Target.Type == "" {
		r.Target.Type = r.Origin.Type
	}
	return nil
}
