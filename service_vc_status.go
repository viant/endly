package endly

import (
	"github.com/viant/toolbox/url"
)


//VcStatusRequest represents version control status
type VcStatusRequest struct {
	Target *url.Resource//local code source repo
}
