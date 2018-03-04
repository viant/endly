package daemon

import (
	"github.com/viant/toolbox/url"
	"strings"
)

//StartRequest represents service request start
type StartRequest struct {
	Target    *url.Resource `required:"true" description:"target host"`                                                                //target host
	Service   string        `required:"true" `                                                                                         //service name
	Exclusion string        `description:"optional exclusion fragment in case there are more then one matching provided name service"` //exclusion if there is more than one service matching service group
}

//StartResponse represents daemon start response
type StartResponse struct {
	*Info
}

//StatusRequest represents status request
type StatusRequest struct {
	Target    *url.Resource `required:"true" description:"target host"` //target host
	Service   string        `required:"true" `                          //service name
	Exclusion string        //exclusion if there is more than one service matching service group
}

//StatusResponse represent status response
type StatusResponse struct {
	*Info
}

//Info represents a service info
type Info struct {
	Service  string //requested service name
	Path     string //path
	Pid      int    //process if
	Type     int    //type
	Domain   string //command how service was launched
	State    string //state
	Launched bool
}

//StopRequest represents a stop request.
type StopRequest struct {
	Target    *url.Resource `required:"true" description:"target host"` //target host
	Service   string        `required:"true"`                           //service name
	Exclusion string        //exclusion if there is more than one service matching service group
}

//StopResponse represents a stop response
type StopResponse struct {
	*Info
}

//IsActive returns true if service is running
func (s *Info) IsActive() bool {
	return strings.ToLower(s.State) == "running"
}
