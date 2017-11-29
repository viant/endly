package endly

import (
	"github.com/viant/toolbox/url"
	"strings"
)

//DaemonStatusRequest represents status request
type DaemonStatusRequest struct {
	Target    *url.Resource //target host
	Service   string        //service name
	Exclusion string        //exclusion if there is more than one service matching service group
}

//DaemonInfo represents a service info
type DaemonInfo struct {
	Service string //requested service name
	Path    string //path
	Pid     int    //process if
	Type    int    //type
	Domain    string //command how service was launched
	State   string //state
	Launched  bool
}

//IsActive returns true if service is running
func (s *DaemonInfo) IsActive() bool {
	return strings.ToLower(s.State) == "running"
}
