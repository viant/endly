package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//VcCheckoutRequest represents checkout request. If target directory exist and contains matching origin URL,
// only taking the latest changes without overriding local if performed, otherwise full checkout
type VcCheckoutRequest struct {
	Type               string        `required:"true" description:"version control type: git, svn"`
	Target             *url.Resource `required:"true" description:"checkout source  defined by host and path URL"`
	Origin             *url.Resource `required:"true"`
	Modules            []string      `description:"list of modules to checkout"`
	RemoveLocalChanges bool          `description:"flat to remove local directory before checkout"`
}

//VcCheckoutResponse represents checkout response
type VcCheckoutResponse struct {
	Checkouts map[string]*VcInfo
}

//Init initializes request
func (r *VcCheckoutRequest) Init() error {
	versionControlRequestInit(r.Origin, &r.Type)
	return nil
}

//Validate validates request
func (r *VcCheckoutRequest) Validate() error {

	if r.Origin == nil {
		return fmt.Errorf("origin type was empty")
	}
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("version control type was empty for %v", r.Origin.URL)
	}
	return nil
}
