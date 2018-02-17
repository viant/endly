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

//ProcessStopAllResponse represents a stop all response
type ProcessStopAllResponse struct {
	Stdout string
}

//ProcessStopResponse represents a stop response
type ProcessStopResponse struct {
	Stdout string
}
