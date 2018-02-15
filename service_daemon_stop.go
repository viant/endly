package endly

import (
	"github.com/viant/toolbox/url"
)

//DaemonStopRequest represents a stop request.
type DaemonStopRequest struct {
	Target    *url.Resource `required:"true" description:"target host"` //target host
	Service   string        `required:"true"`                           //service name
	Exclusion string        //exclusion if there is more than one service matching service group
}

//DaemonStopResponse represents a stop response
type DaemonStopResponse struct {
	*DaemonInfo
}
