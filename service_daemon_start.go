package endly

import (
	"github.com/viant/toolbox/url"
)

//DaemonStartRequest represents service request start
type DaemonStartRequest struct {
	Target    *url.Resource //target host
	Service   string        //service name
	Exclusion string        //exclusion if there is more than one service matching service group
}
