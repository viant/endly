package endly

import "github.com/viant/toolbox/url"

//ProcessStopRequest represents a stop request
type ProcessStopRequest struct {
	Target *url.Resource
	Pid    int
}

//ProcessStopAllRequest represents a stop all processes matching provided name request
type ProcessStopAllRequest struct {
	Target *url.Resource
	Input  string
}
