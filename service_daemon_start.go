package endly

import (
	"github.com/viant/toolbox/url"
)

//DaemonStartRequest represents service request start
type DaemonStartRequest struct {
	Target    *url.Resource `required:"true" description:"target host"`                                                                //target host
	Service   string        `required:"true" `                                                                                         //service name
	Exclusion string        `description:"optional exclusion fragment in case there are more then one matching provided name service"` //exclusion if there is more than one service matching service group
}

//DaemonStartResponse represents daemon start response
type DaemonStartResponse struct {
	*DaemonInfo
}
